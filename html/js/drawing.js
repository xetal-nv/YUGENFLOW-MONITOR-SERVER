var repvisile = (reportCurrent) || (repshow.length !== 0);

let spacename = "";
let measurement = "sample";
let allmeasurements = [];
let selel = null;
// let lastTS = [];

let regex = new RegExp(':', 'g');
let timeNow = new Date(),
    timeNowHS = ("0" + timeNow.getHours()).slice(-2) + ":" + ("0" + timeNow.getMinutes()).slice(-2),
    incycle = ((parseInt(opStartTime.replace(regex, ''), 10) < parseInt(timeNowHS.replace(regex, ''), 10))
        && (parseInt(timeNowHS.replace(regex, ''), 10) < parseInt(opEndTime.replace(regex, ''), 10)));

if ((!incycle) && (openingTime !== "")) {
    alert("!!! WARNING !!!\nReal-time data is only available " + openingTime + ".\nReporting is always available.\n")
}

let dataDefinitions = [];
let dataArrays = [];
// let dataPoints = [];
let mapnames = {};

function toogleDataSeries(e) {
    if (typeof (e.dataSeries.visible) === "undefined" || e.dataSeries.visible) {
        e.dataSeries.visible = false;
    } else {
        e.dataSeries.visible = true;
    }
    chart.render();
}

chart = new CanvasJS.Chart("chartContainer", {
    animationEnabled: false,
    zoomEnabled: true,
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
        itemclick: toogleDataSeries
    },
    // data: [
    //     {
    //         xValueFormatString: "hh:mm:ss TT",
    //         name: "Actual data",
    //         showInLegend: true,
    //         type: "stepLine",
    //         dataPoints: dataPoints
    //     }
    // ]
    data: dataDefinitions
});
chart.render();

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
        if (SelValue !== "Choose a space") readPlan(SelValue, false); else resetcanvas()
    };

    selectDisplay.onchange = function () {
        let myindex = selectDisplay.selectedIndex;
        let SelValue = selectDisplay.options[myindex].value;
        // console.log(myindex, SelValue);
        switch (SelValue) {
            case "Plan":
                document.getElementById("svgimage").style.display = "block";
                document.getElementById("rtvalues").style.display = "table";
                document.getElementById("chartContainer").style.display = "none";
                break;
            case "Graphs":
                document.getElementById("svgimage").style.display = "none";
                document.getElementById("rtvalues").style.display = "none";
                document.getElementById("chartContainer").style.display = "block";
                break;
            default:
                // this should never happen
                console.log("drawing.js went into an illegal state on the display option");
        }
        // if (SelValue !== "Choose a space") readPlan(SelValue, false); else resetcanvas()
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
                                    //     lastTS[index] = currentTS
                                    // } else if (lastTS[index] !== sampledata.counters[i].counter.ts) {
                                    //     dataArrays[index].push({
                                    //         x: new Date(sampledata.counters[i].counter.ts),
                                    //         y: sampledata.counters[i].counter.val
                                    //     });
                                    //     lastTS[index] = sampledata.counters[i].counter.ts
                                    // } else if (lastTS[index] === sampledata.counters[i].counter.ts) {
                                    //     dataArrays[index].push({
                                    //         x: currentTS,
                                    //         y: dataArrays[dataArrays.length - 1].y
                                    //     });
                                    //     // console.log(dataDefinitions);
                                    // }

                                    // }
                                    validData[tag] = sampledata.counters[i].counter.val;
                                    for (let i = 0; i < allmeasurements.length; i++) {
                                        let refTag = allmeasurements[i].name.substring(0, labellength);
                                        if (validData[refTag]) {
                                            document.getElementById(allmeasurements[i].name).innerText = validData[refTag];
                                            // console.log(allmeasurements[i].name, validData[refTag]);
                                        }
                                    }
                                } else {
                                    console.log("Received corrupted update data", rawdata)
                                }
                            }
                        }
                        // console.log(servedData);
                        // for (let j=0; j<allmeasurements.length; j++) {
                        //     if (!(allmeasurements[i].name in servedData)) {
                        //         // last sample is replicated
                        //         let index = mapnames[allmeasurements[i].name];
                        //         dataArrays[index].push({
                        //             x: currentTS,
                        //             y: dataArrays[dataArrays.length - 1].y
                        //         });
                        //     }
                        // }
                        chart.render();
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
        }
    }

    setInterval(updatedata, 4000)

}

$(document).ready(function () {
    // extract analysis information and set-up the data section
    (function () {
        $.ajax({
            type: 'GET',
            url: ip + "/asys",
            success: function (data) {
                let jsObj = JSON.parse(data);
                let rp = document.getElementById("reptype");
                // console.log(jsObj)
                if (overviewReport) {
                    let ch = document.createElement("option");
                    ch.textContent = "overview";
                    rp.appendChild(ch);
                }
                allmeasurements.push({"name": "current", "value": "0"});
                if (reportCurrent || (rtshow[0] === "dbg")) {
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
                    let tmp = {
                        xValueFormatString: "hh:mm:ss TT",
                        name: allmeasurements[i].name,
                        showInLegend: true,
                        xValueType: "dateTime",
                        type: "stepLine",
                        dataPoints: dataArrays[i]
                    };
                    dataDefinitions.push(tmp);
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


});

