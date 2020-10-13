package avgsManager

import (
	"fmt"
	"gateserver/dataformats"
	"gateserver/support/globals"
	"github.com/fpessolano/mlogger"
	"os"
	"time"
)

// TODO this process will receive every new value and calculate the averages as indicated in a measurement.ini
// TODO reference measurement are to be treated diofferently (not sliding window)
func calculator(space string, latestData chan dataformats.SpaceState, rst chan interface{},
	tick, maxTick int, realTimeDefinitions, referenceDefinitions map[string]int,
	register chan dataformats.SimpleSample) {

	defer func() { //catch or finally
		if err := recover(); err != nil { //catch
			fmt.Fprintf(os.Stderr, "Exception: %v\n", err)
			os.Exit(1)
		}
	}()

	var samples []dataformats.SpaceState

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

			// TODO add reference (non sliding windowq)
			//  and send to register
			for _, period := range realTimeDefinitions {
				var selected []dataformats.SimpleSample
			foundall:
				for i := len(samples) - 1; i >= 0; i-- {
					if samples[i].Ts+int64(period)*1000000000 >= samples[len(samples)-1].Ts {
						selected = append(selected, dataformats.SimpleSample{samples[i].Ts, float64(samples[i].Count)})
					} else {
						if selected[len(selected)-1].Ts != samples[len(samples)-1].Ts-int64(period)*1000000000 {
							selected = append(selected, dataformats.SimpleSample{samples[len(samples)-1].Ts - int64(period)*1000000000,
								float64(samples[i].Count)})
						}
						break foundall
					}
				}
				// TODO do all measurements
				// real time measurement calculation
				if len(selected) > 1 {
					var tot float64 = 0
					length := int(selected[0].Ts - selected[len(selected)-1].Ts)
					for i := len(selected) - 1; i > 0; i-- {
						tot += selected[i].Val * float64(int(selected[i-1].Ts-selected[i].Ts))
						//tot += int(selected[i-1].ts - selected[i].ts)
						//fmt.Println(key, selected[i].val, int(selected[i-1].ts - selected[i].ts), tot)
					}
					tot = float64(int64((tot*100)/float64(length))) / 100
					register <- dataformats.SimpleSample{
						Ts:  selected[0].Ts / 1000000000,
						Val: tot,
					}
					//fmt.Println(space, key, selected[0].Ts/1000000000, tot)
				}
			}
		}
	}
}
