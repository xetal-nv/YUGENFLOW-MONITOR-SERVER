import sys, json

if len(sys.argv) == 2:
    receivedData = json.loads(sys.argv[1].replace("'", '"'))
    print("Space " + receivedData["space"] + " has counter " + receivedData["qualifier"] +
          " equal to " + str(receivedData["value"]))