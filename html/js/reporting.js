$(document).ready(function () {
    document.getElementById("loader").style.visibility = "hidden";
    Date.prototype.getUnixTime = function () {
        return (this.getTime() / 1000 | 0) * 1000
    };
    var startDate,
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

        function sortentryEl0(a, b) {

            if (a[0] < b[0]) return -1;
            if (a[0] > b[0]) return 1;
            return 0;
        }

        function exportreport(header, entrieslist, sampledata, entrydata) {
            let data = header,
                rawdataSample = [],
                rawdataEntries = [],
                finalData = {};
            for (let i = 0; i < entrieslist.length; i++) {
                data += ", entry:" + entrieslist[i][0];
            }
            data += ", server_down\n";
            if (sampledata !== null){
                for (let i = 0; i < sampledata.length; i++) {
                    if ((sampledata[i]["ts"] !== "") && (sampledata[i]["val"] !== "")) {
                        rawdataSample.push([sampledata[i]["ts"], sampledata[i]["val"]])
                    }
                }
                rawdataSample.sort(sortentryEl0);
                for (let i = 0; i < entrydata.length; i++) {
                    if ((entrydata[i]["ts"] !== "") && (entrydata[i]["val"] !== "")) {
                        rawdataEntries.push([entrydata[i]["ts"], entrydata[i]["val"]])
                    }
                }
                rawdataEntries.sort(sortentryEl0);

                let tslist = [];
                let tsstep = -1;
                while (rawdataSample.length > 0) {
                    let sam = rawdataSample.shift(),
                        entries = entrieslist.slice();
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

                    if (rawdataEntries.length > 0) {
                        while (Math.abs(sam[0] - rawdataEntries[0][0]) < samplingWindow) {
                            let ents = rawdataEntries.shift();
                            for (let i = 0; i < ents[1].length; i++) {
                                let ent = ents[1][i];
                                for (let j = 0; j < entries.length; j++) {
                                    if (entries[j][0] === ent[0]) {
                                        entries[j][1] = ent[1];
                                        break;
                                    }
                                }
                            }
                            if (rawdataEntries.length === 0) {
                                break;
                            }
                        }
                    }
                    finalData[sam[0]] = [sam[1], []];
                    for (let i = 0; i < entries.length; i++) finalData[sam[0]][1].push([entries[i][0], entries[i][1]]);
                }
                for (let i = 0; i < tslist.length; i++) {
                    if (i > 0) {
                        if ((tslist[i] - tslist[i - 1]) > (2 * tsstep)) {
                            let diff = Math.trunc((tslist[i] - tslist[i - 1]) / 2);
                            data += new Date(tslist[i] - diff) + ", " + Math.trunc((tslist[i] - diff) / 1000)
                                + ", ";
                            for (let j = 0; j < finalData[tslist[i]][1].length; j++) {
                                data += ", ";
                            }
                            data += ", yes\n";
                        }
                    }
                    data += new Date(tslist[i]) + ", " + Math.trunc(tslist[i] / 1000)
                        + ", " + finalData[tslist[i]][0];
                    for (let j = 0; j < finalData[tslist[i]][1].length; j++) {
                        data += ", " + finalData[tslist[i]][1][j][1];
                    }
                    data += ", no\n";
                }
                var blob = new Blob([data], {type: 'text/plain'}),
                    anchor = document.createElement('a');
                anchor.download = space + "_" + asys + ".csv";
                anchor.href = (window.webkitURL || window.URL).createObjectURL(blob);
                anchor.dataset.downloadurl = ['text/plain', anchor.download, anchor.href].join(':');
                anchor.click();
            }
            document.getElementById("loader").style.visibility = "hidden";
        }

        document.getElementById("loader").style.visibility = "visible";

        function loadsamples(header, api, entrieslist) {
            $.ajax({
                type: 'GET',
                url: ip + "/series?type=sample?space=" + api,
                success: function (rawdata) {
                    let sampledata = JSON.parse(rawdata);
                    // console.log(sampledata)
                    loadEntries(header, api, entrieslist, sampledata);
                },
                error: function (error) {
                    alert("Error " + error);
                }

            });
        }

        function loadEntries(header, api, entrieslist, sampledata) {
            $.ajax({
                type: 'GET',
                url: ip + "/series?type=entry?space=" + api,
                success: function (rawdata) {
                    let entrydata = JSON.parse(rawdata);
                    exportreport(header, entrieslist, sampledata, entrydata);
                },
                error: function (error) {
                    alert("Error " + error);
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
                + "\"#end: " + copyendDate + " \"\n\n"
                + "timestamp (date), timestamp (s), counter";
            let path = space + "?analysis=" + asys + "?start=" + start + "?end=" + end;

            $.ajax({
                type: 'GET',
                url: ip + "/info",
                success: function (rawdata) {
                    let info = JSON.parse(rawdata),
                        currentplan = [];
                    for (let i = 0; i < info.length; i++) {
                        if (info[i]["spacename"] === space) {
                            for (let j = 0; j < info[i]["entries"].length; j++) {
                                currentplan.push([info[i]["entries"][j]["entryid"], 0])
                            }
                        }
                    }
                    if (currentplan !== []) {
                        currentplan.sort(sortentryEl0);
                        loadsamples(header, path, currentplan)
                    }
                },
                error: function (error) {
                    alert("Error " + error);
                }

            });
        }
    }
});