$(document).ready(function () {
        // NOTE: gets spacename and spacenameUncoded from drawing.js
        document.getElementById("loader").style.visibility = "hidden";
        Date.prototype.getUnixTime = function () {
            return (this.getTime() / 1000 | 0) * 1000
        };
        let allowedEndDate = new Date(),
            boundarySamplesVal = -1;
        if (rtshow[0] !== "dbg") {
            allowedEndDate.setDate(allowedEndDate.getDate() - 1)
        }
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
                // maxDate: new Date(),
                maxDate: allowedEndDate,
                onSelect: function () {
                    startDate = this.getDate();
                    updateStartDate();
                }
            }),
            endPicker = new Pikaday({
                field: document.getElementById('end'),
                minDate: new Date(StartDat),
                // maxDate: new Date(),
                maxDate: allowedEndDate,
                onSelect: function () {
                    endDate = this.getDate();
                    updateEndDate();
                }
            }),
            _startDate = startPicker.getDate(),
            _endDate = endPicker.getDate();

        maxtries = 3;


        if (_startDate) {
            startDate = _startDate;
            updateStartDate();
        }

        if (_endDate) {
            endDate = _endDate;
            updateEndDate();
        }

        document.getElementById("gen").addEventListener("click", generateReport);
        document.getElementById("graphdata").addEventListener("click", loadGraphsData);


        function timeLikeDiff(first, second) {
            return (Math.trunc(first / 100) - Math.trunc(second / 100)) * 60 +
                Math.round(((first / 100 - Math.floor(first / 100)) - (second / 100 - Math.floor(second / 100))) * 100)
        }

        function loadGraphsData() {

            // returns two integers in unix format for date and time
            Date.prototype.ToInt2 = function () {
                let date = this.getFullYear().toString() + ("0" + (this.getMonth() + 1).toString()).slice(-2) +
                    ("0" + this.getDate().toString()).slice(-2);
                let time = ("0" + this.getHours()).slice(-2) + ("0" + this.getMinutes()).slice(-2) + ("0" + this.getSeconds()).slice(-2);
                return [parseInt(date, 10), parseInt(time, 10)]
            };

            // takes a string of format hh:mm and uses to replace the time
            Date.prototype.ReplaceTime = function (time) {
                let tmp = time.split(":"),
                    hour = parseInt(tmp[0], 10),
                    minutes = parseInt(tmp[1], 10);
                this.setHours(hour, minutes, 0, 0)
            };

            // return a list fo the samples needed insertion
            function insertSamples(currentPhase, nextPhase, cDay, defVal) {
                let retVal = [],
                    currentDay = new Date(cDay);
                switch (currentPhase) {
                    case 0:
                    case 1:
                        if (nextPhase > 1) {
                            currentDay.ReplaceTime(opStartTime);
                            retVal.push({ts: new Date(currentDay), val: defVal})
                        }
                        if (nextPhase > 2) {
                            currentDay.ReplaceTime(opEndTime);
                            retVal.push({ts: new Date(currentDay), val: defVal})
                        }

                        break;
                    case 2:
                        if (nextPhase > 2) {
                            currentDay.ReplaceTime(opEndTime);
                            retVal.push({ts: new Date(currentDay), val: defVal})
                        }
                        break;
                    default:
                        break;
                }
                return retVal
            }

            // removes all samples ooutside of the working period and creates boundary values set to defVal
            function cleanSampleList(trace, defVal) {
                // console.log(trace);

                let startAn = startDate.ToInt2()[0],
                    endAn = endDate.ToInt2()[0],
                    startRef = parseInt(opStartTime.replace(":", "") + "00", 10),
                    endRef = parseInt(opEndTime.replace(":", "") + "00", 10),
                    refDayDate = startDate,
                    refDay = startAn,
                    cyclePhase = 0,
                    samples = [];

                for (let i = 0; i < trace.length; i++) {
                    let cursorDate = new Date(trace[i].ts),
                        cursor = cursorDate.ToInt2();
                    if (cursor[0] !== refDay) {
                        // Day has changed, we need to add samples if needed
                        samples = samples.concat(insertSamples(cyclePhase, 3, refDayDate, defVal));
                        refDay = cursor[0];
                        refDayDate = cursorDate;
                        cyclePhase = 0
                    }
                    if (cursor[1] <= startRef) {
                        cyclePhase = 1
                    } else if (cursor[1] >= endRef) {
                        if (cyclePhase !== 3) {
                            samples = samples.concat(insertSamples(cyclePhase, 3, refDayDate, defVal));
                            cyclePhase = 3
                        }
                    } else {
                        if (cyclePhase !== 2) {
                            samples = samples.concat(insertSamples(cyclePhase, 2, refDayDate, defVal));
                            cyclePhase = 2
                        } else {
                            // add normal sample
                            samples.push({ts: cursorDate, val: trace[i].val})
                        }
                    }
                }

                // add sample left in the current day
                if (cyclePhase < 3) {
                    samples = samples.concat(insertSamples(cyclePhase, 3, refDayDate, defVal))
                }

                // add sample for empty following days
                cyclePhase = 0;
                refDayDate = new Date(trace[trace.length - 1].ts);
                while (refDay < endAn) {
                    refDayDate.setDate(refDayDate.getDate() + 1);
                    refDay = refDayDate.ToInt2()[0];
                    samples = samples.concat(insertSamples(cyclePhase, 3, refDayDate, defVal));
                    cyclePhase = 0
                }

                // console.log(samples);

                return samples
            }

            function loadAllSample(space, path, meas, maxtries) {
                if (meas.length === 0) {
                    document.getElementById("loader").style.visibility = "hidden";
                    chartArchive.render();
                    return
                }
                let asys = meas[meas.length - 1].name;
                // used for alias
                foundMeas = Object.keys(aliasMeasurement).filter(function (key) {
                    return aliasMeasurement[key] === asys;
                });
                if (foundMeas.length !== 0) {
                    asys = foundMeas[0]
                }
                $.ajax({
                    type: 'GET',
                    timeout: 100000,
                    url: ip + "/series?type=sample?space=" + space + "?analysis=" + asys + path,
                    success: function (rawdata) {
                        // console.log(ip + "/series?type=sample?space=" + space + "?analysis=" + asys + path);
                        meas.pop();
                        let jsonData;
                        try {
                            jsonData = JSON.parse(rawdata);
                        } catch (e) {
                            console.log("received corrupted data")
                        }
                        // console.log(jsonData);
                        if ((jsonData !== undefined) && (jsonData !== null)) {
                            let sampledata = cleanSampleList(jsonData, boundarySamplesVal);
                            // console.log(sampledata);
                            for (let i = 0; i < sampledata.length; i++) {
                                dataArraysArchive[meas.length].push({
                                    x: sampledata[i].ts,
                                    y: sampledata[i].val
                                });
                            }
                        }
                        // console.log(dataArraysArchive[meas.length]);
                        loadAllSample(space, path, meas, 0);
                    },
                    error: function (error) {
                        if (tries === maxtries) {
                            alert("Range of data requested too large or network error.\n Please try a shorter period or try again later.");
                            console.log("Error samples:" + error);
                            document.getElementById("loader").style.visibility = "hidden";
                        } else {
                            // loadsamples(header, api, entrieslist, tries + 1)
                            loadsamples(space, path, meas, maxtries + 1);
                        }
                    }
                });
            }

            // let select = document.getElementById("spacename");
            // var myindex = select.selectedIndex,
            // space = select.options[myindex].value;
            // let space = spacename;
            var copyendDate = new Date(endDate),
                start, end;
            if ((startDate !== undefined) && (endDate !== undefined)
                && (spacename !== "Choose a space")) {
                document.getElementById("loader").style.visibility = "visible";
                start = startDate.getUnixTime();
                copyendDate.setHours(endDate.getHours() + 23);
                copyendDate.setMinutes(endDate.getMinutes() + 59);
                if (copyendDate.getUnixTime() > Date.now()) {
                    end = Date.now();
                } else {
                    end = copyendDate.getUnixTime();
                }
                for (let i = 0; i < dataArraysArchive.length; i++) {
                    dataArraysArchive[i].length = 0;
                }
                let path = "?start=" + start + "?end=" + end;

                loadAllSample(spacename, path, allmeasurements.slice(), 0);


            }

        }

        function generateReport() {

            let select = document.getElementById("spacename");
            let myindex = select.selectedIndex;
            if (repvisile || (rtshow[0] === "dbg")) {
                select = document.getElementById("reptype");
                myindex = select.selectedIndex;
                let asys = select.options[myindex].value;
                if (asys === "overview") {
                    generateOverviewReport()
                } else {
                    generatePeriodicReport()
                }
            } else {
                generateOverviewReport()
            }
        }


        function generateOverviewReport() {

            let days = ['Sunday', 'Monday', 'Tuesday', 'Wednesday', 'Thursday', 'Friday', 'Saturday'];
            let months = ['January', 'February', 'March', 'April', 'May', 'June', 'July', 'August', 'September', 'October',
                'November', 'December'];
            // let dataLock = false;
            let periods = [];

            Date.prototype.getWeek = function () {
                var onejan = new Date(this.getFullYear(), 0, 1);
                return Math.ceil((((this - onejan) / 86400000) + onejan.getDay() + 1) / 7);
            };


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
                                    if (Math.abs(timeLikeDiff(refT, newT)) <= overviewReportDefs[j].precision + 1) {
                                        cycleResult[j + 1] = [sampleTime, sampledata[i].val]
                                        // console.log("first sample");
                                        // console.log(refT, newT);
                                    }
                                } else {
                                    // we need to take the closest sample
                                    let oldT = parseInt(cycleResult[j + 1][0].replace(':', ''), 10);
                                    if ((Math.abs(timeLikeDiff(refT, newT)) <= overviewReportDefs[j].precision + 1)
                                        && (Math.abs(timeLikeDiff(refT, newT)) <= Math.abs(timeLikeDiff(refT, oldT)))) {
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
                                } else if (!noSamplePresenceCheck) {
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
                    timeout: 100000,
                    url: ip + "/series?type=sample?space=" + api + "?analysis=" + analysis,
                    success: function (rawdata) {
                        // console.log(ip + "/series?type=sample?space=" + api + "?analysis=" + analysis,);
                        try {
                            let sampledata = JSON.parse(rawdata);
                            // console.log(sampledata)
                            processavgdata(header, sampledata, api, analysis, tries);
                        } catch (e) {
                            alert("received corrupted data");
                            document.getElementById("loader").style.visibility = "hidden";
                        }
                    },
                    error: function (error) {
                        if (tries === maxtries) {
                            alert("Range of data requested too large or network error.\n Please try a shorter period or try again later.");
                            console.log("Error samples:" + error);
                            document.getElementById("loader").style.visibility = "hidden";
                        } else {
                            // console.log(error);
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
                    // console.log("DEBUG: ", ip + "/presence?space=" + api + "?analysis=" + current.presence);
                    $.ajax({
                        type: 'GET',
                        timeout: 100000,
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
                                }
                            } catch (e) {
                                console.log("received corrupted data: ", rawdata)
                            }
                            // dataLock = false;
                            loadpresence(header, data, presenceSets, api, tries)
                        },
                        error: function (error) {
                            if (tries === maxtries) {
                                alert("Range of data requested too large or network error.\n Please try a shorter period or try again later.");
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
                let keys = Object.keys(data);
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
                    header += daydate + "," + days[parseInt(v[0].split(" ")[1])] + "," + dtmp.getWeek();
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
                            if (incfirstweek === -1) {
                                incfirstweek = 1
                            }
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
                                        if (v[periods[i] + 1] !== undefined) {
                                            if (v[periods[i] + 1].length === 2) {
                                                tmppointDays[i].push(v[periods[i] + 1][1]);
                                            }
                                        }
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
                            if (incfirstweek === -1) {
                                incfirstweek = 0;
                                weeks.push(dtmp.getWeek());
                            }
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
                                            let ct0 = new Date();
                                            let ct = parseInt(ct0.getHours() + ("0" + ct0.getMinutes()).slice(-2), 10);
                                            // console.log(st,et, ct);
                                            if ((ct > et) && rtshow[0] !== "dbg") {
                                                switch (defaultPresence) {
                                                    case 0:
                                                        header += ",false";
                                                        break;
                                                    case 1:
                                                        header += ",true";
                                                        break;
                                                    default:
                                                        header += ","
                                                }
                                            } else {
                                                header += ","
                                            }
                                        } else {
                                            header += ","
                                        }

                                    }
                                }
                            }
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
                    val = Math.trunc(Math.round(val) * 10 / tmpDays.length) / 10
                }
                weekdayavg.push(val);
                val = 0;
                for (let i = 0; i < perioddayavg.length; i++) {
                    val += perioddayavg[i]
                }
                if (perioddayavg.length !== 0) {
                    val = Math.trunc(Math.round(val) * 10 / perioddayavg.length) / 10
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
                                    let acc = 0;
                                    for (let k = 0; k < valPoint.length; k++) {
                                        acc += valPoint[k]
                                    }
                                    if (valPoint.length !== 0) {
                                        weekpointavg[i].push(Math.trunc(Math.round(acc) * 10 / valPoint.length) / 10)
                                    } else {
                                        weekpointavg[i].push(0)
                                    }
                                }
                                valPoint = []
                            }
                        }
                        let acc = 0;
                        for (let k = 0; k < valPoint.length; k++) {
                            acc += valPoint[k]
                        }
                        if (valPoint.length !== 0) {
                            weekpointavg[i].push(Math.trunc(Math.round(acc) * 10 / valPoint.length) / 10)
                        } else {
                            weekpointavg[i].push(0)
                        }
                    }
                }
                header += "\n";
                for (let i = 0; i < weekdayavg.length; i++) {
                    if ((i === 0) && (incfirstweek === 0)) {
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
                header += "Average full period,,,";
                for (let k = 0; k < overviewReportDefs.length; k++) {
                    j = periods.indexOf(k);
                    if (j !== -1) {
                        let acc = 0;
                        for (let i = 0; i < periodpointavg[j].length; i++) {
                            acc += periodpointavg[j][i]
                        }
                        if (periodpointavg[j].length !== 0) {
                            header += (Math.trunc(Math.round(acc) * 10 / periodpointavg[j].length) / 10) + ","
                        } else {
                            header += "0,"
                        }
                    } else {
                        header += ","
                    }
                }
                header += "\n\n\n";
                header += "#SUMMARY OCCUPANCY\n"
                    + "\"#working day: from " + overviewReportDefs[overviewReportDefs.length - 1].start
                    + " to " + overviewReportDefs[overviewReportDefs.length - 1].end + " \"\n"
                    + "\n";
                header += "Description, Week, Average\n";
                for (let i = 0; i < weekdayavg.length; i++) {
                    if ((i === 0) && (incfirstweek === 0)) {
                        header += "Working day average*, " + weeks[i] + "," + weekdayavg[i] + "\n";
                    } else {
                        header += "Working day average, " + weeks[i] + "," + weekdayavg[i] + "\n";
                    }
                }
                header += "Working day average,full period," + perioddayavg[0] + "\n";
                if (incfirstweek === 0) {
                    header += "\n\n* Partial week\n"
                }
                var blob = new Blob([header], {type: 'text/plain'}),
                    anchor = document.createElement('a');
                var currentTime = new Date();
                anchor.download = currentTime.getFullYear().toString() + "_" + (currentTime.getMonth() + 1).toString() + "_" +
                    currentTime.getDate().toString() + "_" + spacenameUncoded.replace(/ /g, "_") + "_" + asys + ".csv";
                anchor.href = (window.webkitURL || window.URL).createObjectURL(blob);
                anchor.dataset.downloadurl = ['text/plain', anchor.download, anchor.href].join(':');
                anchor.click();
                document.getElementById("loader").style.visibility = "hidden";
            }

            let asys = "overview",
                copyendDate = new Date(endDate),
                start, end;
            if ((startDate !== undefined) && (endDate !== undefined)
                && (spacename !== "Choose a space")) {
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
                    + "\"#edition: " + edition + " \"\n"
                    + "\"#space: " + spacenameUncoded + " \"\n"
                    + "\"#datatype: average sample \n"
                    + "\"#start: " + startDate.toDateString() + " \"\n"
                    + "\"#end: " + copyendDate.toDateString() + " \"\n"
                    + reportWarning + "\n\n"
                    + "#OCCUPANCY REPORT\n\n"
                    + "Date,Day,Week,";
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
                header = header.substring(0, header.length - 1) + "\n";
                let path = spacename + "?start=" + start + "?end=" + end;

                loadavgsamples(header, path, refOverviewAsys, 0);

            }
        }

        function generatePeriodicReport() {

            function exportReport(header, sampledata) {
                let data = header,
                    rawdataSample = [],
                    finalData = {};
                data += "\n";
                if ((sampledata["data"] !== null) && (sampledata["data"] !== undefined)) {
                    sampledata = sampledata["data"]

                    for (let i = 0; i < sampledata.length; i++) {
                        let d = new Date(sampledata[i]["ts"]);
                        var datestring = ("0" + d.getDate()).slice(-2) + "-" + ("0" + (d.getMonth() + 1)).slice(-2) + "-" +
                                d.getFullYear() + " " + ("0" + d.getHours()).slice(-2) + ":" + ("0" + d.getMinutes()).slice(-2);
                        data += datestring + ", " + Math.trunc(sampledata[i]["ts"] / 1000)
                        if (sampledata[i]["corruptedData"]) {
                            // data is corrupted and skipped
                            data += "\n"

                        } else {
                            data += ", " + sampledata[i]["avgPresence"];
                            for (let k = 0; k < spaceDefinitions[spacename].length; k++) {
                                for (let j = 0; j < sampledata[i]["totalEntries"].length; j++) {
                                    if (sampledata[i]["totalEntries"][j]["id"] === spaceDefinitions[spacename][k]) {
                                        data += "," + sampledata[i]["totalEntries"][j]["netflow"] + "," + sampledata[i]["totalEntries"][j]["in"] +
                                        "," + sampledata[i]["totalEntries"][j]["out"];
                                    }
                                }
                            }
                            data += "\n";
                        }
                    }

                    var blob = new Blob([data], {type: 'text/plain'}),
                        anchor = document.createElement('a');
                    var currentTime = new Date();
                    anchor.download = currentTime.getFullYear().toString() + "_" + (currentTime.getMonth() + 1).toString() + "_" +
                        currentTime.getDate().toString() + "_" + spacenameUncoded.replace(/ /g, "_") + "_" + asys + ".csv";
                    // anchor.download = space + "_" + asys + ".csv";
                    anchor.href = (window.webkitURL || window.URL).createObjectURL(blob);
                    anchor.dataset.downloadurl = ['text/plain', anchor.download, anchor.href].join(':');
                    anchor.click();
                } else {
                    alert("No data available for the selected time.");
                }

                document.getElementById("loader").style.visibility = "hidden";
            }

            function loadsamples(header, api, tries) {
                $.ajax({
                    type: 'GET',
                    timeout: 100000,
                    url: ip + "/series?type=entry?space=" + api,
                    success: function (rawdata) {
                        // console.log(ip + "/series?type=entry?space=" + api);
                        let sampledata;
                        try {
                            sampledata = JSON.parse(rawdata);
                        } catch (e) {
                            // this also capture when no data is sent unfortunately, check with the API first what is happening
                            console.log(e)
                            console.log("received corrupted data")
                        }

                        exportReport(header, sampledata);
                    },
                    error: function (error) {
                        if (tries === maxtries) {
                            alert("Range of data requested too large or network error.\n Please try a shorter period or try again later.");
                            console.log("Error samples:" + error);
                            document.getElementById("loader").style.visibility = "hidden";
                        } else {
                            // loadsamples(header, api, entrieslist, tries + 1)
                            loadsamples(header, api, tries + 1)
                        }
                    }

                });
            }

            let entryList = spaceDefinitions[spacename],
                select = document.getElementById("reptype"),
                myindex = select.selectedIndex,
                asys = select.options[myindex].value,
                copyendDate = new Date(endDate),
                start, end;
            if ((startDate !== undefined) && (endDate !== undefined)
                && (spacename !== "Choose a space") && (asys !== "Choose a dataset") && (entryList.length !== 0)) {
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
                    + "\"#edition: " + edition + " \"\n"
                    + "\"#space: " + spacenameUncoded + " \"\n"
                    + "\"#datatype: average presence at a given interval \"\n"
                    + "\"#datatype: net, inbound and outbound flow at the given time instant \"\n"
                foundMeas = Object.keys(aliasMeasurement).filter(function (key) {
                    return aliasMeasurement[key] === asys;
                });
                if (foundMeas.length !== 0) {
                    asys = foundMeas[0]
                }
                header += "\"#start: " + startDate + " \"\n"
                    + "\"#end: " + copyendDate + " \"\n\n";
                header += "Date/Time, Epoch Time (s), interval average presence";
                if (spaceDefinitions[spacename] === undefined) {
                    document.getElementById("loader").style.visibility = "hidden";
                    alert("Error in spaced definition for " + spacename + "\nContact support or restart server.");
                    console.log(spacename);
                } else {
                    for (i = 0; i < spaceDefinitions[spacename].length; i++) {
                        header += ", netflow E" + spaceDefinitions[spacename][i] + ", inflow E" + spaceDefinitions[spacename][i] +
                        ", outflow E" + spaceDefinitions[spacename][i];

                    }
                    let path = spacename + "?analysis=" + asys + "?start=" + start + "?end=" + end;
                    loadsamples(header, path, 0);
                }
            }
        }

    }
);