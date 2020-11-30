package exportManager

import (
	"encoding/json"
	"fmt"
	"gateserver/dataformats"
	"gateserver/support/globals"
	"github.com/fpessolano/mlogger"
	"os/exec"
	"strings"
	"time"
)

func customScripting(rst chan interface{}, chActual, chReferences chan dataformats.MeasurementSample) {
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
						Count:      entryFlow.Count,
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
							Netflow:    gateFlow.Netflow,
							TsOverflow: 0,
						}
						if gateFlow.Netflow > 0 {
							gateFlowSnapshot.FlowIn = gateFlow.Netflow
							entryFlowSnapshot.FlowIn += gateFlow.Netflow
						} else {
							gateFlowSnapshot.FlowOut = gateFlow.Netflow
							entryFlowSnapshot.FlowOut += gateFlow.Netflow
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

				// TODO verify and add overflow
				// TODO bug gate flow is wrong and we need to consider the reverse state !!!
				newSnapshotSpace.Ts = data.Ts
				newSnapshotSpace.Val = data.Val + snapshotSpace.Val
				newSnapshotSpace.Qualifier = data.Qualifier
				newSnapshotSpace.Space = data.Space
				newSnapshotSpace.Flows = make(map[string]dataformats.EntryState)
				newSnapshotSpace.FlowIn = snapshotSpace.FlowIn
				newSnapshotSpace.FlowOut = snapshotSpace.FlowOut

				// TODO verify and add overflow
				for entry, entryFlow := range data.Flows {
					entryFlowSnapshot := dataformats.EntryState{
						Id:         entryFlow.Id,
						Ts:         entryFlow.Ts,
						Count:      entryFlow.Count,
						TsOverflow: 0,
						State:      entryFlow.State,
						Reversed:   entryFlow.State,
						Flows:      make(map[string]dataformats.Flow),
					}

					var snapshotEntryGates map[string]dataformats.Flow

					if snapshotEntry, found := snapshotSpace.Flows[entry]; found {
						entryFlowSnapshot.FlowIn = snapshotEntry.FlowIn
						entryFlowSnapshot.FlowOut = snapshotEntry.FlowOut
						entryFlowSnapshot.TsOverflow = snapshotEntry.TsOverflow
						entryFlowSnapshot.Count += snapshotEntry.Count
						// TODO needs to be a deep copy !!
						for name, data := range entryFlowSnapshot.Flows {
							snapshotEntryGates[name] = dataformats.Flow{
								Id:         data.Id,
								Netflow:    data.Netflow,
								TsOverflow: data.TsOverflow,
								FlowIn:     data.FlowIn,
								FlowOut:    data.FlowOut,
							}
						}
						//snapshotEntryGates = entryFlowSnapshot.Flows
					} else {
						entryFlowSnapshot.FlowIn = 0
						entryFlowSnapshot.FlowOut = 0
						entryFlowSnapshot.TsOverflow = 0
						snapshotEntryGates = nil
					}

					// TODO verify and add overflow
					for gate, gateFlow := range entryFlow.Flows {
						gateFlowSnapshot := dataformats.Flow{
							Id:         gateFlow.Id,
							Netflow:    gateFlow.Netflow,
							TsOverflow: 0,
						}
						if gateFlow.Netflow > 0 {
							gateFlowSnapshot.FlowIn = gateFlow.Netflow
							entryFlowSnapshot.FlowIn += gateFlow.Netflow
						} else {
							gateFlowSnapshot.FlowOut = gateFlow.Netflow
							entryFlowSnapshot.FlowOut += gateFlow.Netflow
						}
						if snapshots != nil {
							if snapshotgate, found := snapshotEntryGates[gate]; found {
								gateFlow.Netflow += snapshotgate.Netflow
								gateFlow.FlowOut += snapshotgate.FlowOut
								gateFlow.FlowIn += snapshotgate.FlowIn
								gateFlow.TsOverflow = snapshotgate.TsOverflow
							}
						}
						entryFlowSnapshot.Flows[gate] = gateFlowSnapshot
					}

					// TODO verify and add overflow
					//entryFlowSnapshot.FlowIn = EntryFlowIn
					//entryFlowSnapshot.FlowOut = EntryFlowOut
					newSnapshotSpace.FlowIn += entryFlowSnapshot.FlowIn
					newSnapshotSpace.FlowOut += entryFlowSnapshot.FlowOut
					newSnapshotSpace.Flows[entry] = entryFlowSnapshot
				}
			}
			snapshots[data.Space] = newSnapshotSpace
		}
		fmt.Printf("%+v\n", snapshots[data.Space])
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
