var repvisile = (reportCurrent) || (repshow.length !== 0);

let spacename = "";
let measurement = "sample";
let allmeasurements = [];
let selel = null;

let regex = new RegExp(':', 'g');
let timeNow = new Date(),
    timeNowHS = ("0" + timeNow.getHours()).slice(-2) + ":" + ("0" + timeNow.getMinutes()).slice(-2),
    incycle = ((parseInt(opStartTime.replace(regex, ''), 10) < parseInt(timeNowHS.replace(regex, ''), 10))
        && (parseInt(timeNowHS.replace(regex, ''), 10) < parseInt(opEndTime.replace(regex, ''), 10)));

if ((!incycle) && (openingTime !== "")) {
    alert("!!! WARNING !!!\nReal-time data is only available " + openingTime + ".\nReporting is always available.\n")
}

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
    let select = document.getElementById("spacename");
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
                let jsObj = JSON.parse(data);
                plan = draw.svg(jsObj["qualifier"]);
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
                    let total = document.getElementById(name);
                    selel = total;
                    total.setAttribute("class", "st2");
                    total.onmousedown = function () {
                        // console.log("found " + name)
                        measurement = "sample";
                        if (selel != null) selel.setAttribute("class", "st1");
                        total.setAttribute("class", "st2");
                        selel = total;
                        for (let i = 0; i < allmeasurements.length; i++) {
                            document.getElementById(allmeasurements[i].name).innerText = "";
                            if (allmeasurements[i].name !== "current") {
                                document.getElementById(allmeasurements[i].name + "_").style.color = "black";
                            }
                        }
                    };
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
        select.appendChild(el);
    }

    resetcanvas();

    select.onchange = function () {
        var myindex = select.selectedIndex;
        var SelValue = select.options[myindex].value;
        if (plan != null) plan.clear();
        // need to remove the onclick events
        if (SelValue !== "Choose a space") readPlan(SelValue, false); else resetcanvas()
    };

    function updatedata() {
        let regex = new RegExp(':', 'g');
        let timeNow = new Date(),
            timeNowHS = ("0" + timeNow.getHours()).slice(-2) + ":" + ("0" + timeNow.getMinutes()).slice(-2),
            incycle = ((parseInt(opStartTime.replace(regex, ''), 10) < parseInt(timeNowHS.replace(regex, ''), 10))
                && (parseInt(timeNowHS.replace(regex, ''), 10) < parseInt(opEndTime.replace(regex, ''), 10)));

        if ((spacename !== "") && ((incycle) || (openingTime === ""))) {
            let urlv = ip + "/" + measurement.split("_")[0] + "/" + spacename + "/";
            // console.log(measurement)
            for (let i = 0; i < allmeasurements.length; i++) {

                (function () {
                    $.ajax({
                        type: 'GET',
                        url: urlv + allmeasurements[i].name,
                        success: function (data) {
                            let spaces = JSON.parse(data);
                            // console.log(data);
                            if (spaces["valid"]) {
                                // if (allmeasurements[i].name === "current") {
                                    // document.getElementById("lastts").innerText = timeConverter(spaces["counter"]["ts"]).toString();
                                    // console.log(new Date());
                                    document.getElementById("lastts").innerText = new Date().toLocaleString();
                                // }
                                let dt = "n/a";
                                // console.log(measurement)
                                let ms = measurement.split("_");
                                // console.log(ms);
                                switch (ms[0]) {
                                    case "sample":
                                        dt = spaces["counter"]["val"];
                                        break;
                                    case "entry":
                                        // console.log(data);
                                        // console.log(allmeasurements[i].name =="current");
                                        if (allmeasurements[i].name === "current") {
                                            if (spaces["counter"]["val"] !== null) {
                                                for (let i = 0; i < spaces["counter"]["val"].length; i++) {
                                                    if (spaces["counter"]["val"][i][0].toString() === ms[1]) {
                                                        dt = spaces["counter"]["val"][i][1];
                                                        break;
                                                    }
                                                }
                                            }
                                        }
                                        break;
                                    default:
                                        break;
                                }
                                // console.log(dt);
                                // in case of corrupted JSON we skip uopating the page
                                if (/^-{0,1}\d+$/.test(dt)) {
                                    document.getElementById(allmeasurements[i].name).innerText = dt;
                                }
                            }
                        },
                        error: function (jqXhr, textStatus, error) {
                            console.log("Failed to connect to update data");
                            console.log(jqXhr);
                        }
                    });
                })();

            }
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
                if (reportCurrent) {
                    let ch = document.createElement("option");
                    allmeasurements.push({"name": "current", "value": "0"});
                    ch.textContent = "current";
                    rp.appendChild(ch);
                }
                for (let i = 0; i < jsObj.length; i++) {
                    let el = {"name": jsObj[i]["name"], "value": jsObj[i]["qualifier"]};
                    if (rtshow.indexOf(el.name) > -1) {
                        allmeasurements.push(el);
                    }
                    if (repshow.indexOf(el.name) > -1) {
                        let ch = document.createElement("option");
                        ch.textContent = jsObj[i]["name"];
                        rp.appendChild(ch);
                    }
                }
                if (!repvisile) {
                    rp.style.visibility = "hidden";
                }

                let html = "";
                for (let i = 0; i < allmeasurements.length; i++) {
                    html += "<tr>" +
                        "<td id=\'" + allmeasurements[i].name + "_" + "\'>" + allmeasurements[i].name + "</td>" +
                        "<td id=\'" + allmeasurements[i].name + "\'> n/a </td>";
                }
                $("#analysis").html(html);
            },
            error: function (jqXhr, textStatus, error) {
                alert("Failed to connect to ASYS API");
                console.log(jqXhr);
                // console.log(textStatus);
                // console.log(error);
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
            error: function (jqXhr, textStatus, error) {
                alert("Failed to connect to INFO API");
                console.log(jqXhr);
            }

        });
    })();


});

