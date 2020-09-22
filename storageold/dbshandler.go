package storageold

import (
	"gateserver/supp"
	"log"
)

// handler for a DBS with generic value coming from a channel on a generic interface{}
// a is used to insert the actual value from the generic channel
func handlerDBS(id string, in chan interface{}, rst chan bool, a SampleData, tp string) {

	r := func() {
		log.Printf("register.TimedIntDBS: DBS handler %v started\n", id)
		for {
			select {
			case d := <-in:
				//fmt.Println(id, "received", d)
				//if supp.Debug != 3 && supp.Debug != 4 {
				if e := a.Extract(d); e == nil {
					if a.Valid() {
						//fmt.Println(id, "storing", a, tp)
						switch tp {
						case "SD":
							if err := StoreSampleSD(a, !supp.StringEnding(id, "current", "_")); err != nil {
								log.Printf("storage.TimedIntDBS: DBS handler %v error %v for data %v, type %v\n", id, err, a, tp)
							}
						case "TS":
							if err := StoreSampleTS(a, !supp.StringEnding(id, "current", "_")); err != nil {
								log.Printf("storage.TimedIntDBS: DBS handler %v error %v for data %v, type %v\n", id, err, a, tp)
							}
						default:
							log.Printf("storage.TimedIntDBS: DBS handler %v wrong data type %v\n", id, tp)
						}
					} else {
						//log.Printf("storage.TimedIntDBS: DBS handler discarded empty data %v for %v\n", a, id)
					}
				} else {
					log.Printf("storage.TimedIntDBS: DBS handler extraction error %v for data %v\n", e.Error(), d)
				}
				//}
				//if supp.Debug > 0 {
				//	fmt.Println("DEBUG DBS id:", id, "got data", d, "is current", supp.StringEnding(id, "current", "_"))
				//}
			case aa := <-rst:
				// Reset via API might be dangerous, this is just a  placeholder
				log.Println("storage.TimedIntDBS: handler id:", id, "got reset request", aa, " - does nothing still")
			}
		}
	}
	er := func() {
		log.Printf("storage.TimedIntDBS: recovering from crash\n ")
	}
	go supp.RunWithRecovery(r, er)
}