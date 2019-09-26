var repvisile = (reportCurrent) || (repshow.length !== 0);

let spacename = "";
let spacenameUncoded = "";
let measurement = "sample";
let allmeasurements = [];
// let selel = null;
let repPeriod = 4000;
// let flowWarning = false;
// let lastTS = [];

let regex = new RegExp(':', 'g');
let timeNow = new Date(),
    timeNowHS = ("0" + timeNow.getHours()).slice(-2) + ":" + ("0" + timeNow.getMinutes()).slice(-2),
    incycle = ((parseInt(opStartTime.replace(regex, ''), 10) < parseInt(timeNowHS.replace(regex, ''), 10))
        && (parseInt(timeNowHS.replace(regex, ''), 10) < parseInt(opEndTime.replace(regex, ''), 10)));

if ((!incycle) && (openingTime !== "")) {
    alert("!!! WARNING !!!\nReal-time data is only available " + openingTime + ".\nReporting is always available.\n")
}

let rtdataDefinitions = [];
let archivedataDefinitions = [];
let flowdataDefinitions = [];
let dataArrays = [];
let dataArraysArchive = [];
let dataArraysFlow = [];
// let dataPoints = [];
let mapnames = {};

var ip = "http://" + location.hostname + ":" + location.port;
// console.log(ip);

const colors = [
    '#000000', '#e6194b', '#3cb44b', '#ffe119', '#4363d8', '#f58231', '#911eb4', '#46f0f0', '#f032e6', '#bcf60c', '#fabebe',
    '#008080', '#e6beff', '#9a6324', '#fffac8', '#800000', '#aaffc3', '#808000', '#ffd8b1', '#000075', '#808080'
];

function toogleRTDataSeries(e) {
    e.dataSeries.visible = !(typeof (e.dataSeries.visible) === "undefined" || e.dataSeries.visible);
    chartRT.render();
}

function toogleArchiveDataSeries(e) {
    e.dataSeries.visible = !(typeof (e.dataSeries.visible) === "undefined" || e.dataSeries.visible);
    chartArchive.render();
}

function toogleFlowDataSeries(e) {
    e.dataSeries.visible = !(typeof (e.dataSeries.visible) === "undefined" || e.dataSeries.visible);
    chartFlows.render();
}

chartRT = new CanvasJS.Chart("rtchartContainer", {
    animationEnabled: false,
    zoomEnabled: true,
    exportEnabled: true,
    exportFileName: "chart",
    title: {
        text: "Real-time data",
    },
    subtitles: [{text: "Data available only " + openingTime}],
    axisY: {
        title: "# People",
        includeZero: false
    },
    axisX: {
        valueFormatString: "hh:mm:ss TT",
        labelAngle: -30
    },
    toolTip: {
        shared: true
    },
    legend: {
        cursor: "pointer",
        verticalAlign: "top",
        horizontalAlign: "center",
        dockInsidePlotArea: false,
        itemclick: toogleRTDataSeries
    },
    data: rtdataDefinitions
});

chartFlows = new CanvasJS.Chart("flowchartContainer", {
    animationEnabled: false,
    zoomEnabled: true,
    exportEnabled: true,
    exportFileName: "chart",
    title: {
        text: "Flow data",
    },
    subtitles: [{text: "Data valid only " + openingTime}],
    axisY: {
        title: "# People",
        scaleBreaks: {
            autoCalculate: true
        }
    },
    axisX: {
        valueFormatString: "DD MMM hh:mm:ss TT",
        labelAngle: -30,
    },
    toolTip: {
        shared: true
    },
    legend: {
        cursor: "pointer",
        verticalAlign: "top",
        horizontalAlign: "center",
        dockInsidePlotArea: false,
        itemclick: toogleFlowDataSeries
    },
    data: flowdataDefinitions
});

chartArchive = new CanvasJS.Chart("archivechartContainer", {
    animationEnabled: false,
    zoomEnabled: true,
    exportEnabled: true,
    exportFileName: "chart",
    title: {
        text: "Archive data",
    },
    subtitles: [{text: "Data valid only " + openingTime}],
    axisY: {
        title: "# People",
        // scaleBreaks: {
        //     autoCalculate: true
        // },
        includeZero: false
    },
    axisX: {
        // scaleBreaks: {
        //     autoCalculate: true
        // },
        valueFormatString: "DD MMM hh:mm:ss TT",
        labelAngle: -30,
    },
    toolTip: {
        shared: true
    },
    legend: {
        cursor: "pointer",
        verticalAlign: "top",
        horizontalAlign: "center",
        dockInsidePlotArea: false,
        itemclick: toogleArchiveDataSeries
    },
    data: archivedataDefinitions
});

chartArchive.render();
chartRT.render();
chartFlows.render();

function drawSpace(rawspaces) {

    let spaces = [];
    let draw = SVG('svgimage');
    let plan;
    let selectDisplay = document.getElementById("displayoption");
    let graphType = document.getElementById("datatypes");
    let flowFree = true;
    for (let i = 0; i < rawspaces.length; i++) {
        spaces[i] = rawspaces[i]["spacename"]
    }
    let selectSpace = document.getElementById("spacename");


    function setDisplay() {
        let myindex = selectDisplay.selectedIndex;
        let SelValue = selectDisplay.options[myindex].value;
        // console.log(myindex, SelValue);
        resetcanvas(false);
        switch (SelValue) {
            case "Plan":
                document.getElementById("svgimage").style.display = "block";
                document.getElementById("rtvalues").style.display = "table";
                document.getElementById("rtchartContainer").style.display = "none";
                document.getElementById("archivechartContainer").style.display = "none";
                document.getElementById("flowchartContainer").style.display = "none";
                document.getElementById("datatypes").style.display = "none";
                document.getElementById("picker").style.display = "none";
                document.getElementById("pickerDataset").style.display = "none";
                document.getElementById("gen").style.display = "none";
                document.getElementById("graphdata").style.display = "none";
                break;
            case "Graphs":
                document.getElementById("svgimage").style.display = "none";
                document.getElementById("rtvalues").style.display = "none";
                document.getElementById("rtchartContainer").style.display = "block";
                document.getElementById("archivechartContainer").style.display = "none";
                document.getElementById("flowchartContainer").style.display = "none";
                document.getElementById("datatypes").style.display = "block";
                document.getElementById("picker").style.display = "none";
                document.getElementById("pickerDataset").style.display = "none";
                document.getElementById("gen").style.display = "none";
                document.getElementById("graphdata").style.display = "none";
                graphType.selectedIndex = 0;
                chartRT.render();
                break;
            case "Reporting":
                document.getElementById("rtchartContainer").style.display = "none";
                document.getElementById("archivechartContainer").style.display = "none";
                document.getElementById("flowchartContainer").style.display = "none";
                document.getElementById("datatypes").style.display = "none";
                document.getElementById("rtvalues").style.display = "none";
                document.getElementById("svgimage").style.display = "block";
                document.getElementById("picker").style.display = "block";
                document.getElementById("pickerDataset").style.display = "block";
                document.getElementById("gen").style.display = "block";
                document.getElementById("graphdata").style.display = "none";
                break;
            default:
                // this should never happen
                console.log("drawing.js went into an illegal state on the display option");
        }
    }

    function setGraph() {
        let myindex = graphType.selectedIndex;
        let SelValue = graphType.options[myindex].value;
        // console.log(myindex, SelValue);
        switch (SelValue) {
            case "Real-time":
                document.getElementById("svgimage").style.display = "none";
                document.getElementById("rtvalues").style.display = "none";
                document.getElementById("rtchartContainer").style.display = "block";
                document.getElementById("archivechartContainer").style.display = "none";
                document.getElementById("flowchartContainer").style.display = "none";
                document.getElementById("datatypes").style.display = "block";
                document.getElementById("picker").style.display = "none";
                document.getElementById("gen").style.display = "none";
                document.getElementById("graphdata").style.display = "none";
                chartRT.render();
                break;
            case "Archive":
                document.getElementById("svgimage").style.display = "none";
                document.getElementById("rtvalues").style.display = "none";
                document.getElementById("rtchartContainer").style.display = "none";
                document.getElementById("archivechartContainer").style.display = "block";
                document.getElementById("flowchartContainer").style.display = "none";
                document.getElementById("datatypes").style.display = "block";
                document.getElementById("picker").style.display = "block";
                document.getElementById("gen").style.display = "none";
                document.getElementById("graphdata").style.display = "block";
                chartArchive.render();
                break;
            case "Flows":
                document.getElementById("svgimage").style.display = "none";
                document.getElementById("rtvalues").style.display = "none";
                document.getElementById("rtchartContainer").style.display = "none";
                document.getElementById("archivechartContainer").style.display = "none";
                document.getElementById("flowchartContainer").style.display = "block";
                document.getElementById("datatypes").style.display = "block";
                document.getElementById("picker").style.display = "none";
                document.getElementById("gen").style.display = "none";
                document.getElementById("graphdata").style.display = "none";
                chartFlows.render();
                // flowWarning = true;
                break;
            default:
                // this should never happen
                console.log("drawing.js went into an illegal state on the graph data option");
        }
    }

    function resetcanvas(full) {
        if (full) {
            spacename = "";
            readPlan("logo", true);
        }
        document.getElementById("lastts").innerText = "";
        for (let i = 0; i < allmeasurements.length; i++) {
            document.getElementById(allmeasurements[i].name).innerText = "----"
        }
    }

    function readPlan(name, od) {
        $.ajax({
            type: 'GET',
            url: ip + "/plan/" + name,
            success: function (data) {
                let planDataRaw = JSON.parse(data);
                plan = draw.svg(planDataRaw["qualifier"]);
                if (!od) {
                    spacename = name;
                }
            },
            error: function (error) {
                alert("Error " + error);
            }

        });
    }

    for (let i = 0; i < spaces.length; i++) {
        let opt = spaces[i];
        if (opt in aliasSpacenames) {
            opt = aliasSpacenames[opt]
        }
        let el = document.createElement("option");
        el.textContent = opt;
        el.value = opt;
        selectSpace.appendChild(el);
    }

    resetcanvas(true);

    selectSpace.onchange = function () {
        let myindex = selectSpace.selectedIndex,
            SelValue = selectSpace.options[myindex].value;
        spacenameUncoded = SelValue;
        const keys = Object.keys(aliasSpacenames);
        for (let i = 0; i < keys.length; i++) {
            if (SelValue === aliasSpacenames[keys[i]]) {
                SelValue = keys[i];
                break
            }
        }
        if (plan != null) {
            plan.clear();
            selectDisplay.selectedIndex = 0
        }
        // need to clean the canvas and all graphs when the value changes
        if (SelValue !== "Choose a space") {
            let currentTime = new Date();
            chartRT.options.exportFileName = currentTime.getFullYear().toString() + "_" + (currentTime.getMonth() + 1).toString() + "_" +
                currentTime.getDate().toString() + "_" + spacenameUncoded.replace(/ /g,"_")  + "_RealTime";
            for (let i = 0; i < dataArrays.length; i++) {
                dataArrays[i].length = 0;
            }
            chartRT.render();
            chartArchive.options.exportFileName = currentTime.getFullYear().toString() + "_" + (currentTime.getMonth() + 1).toString() + "_" +
                currentTime.getDate().toString() + "_" + spacenameUncoded.replace(/ /g,"_")  + "_Archive";
            chartArchive.options.title.text = "Archive data: " + spacenameUncoded;
            for (let i = 0; i < dataArraysArchive.length; i++) {
                dataArraysArchive[i].length = 0;
            }
            chartArchive.render();
            chartFlows.options.exportFileName = currentTime.getFullYear().toString() + "_" + (currentTime.getMonth() + 1).toString() + "_" +
                currentTime.getDate().toString() + "_" + spacenameUncoded.replace(/ /g,"_")  + "Flows";
            chartFlows.options.title.text = "Real Time Flow data: " + spacenameUncoded;
            for (let i = 0; i < dataArraysFlow.length; i++) {
                dataArraysFlow[i].length = 0;
            }
            dataArraysFlow.length = 0;
            for (let i = 0; i < flowdataDefinitions.length; i++) {
                flowdataDefinitions[i].length = 0;
            }
            flowdataDefinitions.length = 0;
            chartFlows.render();
            flowFree = true;
            // flowWarning = true;
            readPlan(SelValue, false)
        } else {
            flowFree = false;
            resetcanvas(true)
        }
        setDisplay()
    };

    selectDisplay.onchange = setDisplay;

    graphType.onchange = setGraph;

    function updatedata() {
        let timeNow = new Date(),
            timeNowHS = ("0" + timeNow.getHours()).slice(-2) + ":" + ("0" + timeNow.getMinutes()).slice(-2),
            incycle = ((parseInt(opStartTime.replace(regex, ''), 10) < parseInt(timeNowHS.replace(regex, ''), 10))
                && (parseInt(timeNowHS.replace(regex, ''), 10) < parseInt(opEndTime.replace(regex, ''), 10)));

        if ((spacename !== "") && ((incycle) || (openingTime === ""))) {
            let urlv = ip + "/" + measurement.split("_")[0] + "/" + spacename;

            $.ajax({
                type: 'GET',
                timeout: 5000,
                url: urlv,
                success: function (rawdata) {
                    try {
                        let sampledata = JSON.parse(rawdata);
                        document.getElementById("lastts").innerText = new Date().toLocaleString();
                        let validData = {};
                        let currentTS = new Date();
                        let myindex = graphType.selectedIndex;
                        let SelValue = graphType.options[myindex].value;
                        for (let i = 0; i < sampledata.counters.length; i++) {
                            if (sampledata.counters[i].valid) {
                                let tag = sampledata.counters[i].counter.tag;
                                tag = tag.replace(/\_+/g, " ");
                                tag = tag.split(" ")[2];
                                // TODO for aliases
                                if (tag in aliasMeasurement) {
                                    tag = aliasMeasurement[tag]
                                }
                                if (tag in mapnames) {
                                    // console.log(tag);
                                    // servedData.push(tag);
                                    let index = mapnames[tag];
                                    // if (tag === "current") {
                                    // if (lastTS[index] === 0) {
                                    dataArrays[index].push({
                                        x: currentTS,
                                        y: sampledata.counters[i].counter.val
                                    });
                                    // console.log(dataArrays);
                                    validData[tag] = sampledata.counters[i].counter.val;
                                    for (let i = 0; i < allmeasurements.length; i++) {
                                        // let refTag = allmeasurements[i].name.substring(0, labellength);
                                        if (validData[allmeasurements[i].name]) {
                                            document.getElementById(allmeasurements[i].name).innerText = validData[allmeasurements[i].name];
                                            // console.log(allmeasurements[i].name, validData[refTag]);
                                        }
                                    }
                                }
                            }
                        }
                        if (SelValue === "Real-time") {
                            chartRT.render()
                        }
                        // console.log(validData);
                    } catch (e) {
                        console.log("updatedata failed: ", e)
                    }
                },
                error: function (error) {
                    // console.log("Failed to connect to update data");
                    // console.log(error);
                }

            });
        } else {
            if ((!incycle) && (openingTime !== "")) {
                for (let i = 0; i < dataArrays.length; i++) {
                    dataArrays[i].length = 0;
                }
                chartRT.render();
            }
        }
    }

    function updateFlow() {
        function loadCounter(tries) {
            $.ajax({
                type: 'GET',
                timeout: 5000,
                url: ip + "/sample/" + spacename + "/current",
                success: function (rawdata) {
                    try {
                        let sampledata = JSON.parse(rawdata);
                        loadEntries(sampledata, 0)
                    } catch (e) {
                        alert("received corrupted counter data");
                    }
                },
                error: function (error) {
                    if (tries === maxtries) {
                        alert("Network error.\n Please try again later.");
                        console.log("Error samples:" + error);
                    } else {
                        // console.log(error);
                        // loadsamples(header, api, entrieslist, tries + 1)
                        loadCounter(tries + 1)
                    }
                }

            });
        }

        function loadEntries(counter, tries) {
            $.ajax({
                type: 'GET',
                timeout: 10000,
                url: ip + "/entry/" + spacename + "/current",
                success: function (rawdata) {
                    // console.log(ip + "/entry/" + spacename + "/current")
                    try {
                        let sampledata = JSON.parse(rawdata);
                        // if ((sampledata.valid === false) && flowWarning) {
                        //     alert("Please enable entry data,\ninserting the authorisation pin.");
                        //     flowWarning = false
                        // } else {
                        // console.log(sampledata);
                        if (sampledata.valid === true) {
                            if ((sampledata.counter.entries !== undefined) && ((sampledata.counter.entries !== null))) {
                                if (flowdataDefinitions.length === 0) {
                                    // first sample, we need to set the graph fully
                                    dataArraysFlow.push([]);
                                    let tmpdef = {
                                        xValueFormatString: "DD MMM, YYYY @ hh:mm:ss TT",
                                        markerType: "none",
                                        name: "Total counter",
                                        connectNullData: true,
                                        showInLegend: true,
                                        xValueType: "dateTime",
                                        type: "stepLine",
                                        color: colors[0],
                                        dataPoints: dataArraysFlow[0]
                                    };
                                    flowdataDefinitions.push(tmpdef);
                                    for (let i = 0; i < sampledata.counter.entries.length; i++) {
                                        dataArraysFlow.push([]);
                                        dataArraysFlow.push([]);
                                        let tmpdefin = {
                                            xValueFormatString: "DD MMM, YYYY @ hh:mm:ss TT",
                                            name: "Flow-in entry: " + sampledata.counter.entries[i].id,
                                            connectNullData: true,
                                            showInLegend: true,
                                            xValueType: "dateTime",
                                            type: "stepLine",
                                            color: colors[(i + 1) % colors.length],
                                            markerType: "cross",
                                            dataPoints: dataArraysFlow[2 * i + 1]
                                        };
                                        flowdataDefinitions.push(tmpdefin);
                                        let tmpdefout = {
                                            xValueFormatString: "DD MMM, YYYY @ hh:mm:ss TT",
                                            name: "Flow-out entry: " + sampledata.counter.entries[i].id,
                                            connectNullData: true,
                                            showInLegend: true,
                                            xValueType: "dateTime",
                                            type: "stepLine",
                                            color: colors[(i + 1) % colors.length],
                                            markerType: "triangle",
                                            lineDashType: "dash",
                                            dataPoints: dataArraysFlow[2 * i + 2]
                                        };
                                        flowdataDefinitions.push(tmpdefout);
                                        // console.log(dataArraysFlow);
                                    }
                                    // console.log("initial definition", flowdataDefinitions, dataArraysFlow);
                                }

                                // console.log("extraction of data from ",counter,"and",sampledata.counter.entries);
                                // for (let j = 0; j < sampledata.counter.entries.length; j++) {
                                //     console.log(sampledata.counter.entries[j].in);
                                //     console.log(sampledata.counter.entries[j].out)
                                // }
                                // console.log(counter);
                                if (counter.valid && sampledata.valid) {
                                    dataArraysFlow[0].push({
                                        x: counter.counter.ts,
                                        y: counter.counter.val
                                    });
                                    for (let i = 0; i < sampledata.counter.entries.length; i++) {
                                        dataArraysFlow[2 * i + 1].push({
                                            x: sampledata.counter.ts,
                                            y: sampledata.counter.entries[i].in
                                        });
                                        dataArraysFlow[2 * i + 2].push({
                                            x: sampledata.counter.ts,
                                            y: sampledata.counter.entries[i].out
                                        });
                                    }
                                }
                                // console.log(counter, sampledata)
                                // console.log(flowdataDefinitions)
                                // console.log(dataArraysFlow);
                                chartFlows.render()
                            }
                        }
                        // }
                        flowFree = true
                    } catch (e) {
                        alert("received corrupted entry data or unauthorised access");
                        console.log(e)
                    }
                },
                error: function (error) {
                    if (tries === maxtries) {
                        alert("Network error.\n Please try again later.");
                        console.log("Error samples:" + error);
                    } else {
                        // console.log(error);
                        // loadsamples(header, api, entrieslist, tries + 1)
                        loadEntries(counter, tries + 1)
                    }
                }

            });
        }

        if (flowFree) {
            flowFree = false;
            // let regex = new RegExp(':', 'g');
            let timeNow = new Date(),
                timeNowHS = ("0" + timeNow.getHours()).slice(-2) + ":" + ("0" + timeNow.getMinutes()).slice(-2),
                incycle = ((parseInt(opStartTime.replace(regex, ''), 10) < parseInt(timeNowHS.replace(regex, ''), 10))
                    && (parseInt(timeNowHS.replace(regex, ''), 10) < parseInt(opEndTime.replace(regex, ''), 10)));

            if ((spacename !== "") && ((incycle) || (openingTime === ""))) {
                loadCounter(0)
            } else {
                if ((!incycle) && (openingTime !== "") && (dataArraysFlow.length > 0)) {
                    for (let i = 0; i < dataArraysFlow.length; i++) {
                        dataArraysFlow[i].length = 0;
                    }
                    chartFlows.render();
                }

            }
        }
    }

    setInterval(updatedata, repPeriod);
    // if (rtshow[0] === "dbg") {
    setInterval(updateFlow, repPeriod)
    // }

}

$(document).ready(function () {
        Date.prototype.getUnixTime = function () {
            return (this.getTime() / 1000 | 0) * 1000
        };
        // extract analysis information and set-up the data section
        (function () {
            $.ajax({
                type: 'GET',
                url: ip + "/asys",
                success: function (data) {
                    let jsObj = JSON.parse(data);
                    let rp = document.getElementById("reptype");
                    // console.log(jsObj);
                    if ((rtshow[0] !== "dbg") && (jsObj.length > 0)) {
                        repPeriod = parseInt(jsObj[0].qualifier, 10);
                        for (let i = 1; i < jsObj.length; i++) {
                            let tmp = parseInt(jsObj[i].qualifier, 10);
                            if (tmp < repPeriod) {
                                repPeriod = tmp
                            }
                        }
                        repPeriod -= 1;
                        repPeriod *= 1000;
                    }
                    // console.log(repPeriod);
                    if (overviewReport) {
                        let ch = document.createElement("option");
                        ch.textContent = "overview";
                        rp.appendChild(ch);
                    }
                    if (reportCurrent || (rtshow[0] === "dbg")) {
                        allmeasurements.push({"name": "current", "value": "0"});
                        let ch = document.createElement("option");
                        ch.textContent = "current";
                        rp.appendChild(ch);
                    }
                    // console.log(rtshow.length)
                    if (rtshow.length !== 0) {
                        for (let i = 0; i < jsObj.length; i++) {
                            let el = {"name": jsObj[i]["name"], "value": jsObj[i]["qualifier"]};
                            // TODO  here for aliases
                            if (el.name in aliasMeasurement) {
                                el.name = aliasMeasurement[el.name]
                            }
                            if ((rtshow.indexOf(jsObj[i]["name"]) > -1) || (rtshow[0] === "dbg")) {
                                allmeasurements.push(el);
                            }
                            if ((repshow.indexOf(jsObj[i]["name"]) > -1) || (rtshow[0] === "dbg")) {
                                let ch = document.createElement("option");
                                ch.textContent = el.name;
                                rp.appendChild(ch);
                            }
                        }
                    } else {
                        // document.getElementById("rttitle").style.visibility = "hidden";
                        document.getElementById("rttitle").className = 'hidden';
                        // document.getElementById("rtvalues").style.visibility = "hidden";
                        document.getElementById("rtvalues").className = 'hidden';
                        // document.getElementById("MyElement").classList.add('hidden');
                    }
                    if ((!repvisile && (rtshow[0] !== "dbg"))) {
                        // rp.style.visibility = "hidden";
                        rp.className = 'hidden';
                    }

                    let html = "";
                    for (let i = 0; i < allmeasurements.length; i++) {
                        html += "<tr>" +
                            "<td id=\'" + allmeasurements[i].name + "_" + "\'>" + allmeasurements[i].name + "</td>" +
                            "<td id=\'" + allmeasurements[i].name + "\'> n/a </td>";

                        dataArrays.push([]);
                        dataArraysArchive.push([]);
                        let tmprt = {
                            xValueFormatString: "DD MMM, YYYY @ hh:mm:ss TT",
                            markerType: "none",
                            name: allmeasurements[i].name,
                            connectNullData: true,
                            showInLegend: true,
                            xValueType: "dateTime",
                            type: "stepLine",
                            dataPoints: dataArrays[i]
                        };
                        let tmparch = {
                            xValueFormatString: "DD MMM, YYYY @ hh:mm:ss TT",
                            markerType: "none",
                            connectNullData: true,
                            name: allmeasurements[i].name,
                            showInLegend: true,
                            xValueType: "dateTime",
                            type: "stepLine",
                            dataPoints: dataArraysArchive[i]
                        };
                        rtdataDefinitions.push(tmprt);
                        archivedataDefinitions.push(tmparch);
                        mapnames[allmeasurements[i].name] = i;
                        // lastTS.push(0);
                    }

                    // console.log(mapnames);
                    $("#analysis").html(html);
                },
                error: function (jqXhr) {
                    alert("Failed to connect to ASYS API");
                    console.log(jqXhr);
                }

            });
        })();

        // extract space information and set-up the canvas and selection menu
        (function () {
            $.ajax({
                type: 'GET',
                url: ip + "/info",
                success: function (data) {
                    let spaces = JSON.parse(data);
                    drawSpace(spaces)
                },
                error: function (jqXhr) {
                    alert("Failed to connect to INFO API");
                    console.log(jqXhr);
                }

            });
        })();


    }
);

