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

            let select = document.getElementById("spacename");
            var myindex = select.selectedIndex;
            if (repvisile) {
                select = document.getElementById("reptype");
                myindex = select.selectedIndex;
                var asys = select.options[myindex].value;
                if (asys === "overview") {
                    generateOverviewReport()
                } else {
                    generatePeriodicReport()
                }
            } else {
                generateOverviewReport()
            }
        }

        function generatePeriodicReport() {

            // function sortentryEl0(a, b) {
            //
            //     if (a[0] < b[0]) return -1;
            //     if (a[0] > b[0]) return 1;
            //     return 0;
            // }

            function exportReport(header, sampledata) {
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
                        try {
                        let sampledata = JSON.parse(rawdata);} catch (e) {console.log("received corrupted data: ",rawdata)}
                        // console.log(sampledata)
                        exportReport(header, sampledata);
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

                // start_report(header, path, 0);
                loadsamples(header, path, 0);


            }
        }

        function generateOverviewReport() {

            // let overviewReportDefs = [
            //     {name: "at 10:00", start: "", end: "", point: "10:00", precision: 30, presence: "", id: 0},
            //     {name: "08:00 to 12:00", start: "08:00", end: "12:00", point: "", precision: 0, presence: "", id: 0},
            //     {name: "at 20:30", start: "", end: "", point: "20:30", precision: 30, presence: "", id: 0},
            //     {name: "13:00 to 22:00", start: "13:00", end: "22:00", point: "", precision: 0, presence: "", id: 0},
            //     {name: "active 20:00 to 06:00", start: "", end: "", point: "", precision: 0, presence: "test", id: 0},
            //     {name: "active 02:00 to 06:00", start: "", end: "", point: "", precision: 0, presence: "test", id: 0},
            //     {name: "day", start: "08:00", end: "18:00", point: "", precision: 30, presence: "", id: 0, skip: true},
            //
            // ];

            let days = ['Sunday', 'Monday', 'Tuesday', 'Wednesday', 'Thursday', 'Friday', 'Saturday'];
            let months = ['January', 'February', 'March', 'April', 'May', 'June', 'July', 'August', 'September', 'October',
                'November', 'December'];
            // let dataLock = false;
            let periods = [];

            Date.prototype.getWeek = function() {
                var onejan = new Date(this.getFullYear(), 0, 1);
                return Math.ceil((((this - onejan) / 86400000) + onejan.getDay() + 1) / 7);
            }


            // processavgdata analyses the data from presence averages and start the
            // presence detection data collection
            function processavgdata(header, sampledata, api, analysis, tries) {
                // console.log(overviewReportDefs);
                // let data = header;
                // data += "\n";
                // console.log(sampledata);

                // identify the presence sets
                let presenceSets = [];
                let presenceSetsFlags = [];
                for (let j = 0; j < overviewReportDefs.length; j++) {
                    if (overviewReportDefs[j].presence !== "") {
                        overviewReportDefs[j].id = j;
                        presenceSets.push(overviewReportDefs[j]);
                        presenceSetsFlags.push(false);
                    }
                }

                if ((sampledata === null) || (sampledata === undefined)) {
                    alert("No data available for the selected time.");
                    document.getElementById("loader").style.visibility = "hidden";
                } else {
                    let currentDay = "", //current days in analysis
                        allResults = {}, // holds all results
                        cycleResult = new Array(overviewReportDefs.length + 1); // hold the current day result
                    // console.log(sampledata);
                    for (let i = 0; i < sampledata.length; i++) {
                        // console.log(sampledata[i].ts);
                        let d = new Date(sampledata[i].ts);
                        // var sampleDate = ("0" + d.getDate()).slice(-2) + "-" + ("0" + (d.getMonth() + 1)).slice(-2) + " " + d.getDay(),
                        var sampleDate = d.getFullYear() + "-" + ("0" + (d.getMonth() + 1)).slice(-2) + "-" + ("0" + d.getDate()).slice(-2) +
                            " " + d.getDay(),
                            sampleTime = ("0" + d.getHours()).slice(-2) + ":" + ("0" + d.getMinutes()).slice(-2);

                        // checks if this is a new day and sets al variables in case it is
                        if (currentDay !== sampleDate) {
                            if (currentDay !== "") {
                                for (let i = 0; i < presenceSets.length; i++) {
                                    if (cycleResult[presenceSets[i].id + 1] === undefined) {
                                        presenceSetsFlags[i] = true
                                    }
                                }
                                allResults[cycleResult[0]] = cycleResult;
                                cycleResult = new Array(overviewReportDefs.length + 1);
                            }
                            currentDay = sampleDate;
                            cycleResult[0] = currentDay;
                            // console.log(cycleResult)
                        }

                        // presenceSets = [];

                        for (let j = 0; j < overviewReportDefs.length; j++) {
                            if (overviewReportDefs[j].point !== "") {
                                // console.log(sampleTime);
                                // this is a point measure
                                let refT = parseInt(overviewReportDefs[j].point.replace(':', ''), 10),
                                    newT = parseInt(sampleTime.replace(':', ''), 10);
                                if ((cycleResult[j + 1] === undefined) || (cycleResult[j + 1] === null)) {
                                    // this is the first sample
                                    if (Math.abs(refT - newT) <= overviewReportDefs[j].precision) {
                                        cycleResult[j + 1] = [sampleTime, sampledata[i].val]
                                        // console.log("first sample");
                                        // console.log(refT, newT);
                                    }
                                } else {
                                    // we need to take the closest sample
                                    let oldT = parseInt(cycleResult[j + 1][0].replace(':', ''), 10);
                                    if ((Math.abs(refT - newT) <= overviewReportDefs[j].precision)
                                        && (Math.abs(refT - newT) <= Math.abs(refT - oldT))) {
                                        cycleResult[j + 1] = [sampleTime, sampledata[i].val];
                                        // console.log("next sample");
                                        // console.log(refT, oldT, newT);
                                    }
                                }
                            } else if ((overviewReportDefs[j].start !== "") && (overviewReportDefs[j].end !== ""))
                                if (overviewReportDefs[j].presence === "") {
                                    let sst = parseInt(overviewReportDefs[j].start.replace(':', ''), 10),
                                        sed = parseInt(overviewReportDefs[j].end.replace(':', ''), 10),
                                        stime = parseInt(sampleTime.replace(':', ''), 10);
                                    if ((stime >= sst) && (stime <= sed)) {
                                        // this is an interval detection
                                        // since the period is covered by samples, we can use arithmetic average
                                        // console.log(cycleResult[j]);
                                        if ((cycleResult[j + 1] === undefined) || (cycleResult[j + 1] === null)) {
                                            // first sample
                                            cycleResult[j + 1] = [1, sampledata[i].val]
                                        } else {
                                            cycleResult[j + 1][1] = (cycleResult[j + 1][1] * cycleResult[j + 1][0] + sampledata[i].val)
                                                / (cycleResult[j + 1][0] + 1);
                                            cycleResult[j + 1][0] += 1;
                                        }
                                    }
                                    // the if below is for development and will be removed
                                } else if (!developmentflag) {
                                    // check for presence, including closure time check
                                    // when presence can be determined for all days no need to enquire the server
                                    let name = api.split("?")[0];
                                    if (name.length > labellength) {
                                        name = name.slice(0, labellength)
                                    }
                                    let clts = parseInt(spaceTimes[name][0].replace(':', ''), 10),
                                        clte = parseInt(spaceTimes[name][1].replace(':', ''), 10),
                                        stime = parseInt(sampleTime.replace(':', ''), 10);
                                    if ((stime < clts) || (stime > clte)) {
                                        // console.log("not in closure");
                                        // when there has been no valid or onlym zero sample, we check for activity
                                        if ((cycleResult[j + 1] === undefined) || (cycleResult[j + 1] === 0)) {
                                            let sst = parseInt(overviewReportDefs[j].start.replace(':', ''), 10),
                                                sed = parseInt(overviewReportDefs[j].end.replace(':', ''), 10);
                                            // stime = parseInt(sampleTime.replace(':', ''), 10);
                                            if ((stime >= sst) && (stime <= sed)) {
                                                cycleResult[j + 1] = sampledata[i].val * 2
                                                // console.log("valid", sampledata[i].val)
                                                // }
                                            }
                                        }
                                    }
                                }
                            // } else if (overviewReportDefs[j].presence !== "") {
                            //     overviewReportDefs[j].id = j;
                            //     presenceSets.push(overviewReportDefs[j])
                            // }
                        }
                    }
                    // console.log(allResults);
                    allResults[cycleResult[0]] = cycleResult;
                    for (let i = 0; i < presenceSets.length; i++) {
                        if (cycleResult[presenceSets[i].id + 1] === undefined) {
                            presenceSetsFlags[i] = true
                        }
                    }
                    let filteredPresenceSets = [];
                    for (let i = 0; i < presenceSets.length; i++) {
                        if (presenceSetsFlags[i]) {
                            filteredPresenceSets.push(presenceSets[i])
                        }
                    }

                    // console.log(allResults);
                    // console.log(filteredPresenceSets);

                    // pass the data to the next func for presence
                    // console.log(allResults);
                    loadpresence(header, allResults, filteredPresenceSets, api, tries)

                    // temporary, to be deleted
                    // generateOverview(header, allResults);
                }
            }

            function loadavgsamples(header, api, analysis, tries) {
                $.ajax({
                    type: 'GET',
                    timeout: 5000,
                    url: ip + "/series?type=sample?space=" + api + "?analysis=" + analysis,
                    success: function (rawdata) {
                        try{
                        let sampledata = JSON.parse(rawdata);
                    // console.log(sampledata)
                    processavgdata(header, sampledata, api, analysis, tries);} catch (e) {
                            alert("received corrupted data: " + rawdata);
                            document.getElementById("loader").style.visibility = "hidden";
                        }
                    },
                    error: function (error) {
                        if (tries === maxtries) {
                            alert("Server or network error.\n Please try again later.");
                            console.log("Error samples:" + error);
                            document.getElementById("loader").style.visibility = "hidden";
                        } else {
                            // loadsamples(header, api, entrieslist, tries + 1)
                            loadavgsamples(header, api, analysis, tries + 1)
                        }
                    }

                });
            }

            function loadpresence(header, data, presenceSets, api, tries) {
                // while (dataLock) {
                // }
                // dataLock = true;
                // for (let i = 0; i < presenceSets.length; i++) {
                //     console.log(presenceSets[i])
                // }
                // console.log(data)

                if (presenceSets.length === 0) {
                    // generate report
                    // console.log("loadpresence", data);
                    generateOverview(header, data);
                } else {
                    // load data
                    let current = presenceSets[presenceSets.length - 1];
                    // console.log("DEBUG: ", ip + "/series?type=presence?space=" + api + "?analysis=" + current.presence);
                    console.log("DEBUG: ", ip + "/presence?space=" + api + "?analysis=" + current.presence);
                    $.ajax({
                        type: 'GET',
                        timeout: 30000,
                        // url: ip + "/series?type=presence?space=" + api + "?analysis=" + current.presence,
                        url: ip + "/presence?space=" + api + "?analysis=" + current.presence,
                        success: function (rawdata) {
                            presenceSets.pop();
                            try {
                            let sampledata = JSON.parse(rawdata);
                            // console.log("DEBUG", current.name, sampledata);
                            // return;
                            // console.log(data[5]);
                            // console.log(current.id);
                            // data[current.id] = sampledata[0].val;

                            // remove presence measure since loaded from the server
                            if ((sampledata !== null) && (sampledata !== undefined)) {
                                for (let i = 0; i < sampledata.length; i++) {
                                    let d = new Date(sampledata[i].ts);
                                    var sampleDate = d.getFullYear() + "-" + ("0" + (d.getMonth() + 1)).slice(-2) + "-" + ("0" + d.getDate()).slice(-2) +
                                        " " + d.getDay();
                                    // sampleTime = ("0" + d.getHours()).slice(-2) + ":" + ("0" + d.getMinutes()).slice(-2);
                                    // for (let i=0; i<data[sampleDate].length; i++) {console.log(data[sampleDate][i]);}
                                    data[sampleDate][current.id + 1] = sampledata[i].val;
                                }
                            }} catch (e) {console.log("received corrupted data: ",rawdata)}
                            // dataLock = false;
                            loadpresence(header, data, presenceSets, api, tries)
                        },
                        error: function (error) {
                            if (tries === maxtries) {
                                alert("Server or network error.\n Please try again later.");
                                console.log("Error samples:" + error);
                                document.getElementById("loader").style.visibility = "hidden";
                            } else {
                                // loadsamples(header, api, entrieslist, tries + 1)
                                // dataLock = false;
                                loadpresence(header, data, presenceSets, api, tries + 1)
                            }
                        }

                    });
                }
            }

            function generateOverview(header, data) {
                // TODO HERE - delete fake data
                // data = {
                //     "2019-07-31 3": ["31-07-2019 3", , , [0, 0], , , , ,],
                //     "2019-08-01 4": ["1-08-2019 4", 22, [0, 9], [0, 5], [0, 25], [0, 3], 22, 22, [0, 5]],
                //     "2019-08-02 5": ["2-08-2019 5", 22, [0, 9], [0, 5], [0, 25], [0, 3], 22, 22, [0, 5]],
                //     "2019-08-03 6": ["3-08-2019 6", 22, [0, 9], [0, 5], [0, 25], [0, 3], 22, 22, [0, 5]],
                //     "2019-08-04 0": ["4-08-2019 0", 22, [0, 9], [0, 5], [0, 25], [0, 3], 22, 22, [0, 5]],
                //     "2019-08-05 1": ["5-08-2019 1", 22, [0, 9], [0, 5], [0, 25], [0, 3], 22, 22, [0, 5]],
                //     "2019-08-06 2": ["6-08-2019 2", 22, [0, 9], [0, 5], , , , , [0, 5]],
                // };
                // console.log(data);
                let keys = Object.keys(data);
                // console.log(keys);
                let perioddayavg = [];
                let weekdayavg = [];
                let periodpointavg;
                let weekpointavg = [];
                let tmpDays = [];
                let tmppointDays;
                let weeks = [];
                let incfirstweek = -1;
                keys.sort();
                // console.log(periods);
                if (periods.length !== 0) {
                    tmppointDays = new Array(periods.length);
                    weekpointavg = new Array(periods.length);
                    periodpointavg = new Array(periods.length);
                    for (let i = 0; i < periods.length; i++) {
                        tmppointDays[i] = [-1];
                        weekpointavg[i] = [];
                        periodpointavg[i] = []
                    }
                }
                // console.log(tmppointDays);
                for (let k in keys) {
                    // console.log(data[keys[k]]);
                    let v = data[keys[k]];
                    let daydatetmp = (v[0].split(" ")[0]).split("-");
                    let daydate = daydatetmp[2] + " " + months[parseInt(daydatetmp[1]) - 1] + " " + daydatetmp[0];
                    // console.log(daydatetmp);
                    var dtmp = new Date(daydatetmp[0], parseInt(daydatetmp[1]) - 1, daydatetmp[2]);
                    // console.log(d, d.getWeek());
                    // return
                    header += daydate + "," + days[parseInt(v[0].split(" ")[1])]+ "," + dtmp.getWeek();
                    let valid = false;
                    for (let a = 1; a < v.length; a++) {
                        if (v[a] !== undefined) {
                            valid = true
                        }
                    }
                    // console.log(valid);
                    if (valid) {
                        if ((data[keys[k]][0].split(" ")[1] === "0") && (tmpDays.length !== 0)) {
                            // start week, calculate and store average, reset tmp array
                            let val = 0;
                            weeks.push(dtmp.getWeek());
                            if (incfirstweek === -1) {incfirstweek=1}
                            for (let i = 0; i < tmpDays.length; i++) {
                                val += tmpDays[i]
                            }
                            // console.log(val, tmp.length);
                            if (tmpDays.length !== 0) {
                                val = Math.round(val / tmpDays.length)
                            }
                            weekdayavg.push(val);
                            // console.log(tmp);
                            if (overviewSkipDays.indexOf(data[keys[k]][0].split(" ")[1]) === -1) {
                                tmpDays = [v[v.length - 1][1]];
                                if (tmppointDays.length !== 0) {
                                    for (let i = 0; i < periods.length; i++) {
                                        tmppointDays[i].push(v[periods[i] + 1][1]);
                                        tmppointDays[i].push(-1);
                                    }
                                }
                            } else {
                                tmpDays = [];
                                if (tmppointDays.length !== 0) {
                                    for (let i = 0; i < periods.length; i++) {
                                        tmppointDays[i].push(-1);
                                    }
                                }
                            }
                            // console.log(tmp);
                        } else {
                            // console.log(v[v.length-1][1]);
                            if (incfirstweek === -1) {incfirstweek=0;weeks.push(dtmp.getWeek());}
                            if (overviewSkipDays.indexOf(data[keys[k]][0].split(" ")[1]) === -1) {
                                if (v[v.length - 1] === undefined) {
                                    v[v.length - 1] = [0, 0]
                                }
                                tmpDays.push(v[v.length - 1][1]);
                                if (tmppointDays.length !== 0) {
                                    for (let i = 0; i < periods.length; i++) {
                                        // console.log(v, periods[i]);
                                        // we assume that, except for the last sample, undefined is missing data to be set tot zero
                                        // comment the first if when this is not valid
                                        if ((v[periods[i] + 1] === undefined) && (k !== (keys.length - 1).toString())) {
                                            v[periods[i] + 1] = [0, 0]
                                        }
                                        if (v[periods[i] + 1] !== undefined) {
                                            tmppointDays[i].push(v[periods[i] + 1][1])
                                        }
                                    }
                                }
                            }
                        }
                        if (overviewSkipDays.indexOf(data[keys[k]][0].split(" ")[1]) === -1) {
                            perioddayavg.push(v[v.length - 1][1])
                        }
                        if (overviewSkipDays.indexOf(data[keys[k]][0].split(" ")[1]) === -1) {
                            for (let j = 0; j < overviewReportDefs.length; j++) {
                                // console.log(overviewReportDefs[j]);
                                if (!overviewReportDefs[j].skip) {
                                    // console.log(v[j+1]);
                                    if ((v[j + 1] !== null) && (v[j + 1] !== undefined)) {
                                        if (overviewReportDefs[j].presence !== "") {
                                            if (v[j + 1] >= 2) {
                                                header += ",true"
                                            } else {
                                                header += ",false"
                                            }
                                        } else {
                                            header += "," + Math.round(v[j + 1][1]);
                                        }
                                    } else {
                                        if (overviewReportDefs[j].presence !== "") {
                                            let et = parseInt(overviewReportDefs[j].end.replace(':', ''), 10);
                                            // let st = parseInt(overviewReportDefs[j].start.replace(':', ''), 10);
                                            // if (st > et) {st = 0};
                                            let ct0 = new Date();
                                            let ct = parseInt(ct0.getHours() + ("0" + ct0.getMinutes()).slice(-2), 10)
                                            // console.log(st,et, ct);
                                            if (ct > et) {
                                                switch(defaultPresence) {
                                                    case 0:
                                                        header += ",false";
                                                        break
                                                    case 1:
                                                        header += ",true"
                                                        break
                                                    default:
                                                        header += ","
                                                }
                                            } else {header += ","}
                                        } else {header += ","}
                                        
                                    }
                                }
                            }
                            // console.log(keys[k], data[keys[k]])
                        }
                    }
                    header += "\n";
                }
                // console.log(tmppointDays);
                // console.log(tmpDays);
                let val = 0;
                for (let i = 0; i < tmpDays.length; i++) {
                    val += tmpDays[i]
                }
                // console.log(val, tmp.length);
                if (tmpDays.length !== 0) {
                    val = Math.round(val / tmpDays.length)
                }
                weekdayavg.push(val);
                val = 0;
                for (let i = 0; i < perioddayavg.length; i++) {
                    val += perioddayavg[i]
                }
                if (perioddayavg.length !== 0) {
                    val = Math.round(val / perioddayavg.length)
                }
                perioddayavg = [val];
                // header += "\n";
                if (periods.length !== 0) {
                    // calculate all week averages first and store all data for the period average
                    for (let i = 0; i < periods.length; i++) {
                        // console.log(tmppointDays[i]);
                        let valPoint = [];
                        for (let j = 0; j < tmppointDays[i].length; j++) {
                            // console.log(tmppointDays[i][j]);
                            if (tmppointDays[i][j] !== -1) {
                                // store data point
                                valPoint.push(tmppointDays[i][j]);
                                periodpointavg[i].push(tmppointDays[i][j])
                            } else {
                                if (valPoint.length !== 0) {
                                    // console.log(valPoint);
                                    let acc = 0;
                                    for (let k = 0; k < valPoint.length; k++) {
                                        acc += valPoint[k]
                                    }
                                    if (valPoint.length !== 0) {
                                        weekpointavg[i].push(Math.round(acc / valPoint.length))
                                    } else {
                                        weekpointavg[i].push(0)
                                    }
                                    // console.log(i, valPoint, weekpointavg[i]);
                                }
                                valPoint = []
                            }
                        }
                        // console.log(valPoint);
                        let acc = 0;
                        for (let k = 0; k < valPoint.length; k++) {
                            acc += valPoint[k]
                        }
                        if (valPoint.length !== 0) {
                            weekpointavg[i].push(Math.round(acc / valPoint.length))
                        } else {
                            weekpointavg[i].push(0)
                        }
                        // console.log(i, valPoint, weekpointavg[i])
                    }
                }
                // console.log(weekpointavg);
                // write period report if required
                // if ((periods !== undefined) && (weekdayavg !== undefined)) {
                //     for (let j = 0; j < periods.length; j++) {
                //         for (let i = 0; i < weekdayavg.length; i++) {
                //             header += "Presence average " + overviewReportDefs[periods[j]].name + " for week " + i + ", " + weekpointavg[j][i] + "\n";
                //         }
                //         let acc = 0;
                //         for (let i = 0; i < periodpointavg[j].length; i++) {
                //             acc += periodpointavg[j][i]
                //             // console.log(periodpointavg[j][i])
                //         }
                //         // console.log(acc);
                //         if (periodpointavg[j].length !== 0) {
                //             header += "Presence average " + overviewReportDefs[periods[j]].name + " for the full period, " +
                //                 Math.round(acc / periodpointavg[j].length) + "\n"
                //         } else {
                //             header += "Presence average " + overviewReportDefs[periods[j]].name + " for the full period, 0"
                //         }
                //         header += "\n";
                //     }
                // console.log(periods);
                for (let i = 0; i < weekdayavg.length; i++) {
                    if ((i===0) && (incfirstweek===0)) {                    
                        header += "Average weekly*,, " + weeks[i] + ",";
                } else {
                    header += "Average weekly,, " + weeks[i] + ",";
                }
                    
                    for (let k = 0; k < overviewReportDefs.length; k++) {
                        j = periods.indexOf(k);
                        // console.log(j, k)
                        if (j !== -1) {
                            header += weekpointavg[j][i] + ","
                        } else {
                            header += ","
                        }
                    }
                    header += "\n"
                }
                // console.log(periodpointavg);
                header += "Average full period,,,";
                // for (let i = 0; i < periods.length; i++) {
                for (let k = 0; k < overviewReportDefs.length; k++) {
                    j = periods.indexOf(k);
                    if (j !== -1) {
                        let acc = 0;
                        for (let i = 0; i < periodpointavg[j].length; i++) {
                            acc += periodpointavg[j][i]
                            // console.log(periodpointavg[j][i])
                        }
                        if (periodpointavg[j].length !== 0) {
                            header += Math.round(acc / periodpointavg[j].length) + ","
                        } else {
                            header += "0,"
                        }
                    } else {
                        header += ","
                    }
                }
                // }
                header += "\n\n\n";
                header += "#SUMMARY OCCUPANCY\n" 
                      + "\"#working day: from " + overviewReportDefs[overviewReportDefs.length - 1].start 
                      + " to " + overviewReportDefs[overviewReportDefs.length - 1].end + " \"\n"
                      +"\n";
                header += "Description, Week, Average\n";
                for (let i = 0; i < weekdayavg.length; i++) {
                    if ((i===0) && (incfirstweek===0)) {                    
                        header += "Working day average*, " + weeks[i] + "," + weekdayavg[i] + "\n";
                } else {
                    header += "Working day average, " + weeks[i] + "," + weekdayavg[i] + "\n";
                }
                }
                header += "Working day average,full period," + perioddayavg[0] + "\n";
                // console.log(periodavg[0]);
                // console.log(weekavg);
                // console.log(header);
                if (incfirstweek===0) {
                    header += "\n\n* Partial week\n"
                }
                var blob = new Blob([header], {type: 'text/plain'}),
                    anchor = document.createElement('a');
                var currentTime = new Date();
                anchor.download = currentTime.getFullYear().toString() + "_" + (currentTime.getMonth() + 1).toString() + "_" +
                    currentTime.getDate().toString() + "_" + space + "_" + asys + ".csv";
                anchor.href = (window.webkitURL || window.URL).createObjectURL(blob);
                anchor.dataset.downloadurl = ['text/plain', anchor.download, anchor.href].join(':');
                anchor.click();
                document.getElementById("loader").style.visibility = "hidden";
            }

            let select = document.getElementById("spacename");
            var myindex = select.selectedIndex,
                space = select.options[myindex].value;
            // select = document.getElementById("reptype");
            // myindex = select.selectedIndex;
            var asys = "overview",
                copyendDate = new Date(endDate),
                start, end;
            if ((startDate !== undefined) && (endDate !== undefined)
                && (space !== "Choose a space")) {
                document.getElementById("loader").style.visibility = "visible";
                start = startDate.getUnixTime();
                copyendDate.setHours(endDate.getHours() + 23);
                copyendDate.setMinutes(endDate.getMinutes() + 59);
                if (copyendDate.getUnixTime() > Date.now()) {
                    end = Date.now();
                } else {
                    end = copyendDate.getUnixTime();
                }
                let header = "sep=,\n" +
                    "#Xetal Flow Monitoring: " + version + " \n"
                    + "\"#user: " + user + " \"\n"
                    + "\"#space: " + space + " \"\n"
                    + "\"#start: " + startDate.toDateString() + " \"\n"
                    + "\"#end: " + copyendDate.toDateString() + " \"\n"
                    + reportWarning + "\n\n"
                    +  "#OCCUPANCY REPORT\n\n" 
                    // + "\"NOTE: all values are averages if not specified otherwise\"\n\n"
                    // + "Date,Day,";
                    + "Date,Day,Week,";
                // for (let i in overviewReportDefs) {
                for (let i = 0; i < overviewReportDefs.length; i++) {
                    if (!overviewReportDefs[i].skip) {
                        if (overviewReportDefs[i].presence === "") {
                            if (overviewReportDefs[i].point !== "") {
                                header += "presence ";
                                // periods.push(i)
                            } else {
                                header += "presence average "
                            }
                            periods.push(i)
                        }
                        header += overviewReportDefs[i].name + ",";
                    }
                }
                // console.log(header);
                // header += "Date, , at 10:00, 08:00 to 12:00, at 14:00, 13:00 to 17:00, activity from 10:00 to 06:00?\n";
                header = header.substring(0, header.length - 1) + "\n";
                let path = space + "?start=" + start + "?end=" + end;

                loadavgsamples(header, path, refOverviewAsys, 0);

                // for (let i = 0; i < overviewReportDefs.length; i++) {
                //     console.log(overviewReportDefs2[i]);
                // }

            }
        }

    }
);