package avgsManager

import (
	"fmt"
	"gateserver/dataformats"
	"gateserver/support/globals"
	"github.com/fpessolano/mlogger"
	"os"
	"time"
)

func calculator(space string, latestData chan dataformats.SpaceState, rst chan interface{},
	tick, maxTick int, realTimeDefinitions, referenceDefinitions map[string]int,
	regRealTime, regReference chan dataformats.SimpleSample) {

	// for development only, comment afterwards
	defer func() {
		if err := recover(); err != nil {
			fmt.Fprintf(os.Stderr, "Exception: %v\n", err)
			os.Exit(1)
		}
	}()

	var samples []dataformats.SpaceState
	lastReferenceMeasurement := make(map[string]int64)

	for i := range referenceDefinitions {
		lastReferenceMeasurement[i] = 0
	}

	mlogger.Info(globals.AvgsLogger,
		mlogger.LoggerData{"avgsManager.calculator for space: " + space,
			"service started",
			[]int{0}, true})

	if maxTick < tick {
		fmt.Printf("Measurement definition are invalid as maximum %v is smaller than tick %v\n", maxTick, tick)
		<-rst
		mlogger.Info(globals.AvgsLogger,
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
				mlogger.Info(globals.AvgsLogger,
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
				// we add the new sample and adjust the sliding windows making sure the first and last are
				// aligned with the maximum sliding windows size
				refTs := data.Ts
				samples = append(samples, data)
				for samples[0].Ts < refTs-int64(maxTick)*1000000000 || len(samples) > 1 {
					if samples[1].Ts <= refTs-int64(maxTick)*1000000000 {
						samples = samples[1:]
					} else {
						samples[0].Ts = refTs - int64(maxTick)*1000000000
						break
					}
				}
			}
			// real time measurements
			for measurementName, period := range realTimeDefinitions {
				var selected []dataformats.SimpleSample
				adjPeriod := int64(period) * 1000000000
			foundall:
				for i := len(samples) - 1; i >= 0; i-- {
					if samples[i].Ts+adjPeriod >= samples[len(samples)-1].Ts {
						selected = append(selected, dataformats.SimpleSample{Ts: samples[i].Ts, Val: float64(samples[i].Count)})
					} else {
						if selected[len(selected)-1].Ts != samples[len(samples)-1].Ts-adjPeriod {
							selected = append(selected, dataformats.SimpleSample{Ts: samples[len(samples)-1].Ts - adjPeriod,
								Val: float64(samples[i].Count)})
						}
						break foundall
					}
				}

				// measurement calculation
				if len(selected) > 1 {
					var tot float64 = 0
					length := int(selected[0].Ts - selected[len(selected)-1].Ts)
					for i := len(selected) - 1; i > 0; i-- {
						tot += selected[i].Val * float64(int(selected[i-1].Ts-selected[i].Ts))
						//tot += int(selected[i-1].ts - selected[i].ts)
						//fmt.Println(measurementName, selected[i].val, int(selected[i-1].ts - selected[i].ts), tot)
					}
					tot = float64(int64((tot*100)/float64(length))) / 100
					regRealTime <- dataformats.SimpleSample{
						Qualifier: measurementName,
						Ts:        selected[0].Ts / 1000000000,
						Val:       tot,
					}
					//fmt.Println(space, key, selected[0].Ts/1000000000, tot)
				}
			}
			// reference measurements
			for measurementName, period := range referenceDefinitions {
				adjPeriod := int64(period) * 1000000000
				if lastReferenceMeasurement[measurementName]+int64(adjPeriod) < samples[len(samples)-1].Ts {
					// time for a new reference measurement
					var selected []dataformats.SimpleSample
				foundall2:
					for i := len(samples) - 1; i >= 0; i-- {
						if samples[i].Ts+adjPeriod >= samples[len(samples)-1].Ts {
							selected = append(selected, dataformats.SimpleSample{Ts: samples[i].Ts, Val: float64(samples[i].Count)})
						} else {
							if selected[len(selected)-1].Ts != samples[len(samples)-1].Ts-adjPeriod {
								selected = append(selected, dataformats.SimpleSample{Ts: samples[len(samples)-1].Ts - adjPeriod,
									Val: float64(samples[i].Count)})
							}
							break foundall2
						}
					}
					// measurement calculation
					if len(selected) > 1 {
						var tot float64 = 0
						length := int(selected[0].Ts - selected[len(selected)-1].Ts)
						for i := len(selected) - 1; i > 0; i-- {
							tot += selected[i].Val * float64(int(selected[i-1].Ts-selected[i].Ts))
							//tot += int(selected[measurementName-1].ts - selected[measurementName].ts)
							//fmt.Println(measurementName, selected[i].Val, int(selected[i-1].Ts - selected[i].Ts), tot)
						}
						tot = float64(int64((tot*100)/float64(length))) / 100
						regReference <- dataformats.SimpleSample{
							Qualifier: measurementName,
							Ts:        selected[0].Ts / 1000000000,
							Val:       tot,
						}
						//fmt.Println(space, measurementName, selected[0].Ts/1000000000, tot)
						lastReferenceMeasurement[measurementName] = samples[len(samples)-1].Ts
					}
				}
			}

		}
	}
}
