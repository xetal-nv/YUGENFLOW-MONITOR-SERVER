var ip = "http://localhost:8090";
var samplingWindow = 5 * 1000;
var reportCurrent = false;
var user = "DEME";
var spaceTimes = {
"livlab": ["03:00", "05:00"],
};
var labellength = 8;
var overviewReport = true;
let overviewReportDefs = [{name: "activity 18:00 to 06:00?", start: "18:00", end: "06:00", point: "", precision: "", presence: "night", id: 0},
{name: "08:00 to 12:00", start: "08:00", end: "12:00", point: "", precision: "", presence: "", id: 0},
{name: "at 10:00", start: "", end: "", point: "10:00", precision: "30", presence: "", id: 0},
{name: "activity 08:00 to 12:30?", start: "08:00", end: "12:30", point: "", precision: "", presence: "morning", id: 0},
{name: "13:00 to 17:00", start: "13:00", end: "17:00", point: "", precision: "", presence: "", id: 0},
{name: "at 14:00", start: "", end: "", point: "14:00", precision: "30", presence: "", id: 0},
{name: "activity 13:00 to 18:00?", start: "13:00", end: "18:00", point: "", precision: "", presence: "afternoon", id: 0},
{name: "day", start: "08:00", end: "18:00", point: "", precision: "", presence: "", id: 0, skip: true}];
var refOverviewAsys = "20secs";
var overviewSkipDays = [];
var rtshow = ["20secs", "20mins", "hour", "day"];
var repshow = "";
var openingTime = "";
var opStartTime = "";
var opEndTime = "";
