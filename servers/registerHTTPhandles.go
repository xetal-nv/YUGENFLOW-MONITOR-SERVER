package servers

import (
	"countingserver/spaces"
	"countingserver/storage"
	"countingserver/support"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

type Register struct {
	Valid bool                `json:"valid"`
	Error string              `json:"errorcode"`
	Data  storage.GenericData `json:"counter"`
}

type RegisterBank struct {
	Name string     `json:"space"`
	Data []Register `json:"counters"`
}

// handles single register requests
func singleRegisterHTTPhandles(path string, ref string) http.Handler {

	sp := strings.Split(strings.Trim(path, "/"), "/")
	tag := ""
	for i := range sp {
		sp[i] = support.StringLimit(sp[i], support.LabelLength)
		tag += strings.Replace(sp[i], "_", "", -1) + "_"
	}
	tag = tag[:len(tag)-1]
	rt := Register{true, "", storage.DataMap[ref]}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if e := recover(); e != nil {
				if e != nil {
					log.Println("servers.singleRegisterHTTPhandles: recovering from: ", e)
				}
			}
		}()
		//fmt.Printf("%s %s %s \n", r.Method, r.URL, r.Proto)

		//Allow CORS here By * or specific origin
		w.Header().Set("Access-Control-Allow-Origin", "*")

		select {
		case data := <-spaces.LatestBankOut[sp[0]][sp[1]][sp[2]]:
			if e := rt.Data.Extract(data); e != nil {
				rt.Valid = false
				rt.Error = "ID"
			} else {
				rt.Data.SetTag(tag)
			}
		case <-time.After(2000 * time.Millisecond):
			rt.Valid = false
			rt.Error = "TO"
		}

		//noinspection GoUnhandledErrorResult
		json.NewEncoder(w).Encode(rt)
	})
}

// handles space requests
func spaceRegisterHTTPhandles(path string, als []string, ref string) http.Handler {

	sp := strings.Split(strings.Trim(path, "/"), "/")
	tag := ""
	for i := range sp {
		sp[i] = support.StringLimit(sp[i], support.LabelLength)
		tag += strings.Replace(sp[i], "_", "", -1) + "_"
	}
	tag = tag[:len(tag)-1]
	//fmt.Println(tag)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if e := recover(); e != nil {
				if e != nil {
					log.Println("servers.singleRegisterHTTPhandles: recovering from: ", e)
				}
			}
		}()

		//Allow CORS here By * or specific origin
		w.Header().Set("Access-Control-Allow-Origin", "*")

		//var rt RegisterBank
		////var rt2 RegisterBank
		//
		//rt.Name = tag
		////rt2.Name = tag
		//
		////for _, nm := range als {
		////	var tmp Register
		////	tmp.Data = ref.NewEl()
		////	select {
		////	case data := <-spaces.LatestBankOut[sp[0]][sp[1]][nm]:
		////		if e := tmp.Data.Extract(data); e != nil {
		////			tmp.Valid = false
		////			tmp.Error = "ID"
		////		} else {
		////			tmp.Valid = true
		////		}
		////		tmp.Data.SetTag(tag + "_" + strings.Replace(nm, "_", "", -1))
		////	case <-time.After(2000 * time.Millisecond):
		////		tmp.Valid = false
		////		tmp.Error = "TO"
		////	}
		////	rt2.Data = append(rt2.Data, tmp)
		////}
		//
		//var tmpv []*Register
		//var wg sync.WaitGroup
		//
		//for _, nm := range als {
		//	tmp := new(Register)
		//	wg.Add(1)
		//	//var tmp Register
		//	tmp.Data = storage.DataMap[ref]
		//	//rt.Data = append(rt.Data, tmp)
		//	tmpv = append(tmpv, tmp)
		//	go func(tmp *Register) {
		//		defer wg.Done()
		//		select {
		//		case data := <-spaces.LatestBankOut[sp[0]][sp[1]][nm]:
		//			if e := tmp.Data.Extract(data); e != nil {
		//				tmp.Valid = false
		//				tmp.Error = "ID"
		//			} else {
		//				tmp.Valid = true
		//			}
		//			tmp.Data.SetTag(tag + "_" + strings.Replace(nm, "_", "", -1))
		//		case <-time.After(2000 * time.Millisecond):
		//			tmp.Valid = false
		//			tmp.Error = "TO"
		//		}
		//	}(tmp)
		//}
		//
		//wg.Wait()
		//for _, v := range tmpv {
		//	rt.Data = append(rt.Data, *v)
		//}
		//
		////fmt.Println(rt2.Data)
		////fmt.Println(rt.Data)
		//
		////a := []RegisterBank{rt,rt2}
		//
		////noinspection GoUnhandledErrorResult
		////json.NewEncoder(w).Encode(a)

		_ = json.NewEncoder(w).Encode(retrieveSpace(tag, sp, als, ref))

	})
}

// TODO handles datatype requests
func datatypeRegisterHTTPhandles(path string, als [][]string) http.Handler {
	return nil
}

func retrieveSpace(tag string, sp []string, als []string, ref string) (rt RegisterBank) {
	var tmpv []*Register
	var wg sync.WaitGroup

	rt.Name = tag

	for _, nm := range als {
		tmp := new(Register)
		wg.Add(1)
		//var tmp Register
		tmp.Data = storage.DataMap[ref]
		//rt.Data = append(rt.Data, tmp)
		tmpv = append(tmpv, tmp)
		go func(tmp *Register) {
			defer wg.Done()
			select {
			case data := <-spaces.LatestBankOut[sp[0]][sp[1]][nm]:
				if e := tmp.Data.Extract(data); e != nil {
					tmp.Valid = false
					tmp.Error = "ID"
				} else {
					tmp.Valid = true
				}
				tmp.Data.SetTag(tag + "_" + strings.Replace(nm, "_", "", -1))
			case <-time.After(2000 * time.Millisecond):
				tmp.Valid = false
				tmp.Error = "TO"
			}
		}(tmp)
	}

	wg.Wait()
	for _, v := range tmpv {
		rt.Data = append(rt.Data, *v)
	}
	return
}
