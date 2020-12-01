package exportManager

import (
	"encoding/json"
	"fmt"
	"gateserver/dataformats"
	"gateserver/support/globals"
	"github.com/fpessolano/mlogger"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"
)

func customScripting(rst chan interface{}, chActual, chReferences chan dataformats.MeasurementSample) {

	defer func() {
		if v := recover(); v != nil {
			log.Println("capture a panic:", v)
			os.Exit(0)
		}
	}()

	var actualData = false
	snapshots := make(map[string]dataformats.MeasurementSample)
finished:
	for {
		var data dataformats.MeasurementSample
		select {
		case <-rst:
			// TODO save and reload snapshots?
			fmt.Println("Closing exportManager.customScripting")
			time.Sleep(time.Duration(globals.SettleTime) * time.Second)
			rst <- nil
			break finished
		case data = <-chActual:
			actualData = true
		case data = <-chReferences:
			actualData = false
		}
		// snapshots are managed only for actual data
		if actualData {
			// TODO add flows
			fmt.Printf("data -> %+v\n", data)
			// in and out flows are accumulated only for actual data
			newSnapshotSpace := dataformats.MeasurementSample{}
			if snapshotSpace, ok := snapshots[data.Space]; !ok {
				// a deep copy of the first data in the snapshot cache
				newSnapshotSpace.Ts = data.Ts
				newSnapshotSpace.Val = data.Val
				newSnapshotSpace.Qualifier = data.Qualifier
				newSnapshotSpace.Space = data.Space
				newSnapshotSpace.Flows = make(map[string]dataformats.EntryState)
				newSnapshotSpace.FlowIn = 0
				newSnapshotSpace.FlowOut = 0
				for entry, entryFlow := range data.Flows {
					entryFlowSnapshot := dataformats.EntryState{
						Id:         entryFlow.Id,
						Ts:         entryFlow.Ts,
						Netflow:    entryFlow.Variation,
						TsOverflow: 0,
						State:      entryFlow.State,
						Reversed:   entryFlow.State,
						Flows:      make(map[string]dataformats.Flow),
					}
					entryFlowSnapshot.FlowIn = 0
					entryFlowSnapshot.FlowOut = 0
					for gate, gateFlow := range entryFlow.Flows {
						gateFlowSnapshot := dataformats.Flow{
							Id:         gateFlow.Id,
							Netflow:    gateFlow.Variation,
							TsOverflow: 0,
							Reversed:   gateFlow.Reversed,
						}
						if gateFlow.Variation > 0 {
							gateFlowSnapshot.FlowIn = gateFlow.Variation
							entryFlowSnapshot.FlowIn += gateFlow.Variation
						} else {
							gateFlowSnapshot.FlowOut = gateFlow.Variation
							entryFlowSnapshot.FlowOut += gateFlow.Variation
						}
						entryFlowSnapshot.Flows[gate] = gateFlowSnapshot
					}
					//entryFlowSnapshot.FlowIn = EntryFlowIn
					//entryFlowSnapshot.FlowOut = EntryFlowOut
					newSnapshotSpace.FlowIn += entryFlowSnapshot.FlowIn
					newSnapshotSpace.FlowOut += entryFlowSnapshot.FlowOut
					newSnapshotSpace.Flows[entry] = entryFlowSnapshot
				}
				//snapshots[data.Space] = newSnapshotSpace
			} else {
				// the relevant snapshot needs to be updated
				// in case of overflow, flows are rebased and the overflow timestamp is updated
				// a deep copy of the data is made to which we add the values from the snapshot

				// TODO add overflow and reverse!

				// space level data is prepared
				newSnapshotSpace.Ts = data.Ts
				newSnapshotSpace.Val = data.Val
				newSnapshotSpace.Qualifier = data.Qualifier
				newSnapshotSpace.Space = data.Space
				newSnapshotSpace.Flows = make(map[string]dataformats.EntryState)
				newSnapshotSpace.FlowIn = snapshotSpace.FlowIn
				newSnapshotSpace.FlowOut = snapshotSpace.FlowOut

				// each entryName si updated in its flow, variation and count based on the received data
				for entryName, entry := range data.Flows {
					newEntrySnapshot := dataformats.EntryState{
						Id:         entry.Id,
						Ts:         entry.Ts,
						Variation:  entry.Variation,
						Netflow:    entry.Variation,
						TsOverflow: 0,
						State:      entry.State,
						Reversed:   entry.Reversed,
						Flows:      make(map[string]dataformats.Flow),
					}

					temporaryGateMap := make(map[string]dataformats.Flow)

					// the new entry data is updated with the old, if present
					if oldEntrySnapshot, found := snapshotSpace.Flows[entryName]; found {
						newEntrySnapshot.FlowIn = oldEntrySnapshot.FlowIn
						newEntrySnapshot.FlowOut = oldEntrySnapshot.FlowOut
						newEntrySnapshot.TsOverflow = oldEntrySnapshot.TsOverflow
						newEntrySnapshot.Netflow += oldEntrySnapshot.Netflow
						// temporaryGateMap is filled with deep copies
						for gateName, gate := range oldEntrySnapshot.Flows {
							temporaryGateMap[gateName] = dataformats.Flow{
								Id:         gate.Id,
								Variation:  gate.Variation,
								Netflow:    gate.Netflow,
								TsOverflow: gate.TsOverflow,
								FlowIn:     gate.FlowIn,
								FlowOut:    gate.FlowOut,
								Reversed:   gate.Reversed,
							}
						}
						//temporaryGateMap = newEntrySnapshot.Flows
					} else {
						newEntrySnapshot.FlowIn = 0
						newEntrySnapshot.FlowOut = 0
						newEntrySnapshot.TsOverflow = 0
						//temporaryGateMap = nil
					}

					// TODO add overflow and reverse!

					// new gate data is accumulated
					for gateName, gate := range entry.Flows {
						newGateSnapshot := dataformats.Flow{
							Id:         gate.Id,
							Variation:  gate.Variation,
							Netflow:    gate.Variation,
							TsOverflow: 0,
						}
						if gate.Variation > 0 {
							newGateSnapshot.FlowIn = gate.Variation
							newEntrySnapshot.FlowIn += gate.Variation
							newSnapshotSpace.FlowIn += gate.Variation
						} else {
							newGateSnapshot.FlowOut = gate.Variation
							newEntrySnapshot.FlowOut += gate.Variation
							newSnapshotSpace.FlowOut += gate.Variation
						}
						// verify if this gate is already being accumulated and update the new values
						if oldGateData, found := temporaryGateMap[gateName]; found {
							//gate.Netflow += oldGateData.Variation
							newGateSnapshot.FlowOut += oldGateData.FlowOut
							newGateSnapshot.FlowIn += oldGateData.FlowIn
							newGateSnapshot.TsOverflow = oldGateData.TsOverflow
							newGateSnapshot.Netflow += oldGateData.Netflow
						}
						//newGateSnapshot.Netflow = newGateSnapshot.FlowIn + newGateSnapshot.FlowOut

						newEntrySnapshot.Flows[gateName] = newGateSnapshot
					}
					//newEntrySnapshot.Netflow = newSnapshotSpace.FlowOut + newSnapshotSpace.FlowIn

					// TODO add overflow
					newSnapshotSpace.Flows[entryName] = newEntrySnapshot
				}
			}
			snapshots[data.Space] = newSnapshotSpace
		}
		fmt.Printf("%+v\n\n", snapshots[data.Space])

		// TODO remove false
		if encodedData, err := json.Marshal(data); err == nil && false {
			if globals.DebugActive {
				fmt.Printf("Export JSON: %v\n", strings.Replace(string(encodedData), "\"", "'", -1))
			}
			if globals.ExportAsync {
				cmd := exec.Command(globals.ExportActualCommand, globals.ExportActualArgument,
					strings.Replace(string(encodedData), "\"", "'", -1))
				err := cmd.Start()
				if err != nil {
					// script report error
					if globals.DebugActive {
						fmt.Println("Export script has failed:", err.Error())
					}
					mlogger.Error(globals.ExportManagerLog,
						mlogger.LoggerData{Id: "exportManager.customScripting",
							Message: "error exporting data ",
							Data:    []int{1}, Aggregate: true})
				}
			} else {
				cmd, err := exec.Command(globals.ExportActualCommand, globals.ExportActualArgument,
					strings.Replace(string(encodedData), "\"", "'", -1)).Output()
				if err != nil || len(cmd) != 0 {
					// script report error
					if globals.DebugActive {
						if err != nil {
							fmt.Println("Export script has failed:", err.Error())
						} else {
							fmt.Println("Export script reported failure:", string(cmd))
						}
					}
					mlogger.Error(globals.ExportManagerLog,
						mlogger.LoggerData{Id: "exportManager.customScripting",
							Message: "error exporting data ",
							Data:    []int{1}, Aggregate: true})
				}
			}
		}
	}
}
