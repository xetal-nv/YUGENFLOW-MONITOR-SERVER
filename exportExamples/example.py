import sys, json

# example exporting 'current' data

if len(sys.argv) == 2:
    try:
        receivedData = json.loads(sys.argv[1].replace("'", '"'))
        with open('./exportedData.txt', 'a') as f:
            f.write("Space " + receivedData["space"] + " has counter " + receivedData["qualifier"] +
                    " equal to " + str(receivedData["count"]) + " at time " + str(receivedData["timestamp"]) + "\n")
    except:
        print("failed")
