$(document).ready(function () {
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
            minDate: new Date(2000, 12, 31),
            maxDate: new Date(),
            onSelect: function () {
                startDate = this.getDate();
                updateStartDate();
            }
        }),
        endPicker = new Pikaday({
            field: document.getElementById('end'),
            minDate: new Date(2000, 12, 31),
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

    document.getElementById("gen").addEventListener("click", displayDate);

    // TODO HERE

    function displayDate() {
        let select = document.getElementById("spacename");
        var myindex = select.selectedIndex,
            space = select.options[myindex].value;
        select = document.getElementById("reptype");
        myindex = select.selectedIndex;
        var asys = select.options[myindex].value,
            start, end;
        if ((startDate !== undefined) && (endDate !== undefined)
            && (space !== "Choose a space") && (asys !== "Choose a dataset")) {
            start = startDate.getUnixTime();
            endDate.setHours(endDate.getHours() + 23);
            endDate.setMinutes(endDate.getMinutes() + 59);
            if (endDate.getUnixTime() > Date.now()) {
                end = Date.now();
            } else {
                end = endDate.getUnixTime();
            }
            let apipath = "/series?type=sample?space=" + space + "?analysis=" + asys + "?start=" + start + "?end=" + end;
            console.log(apipath);
            $.ajax({
                type: 'GET',
                url: ip + apipath,
                success: function (data) {
                    let jsObj = JSON.parse(data);
                    console.log(jsObj)
                },
                error: function (error) {
                    alert("Error " + error);
                }

            });
        }
    }
});