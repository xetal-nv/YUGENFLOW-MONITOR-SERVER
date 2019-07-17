$(document).ready(function () {
    document.getElementById("loader").style.visibility = "hidden";
    Date.prototype.getUnixTime = function () {
        return (this.getTime() / 1000 | 0) * 1000
    };
    var maxtries,
        startDate,
        endDate,
        updateStartDate = function () {
            startPicker.setStartRange(startDate);
            endPicker.setStartRange(startDate);
            endPicker.setMinDate(startDate);
        },
        updateEndDate = function () {
            startPicker.setEndRange(endDate);
            startPicker.setMaxDate(endDate);
            endPicker.setEndRange(endDate);
        },
        startPicker = new Pikaday({
            field: document.getElementById('start'),
            minDate: new Date(StartDat),
            maxDate: new Date(),
            onSelect: function () {
                startDate = this.getDate();
                updateStartDate();
            }
        }),
        endPicker = new Pikaday({
            field: document.getElementById('end'),
            minDate: new Date(StartDat),
            maxDate: new Date(),
            onSelect: function () {
                endDate = this.getDate();
                updateEndDate();
            }
        }),
        _startDate = startPicker.getDate(),
        _endDate = endPicker.getDate();

    maxtries = 10;


    if (_startDate) {
        startDate = _startDate;
        updateStartDate();
    }

    if (_endDate) {
        endDate = _endDate;
        updateEndDate();
    }

    document.getElementById("gen").addEventListener("click", generateReport);

    function generateReport() {

        // function sortentryEl0(a, b) {
        //
        //     if (a[0] < b[0]) return -1;
        //     if (a[0] > b[0]) return 1;
        //     return 0;
        // }

        // function exportreport(header, entrieslist, sampledata, entrydata) {
        function exportreport(header, sampledata) {
            let data = header,
                rawdataSample = [],
                finalData = {};
            data += "\n";
            if (sampledata !== null) {
                for (let i = 0; i < sampledata.length; i++) {
                    if ((sampledata[i]["ts"] !== "") && (sampledata[i]["val"] !== "")) {
                        rawdataSample.push([sampledata[i]["ts"], sampledata[i]["val"]])
                    }
                }

                // Find the minimum interval in changes, normally this is the measurement step
                // TODO to be replaced with with dynamically constructed js from conf files
                let tslist = [];
                let tsstep = -1;
                while (rawdataSample.length > 0) {
                    let sam = rawdataSample.shift();
                    tslist.push(sam[0]);
                    if (tsstep < 0) {
                        tsstep += 1
                    } else {
                        let cds = tslist[tslist.length - 1] - tslist[tslist.length - 2];
                        if (tsstep === 0) {
                            tsstep = cds
                        } else {
                            tsstep = Math.min(tsstep, cds)
                        }
                    }

                    finalData[sam[0]] = sam[1];
                }

                for (let i = 0; i < tslist.length; i++) {
                    let d = new Date(tslist[i]);
                    var datestring = ("0" + d.getDate()).slice(-2) + "-" + ("0" + (d.getMonth() + 1)).slice(-2) + "-" +
                        d.getFullYear() + " " + ("0" + d.getHours()).slice(-2) + ":" + ("0" + d.getMinutes()).slice(-2);
                    data += datestring + ", " + Math.trunc(tslist[i] / 1000)
                        + ", " + finalData[tslist[i]];
                    data += "\n"
                }

                var blob = new Blob([data], {type: 'text/plain'}),
                    anchor = document.createElement('a');
                anchor.download = space + "_" + asys + ".csv";
                anchor.href = (window.webkitURL || window.URL).createObjectURL(blob);
                anchor.dataset.downloadurl = ['text/plain', anchor.download, anchor.href].join(':');
                anchor.click();
            } else {
                alert("No data available for the selected time.");
            }

            document.getElementById("loader").style.visibility = "hidden";
        }


        // function loadsamples(header, api, entrieslist, tries) {
        function loadsamples(header, api, tries) {
            $.ajax({
                type: 'GET',
                timeout: 5000,
                url: ip + "/series?type=sample?space=" + api,
                success: function (rawdata) {
                    let sampledata = JSON.parse(rawdata);
                    // console.log(sampledata)
                    exportreport(header, sampledata);
                },
                error: function (error) {
                    if (tries === maxtries) {
                        alert("Server or network error.\n Please try again later.");
                        console.log("Error samples:" + error);
                        document.getElementById("loader").style.visibility = "hidden";
                    } else {
                        // loadsamples(header, api, entrieslist, tries + 1)
                        loadsamples(header, api, tries + 1)
                    }
                }

            });
        }

        function start_report(header, path, tries) {
            $.ajax({
                type: 'GET',
                timeout: 2000,
                url: ip + "/info",
                success: function (rawdata) {
                    let info = JSON.parse(rawdata);
                    // noinspection JSMismatchedCollectionQueryUpdate
                    let currentplan = []; // only used as a check of integrity
                    for (let i = 0; i < info.length; i++) {
                        if (info[i]["spacename"] === space) {
                            for (let j = 0; j < info[i]["entries"].length; j++) {
                                currentplan.push([info[i]["entries"][j]["entryid"], 0])
                            }
                        }
                    }
                    if (currentplan !== []) {
                        // currentplan.sort(sortentryEl0);
                        // loadsamples(header, path, currentplan, 0)
                        loadsamples(header, path, 0)
                    } else {
                        alert("Server connection lost.\n Please try again later.");
                        document.getElementById("loader").style.visibility = "hidden";
                    }
                },
                error: function (error) {
                    if (tries === maxtries) {
                        alert("Server connection lost.\n Please try again later.");
                        console.log("Error info:" + error);
                        document.getElementById("loader").style.visibility = "hidden";
                    } else {
                        start_report(header, path, tries + 1);
                    }
                }

            });
        }

        let select = document.getElementById("spacename");
        var myindex = select.selectedIndex,
            space = select.options[myindex].value;
        select = document.getElementById("reptype");
        myindex = select.selectedIndex;
        var asys = select.options[myindex].value,
            copyendDate = new Date(endDate),
            start, end;
        if ((startDate !== undefined) && (endDate !== undefined)
            && (space !== "Choose a space") && (asys !== "Choose a dataset")) {
            document.getElementById("loader").style.visibility = "visible";
            start = startDate.getUnixTime();
            copyendDate.setHours(endDate.getHours() + 23);
            copyendDate.setMinutes(endDate.getMinutes() + 59);
            if (copyendDate.getUnixTime() > Date.now()) {
                end = Date.now();
            } else {
                end = copyendDate.getUnixTime();
            }
            let header = "\"#Xetal Flow Monitoring: " + version + " \"\n"
                + "\"#space: " + space + " \"\n"
                + "\"#dataset: " + asys + " \"\n"
                + "\"#start: " + startDate + " \"\n"
                + "\"#end: " + copyendDate + " \"\n\n";
            header += "Date/Time, Epoch Time (s), average presence";
            let path = space + "?analysis=" + asys + "?start=" + start + "?end=" + end;

            start_report(header, path, 0);


        }
    }
});