package avgsManager

import (
	"fmt"
	"gateserver/dataformats"
	"gateserver/exportManager"
	"gateserver/storage/coredbs"
	"gateserver/support/globals"
	"github.com/fpessolano/mlogger"
	"time"
)

func calculator(space string, latestData chan dataformats.SpaceState, rst chan interface{},
	tick, maxTick int, realTimeDefinitions, referenceDefinitions map[string]int,
	regRealTime, regReference chan dataformats.MeasurementSample, actualsAvailable bool) {

	var samples []dataformats.SpaceState
	lastReferenceMeasurement := make(map[string]int64)

	for i := range referenceDefinitions {
		lastReferenceMeasurement[i] = 0
	}

	mlogger.Info(globals.AvgsManagerLog,
		mlogger.LoggerData{"avgsManager.calculator for space: " + space,
			"service started",
			[]int{0}, true})

	if maxTick < tick {
		fmt.Printf("Measurement definition are invalid as maximum %v is smaller than tick %v\n", maxTick, tick)
		<-rst
		mlogger.Info(globals.AvgsManagerLog,
			mlogger.LoggerData{"avgsManager.calculator for space: " + space,
				"service stopped",
				[]int{0}, true})
		fmt.Println("closing calculator")
		rst <- nil
	} else {
		if globals.DebugActive {
			fmt.Printf("Calculator %+v\n", realTimeDefinitions)
			fmt.Printf("Calculator %+v\n", referenceDefinitions)
		}

	finished:
		for {
			select {
			case <-rst:
				mlogger.Info(globals.AvgsManagerLog,
					mlogger.LoggerData{"avgsManager.calculator for space: " + space,
						"service stopped",
						[]int{0}, true})
				fmt.Println("Closing calculator for space:", space)
				rst <- nil
				break finished
			case <-time.After(time.Duration(tick) * time.Second):
				//fmt.Printf("calculator %v ticked\n", space)
				// we add a sample that is the same as the last one but with a different time stamp
				// the sliding window is also shifted to make sure it contains only relevant samples
				refTs := time.Now().UnixNano()
				var data dataformats.SpaceState
				if len(samples) != 0 {
					data = samples[len(samples)-1]
				}
				data.Ts = refTs + int64(tick)*1000000000
				data.Flows = make(map[string]dataformats.EntryState)
				//fmt.Printf("calculator %v ticked %v\n", space, data)
				samples = append(samples, data)
				for (samples[0].Ts < refTs-int64(maxTick)*1000000000) && len(samples) > 1 {
					if samples[1].Ts <= refTs-int64(maxTick)*1000000000 {
						samples = samples[1:]
					} else {
						samples[0].Ts = refTs - int64(maxTick)*1000000000
						break
					}
				}
			case data := <-latestData:
				//fmt.Printf("calculator %v received %v\n", space, data)
				// first we make a deep copy of data
				newData := dataformats.SpaceState{
					Id:    data.Id,
					Ts:    data.Ts,
					Count: data.Count,
					State: data.State,
					Flows: make(map[string]dataformats.EntryState),
				}

				for en, ev := range data.Flows {
					newData.Flows[en] = dataformats.EntryState{
						Id:       ev.Id,
						Ts:       ev.Ts,
						Count:    ev.Count,
						State:    ev.State,
						Reversed: ev.Reversed,
						Flows:    make(map[string]dataformats.Flow),
					}
					for gn, gv := range ev.Flows {
						newData.Flows[en].Flows[gn] = dataformats.Flow{
							Id:      gv.Id,
							Netflow: gv.Netflow,
						}
					}
				}
				//fmt.Printf("calculator %v received %v\n\n", space, newData)

				// the current data is sent immediately
				if actualsAvailable {
					select {
					case regReference <- dataformats.MeasurementSample{
						Space:     space,
						Qualifier: "actual",
						Ts:        newData.Ts / 1000000000,
						Val:       float64(newData.Count),
						Flows:     newData.Flows}:
					case <-time.After(time.Duration(globals.SettleTime) * time.Second):
					}
				}

				// the current data is exported immediately
				if globals.ExportActualCommand != "" {
					select {
					case exportManager.ExportActuals <- dataformats.MeasurementSample{
						Space:     space,
						Qualifier: "actual",
						Ts:        newData.Ts / 1000000000,
						Val:       float64(newData.Count),
						Flows:     newData.Flows}:
					default:
						// we do not wait as the delay might be related to an external script
					}
				}

				// we add the new sample and adjust the sliding windows making sure the first and last are
				// aligned with the maximum sliding windows size
				refTs := newData.Ts
				samples = append(samples, newData)
				for samples[0].Ts < refTs-int64(maxTick)*1000000000 && len(samples) > 1 {
					if samples[1].Ts <= refTs-int64(maxTick)*1000000000 {
						samples = samples[1:]
					} else {
						samples[0].Ts = refTs - int64(maxTick)*1000000000
						samples[0].Flows = nil
						break
					}
				}
			}

			if globals.RawMode {
				fmt.Printf("Sliding window from %v to %v\n",
					samples[len(samples)-1].Ts-int64(maxTick)*1000000000, samples[len(samples)-1].Ts)
				for _, el := range samples {
					//fmt.Printf("sliding window %+v\n", el)
					fmt.Printf("\tsample %v at %v\n", el.Count, el.Ts)
				}
				fmt.Println()
			}

			// real time measurements
			for measurementName, period := range realTimeDefinitions {
				var selectedSamples []dataformats.MeasurementSample
				adjPeriod := int64(period) * 1000000000

				// samples are selected starting from the latest one, which is used as ending point of the
				// measuring window, and ending with the closest one to the start point of the sliding window
				// for which the timestamp is adjusted to the window size

			foundall:
				for i := len(samples) - 1; i >= 0; i-- {
					if samples[i].Ts+adjPeriod >= samples[len(samples)-1].Ts {
						selectedSamples = append(selectedSamples, dataformats.MeasurementSample{Ts: samples[i].Ts,
							//Val: float64(samples[i].Count)})
							Val: float64(samples[i].Count), Flows: samples[i].Flows})
					} else {
						// we need to properly close the interval
						if selectedSamples[len(selectedSamples)-1].Ts != samples[len(samples)-1].Ts-adjPeriod {
							selectedSamples = append(selectedSamples, dataformats.MeasurementSample{Ts: samples[len(samples)-1].Ts - adjPeriod,
								Val: float64(samples[i].Count)})
						}
						break foundall
					}
				}

				//fmt.Printf("interval %v to %v\n", samples[len(samples)-1].Ts-adjPeriod, samples[len(samples)-1].Ts)
				//for _, el := range samples {
				//	fmt.Printf("selected sample %v at %v\n", el.Count, el.Ts)
				//}
				//fmt.Println()
				//continue

				//for _, el := range selectedSamples {
				//	//fmt.Print(time.Unix(el.Ts/1000000000, 0),  " ")
				//	//fmt.Printf("selected sample %v at %v\n", el.Val, el.Ts)
				//	fmt.Printf("selected sample %+v\n", el)
				//}
				//fmt.Println()
				//continue

				// the selectedSamples slice starts with the latest entry value (at [0])
				// the ending sample is considered for the flow calculation but not for the average count
				if len(selectedSamples) > 1 {

					// both flow and counter can be calculated
					var tot float64 = 0
					flows := make(map[string]dataformats.EntryState)
					length := float64(selectedSamples[0].Ts - selectedSamples[len(selectedSamples)-1].Ts)
					for i := len(selectedSamples) - 1; i >= 0; i-- {
						if i > 0 {
							// we update the total count as for the second most recent sample
							tot += selectedSamples[i].Val * float64(selectedSamples[i-1].Ts-selectedSamples[i].Ts)
						}
						//fmt.Println(tot)
						for sampleEntryName, sampleEntry := range selectedSamples[i].Flows {
							if entry, ok := flows[sampleEntryName]; ok {
								// adjust values and gates
								entry.Count += sampleEntry.Count
								for gateSampleName, gateSampleCurrent := range sampleEntry.Flows {
									if gate, found := entry.Flows[gateSampleName]; found {
										gate.Netflow += gateSampleCurrent.Netflow
										entry.Flows[gateSampleName] = gate
									} else {
										// new gate flow, we deep copy it
										entry.Flows[gateSampleName] = dataformats.Flow{
											Id:      gateSampleCurrent.Id,
											Netflow: gateSampleCurrent.Netflow,
										}
									}
								}
								flows[sampleEntryName] = entry
							} else {
								// new entry, we make a deep copy
								flows[sampleEntryName] = dataformats.EntryState{
									Id:       sampleEntry.Id,
									Ts:       sampleEntry.Ts,
									Count:    sampleEntry.Count,
									State:    sampleEntry.State,
									Reversed: sampleEntry.Reversed,
									Flows:    make(map[string]dataformats.Flow),
								}
								for i, val := range sampleEntry.Flows {
									flows[sampleEntryName].Flows[i] = dataformats.Flow{
										Id:      val.Id,
										Netflow: val.Netflow,
									}
								}
							}
						}
					}

					//fmt.Println()

					// result is limited to two digits
					tot = float64(int64((tot*100)/length)) / 100
					//fmt.Println(tot)
					//fmt.Println()

					if globals.RawMode {
						fmt.Printf("Measurement result: \n\t %+v\n\n", dataformats.MeasurementSample{
							Qualifier: measurementName,
							Ts:        selectedSamples[0].Ts / 1000000000,
							Val:       tot,
							Flows:     flows,
						})
					}

					//fmt.Printf("%+v\n\n", dataformats.MeasurementSample{
					//	Qualifier: measurementName,
					//	Ts:        selectedSamples[0].Ts / 1000000000,
					//	Val:       tot,
					//	Flows:     flows,
					//})

					// we give it little time to transmit the data, it too late data is thrown away
					select {
					case regRealTime <- dataformats.MeasurementSample{
						Qualifier: measurementName,
						Ts:        selectedSamples[0].Ts / 1000000000,
						Val:       tot,
						Flows:     flows,
					}:
					case <-time.After(time.Duration(globals.SettleTime) * time.Second):
					}

				} else {
					// ATTENTION since tick is 1/3 of the minimum measure
					//  this should never happen, if it does we have a bug somewhere
					mlogger.Error(globals.AvgsManagerLog,
						mlogger.LoggerData{"avgsManager.calculator for space: " + space,
							"tick size error",
							[]int{0}, true})
				}
			}

			//continue

			//reference measurements
			for measurementName, period := range referenceDefinitions {
				adjPeriod := int64(period) * 1000000000
				if lastReferenceMeasurement[measurementName]+adjPeriod < samples[len(samples)-1].Ts {
					//fmt.Println("new period", time.Now())
					// time for a new reference measurement
					var selectedSamples []dataformats.MeasurementSample
				foundall2:
					for i := len(samples) - 1; i >= 0; i-- {
						if samples[i].Ts+adjPeriod >= samples[len(samples)-1].Ts {
							selectedSamples = append(selectedSamples, dataformats.MeasurementSample{Ts: samples[i].Ts,
								Val: float64(samples[i].Count), Flows: samples[i].Flows})
						} else {
							if selectedSamples[len(selectedSamples)-1].Ts != samples[len(samples)-1].Ts-adjPeriod {
								selectedSamples = append(selectedSamples, dataformats.MeasurementSample{Ts: samples[len(samples)-1].Ts - adjPeriod,
									Val: float64(samples[i].Count)})
							}
							break foundall2
						}
					}

					//for _, el := range selectedSamples {
					//	//fmt.Printf("selected sample %v at %v\n", el.Val, el.Ts)
					//	fmt.Printf("selected sample %v\n", el)
					//}
					//fmt.Println()
					//lastReferenceMeasurement[measurementName] = samples[len(samples)-1].Ts
					//continue

					// measurement calculation
					if len(selectedSamples) > 1 {
						var tot float64 = 0
						flows := make(map[string]dataformats.EntryState)
						length := float64(selectedSamples[0].Ts - selectedSamples[len(selectedSamples)-1].Ts)
						for i := len(selectedSamples) - 1; i >= 0; i-- {
							if i > 0 {
								tot += selectedSamples[i].Val * float64(selectedSamples[i-1].Ts-selectedSamples[i].Ts)
							}
							//fmt.Println(tot)
							//fmt.Println()
							for sampleEntryName, sampleEntry := range selectedSamples[i].Flows {
								if entry, ok := flows[sampleEntryName]; ok {
									// adjust values and gates
									entry.Count += sampleEntry.Count
									for gateSampleName, gateSampleCurrent := range sampleEntry.Flows {
										if gate, found := entry.Flows[gateSampleName]; found {
											gate.Netflow += gateSampleCurrent.Netflow
											entry.Flows[gateSampleName] = gate
										} else {
											// new gate flow, we deep copy it
											entry.Flows[gateSampleName] = dataformats.Flow{
												Id:      gateSampleCurrent.Id,
												Netflow: gateSampleCurrent.Netflow,
											}
										}
									}
									flows[sampleEntryName] = entry
								} else {
									// new entry, we make a deep copy
									flows[sampleEntryName] = dataformats.EntryState{
										Id:       sampleEntry.Id,
										Ts:       sampleEntry.Ts,
										Count:    sampleEntry.Count,
										State:    sampleEntry.State,
										Reversed: sampleEntry.Reversed,
										Flows:    make(map[string]dataformats.Flow),
									}
									for i, val := range sampleEntry.Flows {
										flows[sampleEntryName].Flows[i] = dataformats.Flow{
											Id:      val.Id,
											Netflow: val.Netflow,
										}
									}
								}
							}

						}
						tot = float64(int64((tot*100)/length)) / 100
						//fmt.Println(tot)
						//fmt.Println()
						newSample := dataformats.MeasurementSample{
							Qualifier: measurementName,
							Space:     space,
							Ts:        selectedSamples[0].Ts / 1000000000,
							Val:       tot,
							Flows:     flows,
						}

						if globals.RawMode {
							fmt.Printf("Reference result:\n\t %+v\n\n", newSample)
						}

						// we give it little time to transmit the data, it too late data is thrown away
						select {
						case regReference <- newSample:
						case <-time.After(time.Duration(globals.SettleTime) * time.Second):
						}

						lastReferenceMeasurement[measurementName] = samples[len(samples)-1].Ts
						go func(nd dataformats.MeasurementSample) {
							_ = coredbs.SaveReferenceData(nd)
						}(newSample)

						// the current data is exported immediately
						if globals.ExportReferenceCommand != "" {
							select {
							case exportManager.ExportReference <- newSample:
							default:
								// we do not wait as the delay might be related to an external script
							}
						}
					}
				}
			}
		}
	}
}
