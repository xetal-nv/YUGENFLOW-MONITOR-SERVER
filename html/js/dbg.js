rtshow = ["dbg"];

function sendPin() {
    let pin = document.getElementById("pin").value;

    $.ajax({
        type: 'GET',
        timeout: 5000,
        url: ip + "/command?pin=" + pin,
        success: function (rawdata) {
            try {
                let sampledata = JSON.parse(rawdata);
                if (sampledata.State === false) {
                    alert("Wrong pin submitted")
                }
            } catch (e) {
                console.log("received corrupted data: ", rawdata)
            }
        },
        error: function (error) {
            // console.log("Failed to send pin to update data");
            // console.log(error);
        }
    });
    // return false
}