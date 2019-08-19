// Given an array is values, it returns an array [[in,out]...] of the data flow generating it
function extractFlow(counter, tsdivider) {
    let flow = [0, 0];
    let flowvector = [];

    flowvector.push([Math.trunc(counter[0][0]/tsdivider), counter[0][1], 0, 0]);

    for (let i = 1; i < counter.length; i++) {
        if (counter[i][1] > counter[i - 1][1]) {
            flow[0] += counter[i][1] - counter[i - 1][1]
            flowvector.push([Math.trunc(counter[i][0]/tsdivider), counter[i][1], flow[0], flow[1]]);
        } else {
            flow[1] += counter[i - 1][1] - counter[i][1]
            flowvector.push([Math.trunc(counter[i][0]/tsdivider), counter[i][1], flow[0], flow[1]]);
        }
        // console.log(counter[i][1], counter[i-1][1], flow.slice());
        // flowvector.push([Math.trunc(counter[i][0]/tsdivider), counter[i][1], flow[0], flow[1]]);
    }
    return flowvector
}