package avgsManager

import (
	"fmt"
	"gateserver/dataformats"
	"gateserver/storage/coredbs"
	"gateserver/support/globals"
	"github.com/fpessolano/mlogger"
	"os"
	"time"
)

func calculator(space string, latestData chan dataformats.SpaceState, rst chan interface{},
	tick, maxTick int, realTimeDefinitions, referenceDefinitions map[string]int,
	regRealTime, regReference chan dataformats.MeasurementSample, actualsAvailable bool) {

	// for development only, comment afterwards
	defer func() {
		if err := recover(); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "Exception: %v\n", err)
			os.Exit(1)
		}
	}()

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
		//fmt.Printf("%+v\n", realTimeDefinitions)
		//fmt.Printf("%+v\n", referenceDefinitions)

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
				refTs := time.Now().UnixNano()
				var data dataformats.SpaceState
				if len(samples) != 0 {
					data = samples[len(samples)-1]
				}
				data.Ts = refTs
				samples = append(samples, data)
				for samples[0].Ts < refTs-int64(maxTick)*1000000000 || len(samples) > 1 {
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
							Id:        gv.Id,
							Variation: gv.Variation,
						}
					}
				}

				// the current data is sent immediately
				if actualsAvailable {
					regReference <- dataformats.MeasurementSample{
						Qualifier: "actual",
						Ts:        newData.Ts / 1000000000,
						Val:       float64(newData.Count),
						Flows:     newData.Flows,
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

				// real time measurements
				for measurementName, period := range realTimeDefinitions {
					var selectedSamples []dataformats.MeasurementSample
					adjPeriod := int64(period) * 1000000000
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

					// the selectedSamples slice starts with the latest entry [0]
					// measurement calculation
					if len(selectedSamples) > 1 {
						// both flow and counter can be calculated
						var tot float64 = 0
						flows := make(map[string]dataformats.EntryState)
						length := int(selectedSamples[0].Ts - selectedSamples[len(selectedSamples)-1].Ts)
						for i := len(selectedSamples) - 1; i >= 0; i-- {
							if i > 0 {
								// we update the total count
								tot += selectedSamples[i].Val * float64(int(selectedSamples[i-1].Ts-selectedSamples[i].Ts))
							}
							for sampleEntryName, sampleEntry := range selectedSamples[i].Flows {
								if entry, ok := flows[sampleEntryName]; ok {
									// adjust values and gates
									entry.Count += sampleEntry.Count
									for gateSampleName, gateSampleCurrent := range sampleEntry.Flows {
										if gate, found := entry.Flows[gateSampleName]; found {
											gate.Variation += gateSampleCurrent.Variation
											entry.Flows[gateSampleName] = gate
										} else {
											// new gate flow, we deep copy it
											entry.Flows[gateSampleName] = dataformats.Flow{
												Id:        gateSampleCurrent.Id,
												Variation: gateSampleCurrent.Variation,
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
											Id:        val.Id,
											Variation: val.Variation,
										}
									}
								}
							}
						}

						// result is limited to two digits
						tot = float64(int64((tot*100)/float64(length))) / 100

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

					}
				}

				//reference measurements
				for measurementName, period := range referenceDefinitions {
					adjPeriod := int64(period) * 1000000000
					if lastReferenceMeasurement[measurementName]+int64(adjPeriod) < samples[len(samples)-1].Ts {
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

						// measurement calculation
						if len(selectedSamples) > 1 {
							var tot float64 = 0
							flows := make(map[string]dataformats.EntryState)
							length := int(selectedSamples[0].Ts - selectedSamples[len(selectedSamples)-1].Ts)
							for i := len(selectedSamples) - 1; i >= 0; i-- {
								if i > 0 {
									tot += selectedSamples[i].Val * float64(int(selectedSamples[i-1].Ts-selectedSamples[i].Ts))
								}
								for sampleEntryName, sampleEntry := range selectedSamples[i].Flows {
									if entry, ok := flows[sampleEntryName]; ok {
										// adjust values and gates
										entry.Count += sampleEntry.Count
										for gateSampleName, gateSampleCurrent := range sampleEntry.Flows {
											if gate, found := entry.Flows[gateSampleName]; found {
												gate.Variation += gateSampleCurrent.Variation
												entry.Flows[gateSampleName] = gate
											} else {
												// new gate flow, we deep copy it
												entry.Flows[gateSampleName] = dataformats.Flow{
													Id:        gateSampleCurrent.Id,
													Variation: gateSampleCurrent.Variation,
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
												Id:        val.Id,
												Variation: val.Variation,
											}
										}
									}
								}

							}
							tot = float64(int64((tot*100)/float64(length))) / 100
							newSample := dataformats.MeasurementSample{
								Qualifier: measurementName,
								Space:     space,
								Ts:        selectedSamples[0].Ts / 1000000000,
								Val:       tot,
								Flows:     flows,
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
						}
					}
				}

			}
		}
	}
}
