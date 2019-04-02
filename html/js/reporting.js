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

    function displayDate() {
        if ((startDate !== undefined) && (endDate !== undefined)) {
            console.log("start: " + startDate);
            console.log("start: " + startDate.getUnixTime());
            endDate.setHours(endDate.getHours() + 23);
            endDate.setMinutes(endDate.getMinutes() + 59);
            // console.log("end: " + Date.now());
            if (endDate.getUnixTime() > Date.now()) {
                console.log("end: Now");
                console.log("end: " + Date.now());
            } else {
                console.log("end: " + endDate);
                console.log("end: " + endDate.getUnixTime());
            }
        }
    }
});