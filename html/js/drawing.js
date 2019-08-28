var repvisile = (reportCurrent) || (repshow.length !== 0);

let spacename = "";
let measurement = "sample";
let allmeasurements = [];
let selel = null;
let repPeriod = 4000;
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
let dataArrays = [];
let dataArraysArchive = [];
// let dataPoints = [];
let mapnames = {};

function toogleRTDataSeries(e) {
    if (typeof (e.dataSeries.visible) === "undefined" || e.dataSeries.visible) {
        e.dataSeries.visible = false;
    } else {
        e.dataSeries.visible = true;
    }
    chartRT.render();
}

function toogleArchiveDataSeries(e) {
    if (typeof (e.dataSeries.visible) === "undefined" || e.dataSeries.visible) {
        e.dataSeries.visible = false;
    } else {
        e.dataSeries.visible = true;
    }
    chartArchive.render();
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
        // scaleBreaks: {
        //     autoCalculate: true
        // },
        includeZero: false
    },
    axisX: {
        // scaleBreaks: {
        //     autoCalculate: true
        // }
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

function timeConverter(UNIX_timestamp) {
    let a = new Date(UNIX_timestamp);
    let months = ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec'];
    let year = a.getFullYear();
    let month = months[a.getMonth()];
    let date = a.getDate();
    let hour = a.getHours();
    let min = a.getMinutes();
    let sec = a.getSeconds();
    return date + ' ' + month + ' ' + year + ' ' + hour + ':' + min + ':' + sec;
}

function drawSpace(rawspaces) {

    let spaces = [];
    let draw = SVG('svgimage');
    let plan;
    let selectSpace = document.getElementById("spacename");
    let selectDisplay = document.getElementById("displayoption");
    let graphType = document.getElementById("datatypes");
    for (i = 0; i < rawspaces.length; i++) {
        spaces[i] = rawspaces[i]["spacename"]
    }

    function resetcanvas() {
        spacename = "";
        readPlan("logo", true);
        document.getElementById("lastts").innerText = "";
        for (let i = 0; i < allmeasurements.length; i++) {
            document.getElementById(allmeasurements[i].name).innerText = ""
        }
        selel = null;
    }

    function readPlan(name, od) {
        $.ajax({
            type: 'GET',
            url: ip + "/plan/" + name,
            success: function (data) {
                let planDataRaw = JSON.parse(data);
                // console.log(planDataRaw);
                plan = draw.svg(planDataRaw["qualifier"]);
                if (!od) {
                    spacename = name;
                    for (let i = 0; i < rawspaces.length; i++) {
                        // console.log(rawspaces[i])
                        if (rawspaces[i]["spacename"] === name) {
                            for (let j = 0; j < rawspaces[i]["entries"].length; j++) {
                                let nm = rawspaces[i]["entries"][j]["entryid"];
                                let el = document.getElementById(nm);
                                if (el != null) {
                                    el.onmousedown = function () {
                                        // console.log("found " + nm);
                                        measurement = "entry_" + nm;
                                        if (selel != null) selel.setAttribute("class", "st1");
                                        el.setAttribute("class", "st2");
                                        selel = el;
                                        for (let i = 0; i < allmeasurements.length; i++) {
                                            document.getElementById(allmeasurements[i].name).innerText = "";
                                            if (allmeasurements[i].name !== "current") {
                                                document.getElementById(allmeasurements[i].name + "_").style.color = "lightgray";
                                            }
                                        }
                                    };
                                }
                            }
                            break;
                        }
                    }
                    // let total = document.getElementById(name);
                    // selel = total;
                    // total.setAttribute("class", "st2");
                    // total.onmousedown = function () {
                    //     // console.log("found " + name)
                    //     measurement = "sample";
                    //     if (selel != null) selel.setAttribute("class", "st1");
                    //     total.setAttribute("class", "st2");
                    //     selel = total;
                    //     for (let i = 0; i < allmeasurements.length; i++) {
                    //         document.getElementById(allmeasurements[i].name).innerText = "";
                    //         if (allmeasurements[i].name !== "current") {
                    //             document.getElementById(allmeasurements[i].name + "_").style.color = "black";
                    //         }
                    //     }
                    // };
                }
            },
            error: function (error) {
                alert("Error " + error);
            }

        });
    }

    for (let i = 0; i < spaces.length; i++) {
        // console.log(spaces[i])
        let opt = spaces[i];
        let el = document.createElement("option");
        el.textContent = opt;
        el.value = opt;
        selectSpace.appendChild(el);
    }

    resetcanvas();

    selectSpace.onchange = function () {
        let myindex = selectSpace.selectedIndex;
        let SelValue = selectSpace.options[myindex].value;
        if (plan != null) {
            plan.clear();
            selectDisplay.selectedIndex = 0
        }
        // need to remove the onclick events
        if (SelValue !== "Choose a space") {
            let currentTime = new Date();
            chartRT.options.exportFileName = currentTime.getFullYear().toString() + "_" + (currentTime.getMonth() + 1).toString() + "_" +
                currentTime.getDate().toString() + "_" + SelValue + "_RealTime";
            chartRT.render();
            chartArchive.options.exportFileName = currentTime.getFullYear().toString() + "_" + (currentTime.getMonth() + 1).toString() + "_" +
                currentTime.getDate().toString() + "_" + SelValue + "_Archive";
            chartArchive.options.title.text = "Archive data: " + SelValue;
            chartArchive.render();
            readPlan(SelValue, false)
        } else {
            resetcanvas()
        }
    };

    selectDisplay.onchange = function () {
        let myindex = selectDisplay.selectedIndex;
        let SelValue = selectDisplay.options[myindex].value;
        // console.log(myindex, SelValue);
        switch (SelValue) {
            case "Plan":
                document.getElementById("svgimage").style.display = "block";
                document.getElementById("rtvalues").style.display = "table";
                document.getElementById("rtchartContainer").style.display = "none";
                document.getElementById("archivechartContainer").style.display = "none";
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
    };

    graphType.onchange = function () {
        let myindex = graphType.selectedIndex;
        let SelValue = graphType.options[myindex].value;
        // console.log(myindex, SelValue);
        switch (SelValue) {
            case "Real-time":
                document.getElementById("svgimage").style.display = "none";
                document.getElementById("rtvalues").style.display = "none";
                document.getElementById("rtchartContainer").style.display = "block";
                document.getElementById("archivechartContainer").style.display = "none";
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
                document.getElementById("datatypes").style.display = "block";
                document.getElementById("picker").style.display = "block";
                document.getElementById("gen").style.display = "none";
                document.getElementById("graphdata").style.display = "block";
                chartArchive.render();
                break;
            default:
                // this should never happen
                console.log("drawing.js went into an illegal state on the graph data option");
        }
    };

    function updatedata() {
        let regex = new RegExp(':', 'g');
        let timeNow = new Date(),
            timeNowHS = ("0" + timeNow.getHours()).slice(-2) + ":" + ("0" + timeNow.getMinutes()).slice(-2),
            incycle = ((parseInt(opStartTime.replace(regex, ''), 10) < parseInt(timeNowHS.replace(regex, ''), 10))
                && (parseInt(timeNowHS.replace(regex, ''), 10) < parseInt(opEndTime.replace(regex, ''), 10)));

        if ((spacename !== "") && ((incycle) || (openingTime === ""))) {
            let urlv = ip + "/" + measurement.split("_")[0] + "/" + spacename;
            // console.log(urlv);

            $.ajax({
                type: 'GET',
                timeout: 5000,
                url: urlv,
                success: function (rawdata) {
                    try {
                        let sampledata = JSON.parse(rawdata);
                        document.getElementById("lastts").innerText = new Date().toLocaleString();
                        // console.log("DEBUG", sampledata.counters);
                        let validData = {};
                        // let servedData = [];
                        let currentTS = new Date();
                        let myindex = graphType.selectedIndex;
                        let SelValue = graphType.options[myindex].value;
                        for (let i = 0; i < sampledata.counters.length; i++) {
                            if (sampledata.counters[i].valid) {
                                let tag = sampledata.counters[i].counter.tag;
                                tag = tag.replace(/\_+/g, " ");
                                tag = tag.split(" ")[2];
                                // console.log(tag);
                                if (tag in mapnames) {
                                    // servedData.push(tag);
                                    let index = mapnames[tag];
                                    // if (tag === "current") {
                                    // if (lastTS[index] === 0) {
                                    dataArrays[index].push({
                                        x: currentTS,
                                        y: sampledata.counters[i].counter.val
                                    });
                                    validData[tag] = sampledata.counters[i].counter.val;
                                    for (let i = 0; i < allmeasurements.length; i++) {
                                        let refTag = allmeasurements[i].name.substring(0, labellength);
                                        if (validData[refTag]) {
                                            document.getElementById(allmeasurements[i].name).innerText = validData[refTag];
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

    setInterval(updatedata, repPeriod)

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
                            if ((rtshow.indexOf(el.name) > -1) || (rtshow[0] === "dbg")) {
                                allmeasurements.push(el);
                            }
                            if ((repshow.indexOf(el.name) > -1) || (rtshow[0] === "dbg")) {
                                let ch = document.createElement("option");
                                ch.textContent = jsObj[i]["name"];
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

