package registers

import (
	"fmt"
	"github.com/dgraph-io/badger"
	"log"
)

var currentDB, statsDB *badger.DB

type serieSample struct {
	ts  int64
	val int
}

func TimedIntDBSSetUp() {
	optsCurr := badger.DefaultOptions
	optsCurr.Dir = "dbs/current"
	optsCurr.ValueDir = "dbs/current"
	optsStats := badger.DefaultOptions
	optsStats.Dir = "dbs/statsDB"
	optsStats.ValueDir = "dbs/statsDB"
	var err error
	currentDB, err = badger.Open(optsCurr)
	if err != nil {
		log.Fatal("registers.TimedIntDBSSetUp: fatal error opening currentDB: ", err)
	}
	statsDB, err = badger.Open(optsStats)
	if err != nil {
		log.Fatal("registers.TimedIntDBSSetUp: fatal error opening statsDB: ", err)
	}
}

func TimedIntDBSClose() {
	//noinspection GoUnhandledErrorResult
	currentDB.Close()
	//noinspection GoUnhandledErrorResult
	statsDB.Close()
}

// TODO all functions
func SetSeries(tag string, step int) {
	// if not initialised it creates a new series
}

func ResetSeries(tag string) {
	// if not initialised it creates a new series
}

func StoreSerieSample(tag string, ts int64, val int, lastFill bool) error {
	// stores value, TS is from stored data
	// in case of large difference with TS fills based on lastFill
	// and returns an error
	return nil
}

func ReadSeries(tag string, ts1, ts2 int64) []serieSample {
	// returns all values between ts1 ans ts2
	return nil
}

// TODO need to be done with a fast key-value DBS!
// TODO HOW
// TODO use badger as a single database,
// TODO for every analysis and space create a first, step and last entry to use for storing and readin
// TODO every new sample will use as index a name plkus the last+ step and update last, not the timestamp received
// TODO if the timestamp received is different of more than 1 step, we will add a sam[;e equal to the previousone
// TODO if the database is started and the last is very different that the current time, we will fill the series with zeros
// TODO give the option to reset the database via API ?
// TODO for the current data we should use a separate database with limited life span of samples as from the .env file
func TimedIntDBS(id string, in chan DataCt, rst chan bool) {
	fmt.Println("DBS id:", id, "called TBD")
	for {
		select {
		case d := <-in:
			fmt.Println("DBS id:", id, "got data", d)
		case a := <-rst:
			fmt.Println("DBS id:", id, "got reset request", a, " TBD")
		}
	}
}
