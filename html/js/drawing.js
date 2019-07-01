let spacename = "";
let measurement = "sample";
let allmeasurements = [{"name": "current", "value": "0"}];
let selel = null;

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
        document.getElementById("lastts").innerText = "n/a";
        for (let i = 0; i < allmeasurements.length; i++) {
            document.getElementById(allmeasurements[i].name).innerText = "n/a"
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
        if (spacename !== "") {
            let urlv = ip + "/" + measurement.split("_")[0] + "/" + spacename + "/";
            // console.log(allmeasurements)
            for (let i = 0; i < allmeasurements.length; i++) {

                (function () {
                    $.ajax({
                        type: 'GET',
                        url: urlv + allmeasurements[i].name,
                        success: function (data) {
                            let spaces = JSON.parse(data);
                            // console.log(data);
                            if (spaces["valid"]) {
                                if (allmeasurements[i].name === "current") {
                                    document.getElementById("lastts").innerText = timeConverter(spaces["counter"]["ts"]).toString();
                                }
                                let dt = "n/a";
                                // console.log(measurement)
                                let ms = measurement.split("_");
                                // console.log(ms);
                                switch (ms[0]) {
                                    case "sample":
                                        dt = spaces["counter"]["val"];
                                        // console.log("sample");
                                        break;
                                    case "entry":
                                        // console.log(data);
                                        if (spaces["counter"]["val"] !== null) {
                                            for (let i = 0; i < spaces["counter"]["val"].length; i++) {
                                                if (spaces["counter"]["val"][i][0].toString() === ms[1]) {
                                                    dt = spaces["counter"]["val"][i][1];
                                                    break;
                                                }
                                            }
                                        }
                                        // console.log("entry");
                                        break;
                                    default:
                                        break;
                                }
                                // in case of corrupted JSON we skip uodating the page
                                if (/^\d+$/.test(dt)) {
                                    document.getElementById(allmeasurements[i].name).innerText = dt;
                                } else {
                                    console.log(spaces)
                                }
                            }
                        },
                        error: function (error) {
                        }

                    });
                })();

            }
        }
    }

    setInterval(updatedata, 300)

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
                let ch = document.createElement("option");
                ch.textContent = "current";
                rp.appendChild(ch);
                for (let i = 0; i < jsObj.length; i++) {
                    let el = {"name": jsObj[i]["name"], "value": jsObj[i]["qualifier"]};
                    allmeasurements.push(el);
                    let ch = document.createElement("option");
                    ch.textContent = jsObj[i]["name"];
                    rp.appendChild(ch);
                }
                let html = "";
                for (let i = 0; i < allmeasurements.length; i++) {
                    html += "<tr>" +
                        "<td>" + allmeasurements[i].name + "</td>" +
                        "<td id=\'" + allmeasurements[i].name + "\'> n/a </td>";
                }
                $("#analysis").html(html);
            },
            error: function (error) {
                alert("Error " + error);
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
            error: function (error) {
                alert("Error " + error);
            }

        });
    })();


});

