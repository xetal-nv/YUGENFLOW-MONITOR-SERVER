package registers

import (
	"countingserver/support"
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/dgraph-io/badger"
	"log"
	"os"
	"strconv"
	"sync"
	"time"
)

// Codeddata is the data format used in the database
type codeddata []byte

// Data is the data format for the header
type headerData struct {
	fromRst  uint64
	step     uint32
	lastUpdt uint64
	created  uint64
}

var currentDB, statsDB *badger.DB
var once sync.Once
var currentTTL time.Duration
var tagStart map[string][]int64

type serieSample struct {
	ts  int64
	val int
}

func TimedIntDBSSetUp() error {
	var err error
	tagStart = make(map[string][]int64)
	currentTTL = time.Hour * 24 * 14
	if v := os.Getenv("EXPTIME"); v != "" {
		if vd, er := strconv.Atoi(v); er != nil {
			currentTTL = time.Hour * 24 * time.Duration(vd)
		}
	}
	log.Printf("registers.TimedIntDBSSetUp: current TTL set to %v\n", currentTTL)
	once.Do(func() {
		optsCurr := badger.DefaultOptions
		optsCurr.Dir = "dbs/current"
		optsCurr.ValueDir = "dbs/current"
		optsStats := badger.DefaultOptions
		optsStats.Dir = "dbs/statsDB"
		optsStats.ValueDir = "dbs/statsDB"
		currentDB, err = badger.Open(optsCurr)
		if err == nil {

			statsDB, err = badger.Open(optsStats)
		} else {
			if err != nil {
				currentDB.Close()
			}
		}
	})
	return err
}

func TimedIntDBSClose() {
	//noinspection GoUnhandledErrorResult
	currentDB.Close()
	//noinspection GoUnhandledErrorResult
	statsDB.Close()
}

// External functions/API
func SetSeries(tag string, step int) (bool, error) {
	var err error
	found := true
	if _, ok := tagStart[tag]; !ok {
		// if not initialised it creates a new series
		// sets the entry in tagStart
		nt := []byte(tag + "0")
		if _, e := read(nt, 28, *currentDB); e != nil {
			found = false
			fmt.Println("not found")
			a := headerData{}
			a.fromRst = uint64(support.Timestamp())
			a.step = uint32(step)
			a.lastUpdt = a.fromRst
			a.created = a.fromRst
			b := a.marshall()
			err = b.Update(nt, *currentDB, false)
			tagStart[tag] = []int64{int64(a.fromRst), int64(a.step)}
		} else {
			if c, e := read([]byte{0}, 28, *currentDB); e != nil {
				err = e
			} else {
				if a, e := c.unmarshall(); e != nil {
					err = e
				} else {
					tagStart[tag] = []int64{int64(a.fromRst), int64(a.step)}
				}

			}
		}
	}
	return found, err
}

// TODO
func StoreSerieSample(tag string, ts int64, val int) error {
	// stores value, TS is from stored data
	// in case of large difference with TS fills based on lastFill
	// and returns an error
	var err error

	if st, ok := tagStart[tag]; ok {
		fmt.Println(st)
		i := (ts - st[0]) / (st[1] * 1000)
		fmt.Println(ts, st, i)
	} else {
		err = errors.New("Serie " + tag + " not found")
	}
	return err
}

func ReadSeries(tag string, ts1, ts2 int64) []serieSample {
	// returns all values between ts1 ans ts2
	return nil
}

func GetDefinition(tag string) []int64 {
	return tagStart[tag]
}

// Core database functions

// Unmarshall decodes a codeddata into headerData
func (c codeddata) unmarshall() (headerData, error) {

	d := headerData{}
	if len(c) != 28 {
		return d, errors.New("Invalid raw data provided")
	}
	d.fromRst = binary.LittleEndian.Uint64(c[0:8])
	d.step = binary.LittleEndian.Uint32(c[8:12])
	d.lastUpdt = binary.LittleEndian.Uint64(c[12:20])
	d.created = binary.LittleEndian.Uint64(c[20:28])
	return d, nil

}

// Marshall encodes a Data values into codeddata
func (d headerData) marshall() codeddata {
	r := make([]byte, 8)
	binary.LittleEndian.PutUint64(r, d.fromRst)
	b := make([]byte, 4)
	binary.LittleEndian.PutUint32(b, d.step)
	r = append(r, b...)
	c := make([]byte, 8)
	binary.LittleEndian.PutUint64(c, d.lastUpdt)
	r = append(r, c...)
	binary.LittleEndian.PutUint64(c, d.created)
	r = append(r, c...)
	return r
}

// View read an entry
func read(id []byte, l int, db badger.DB) (codeddata, error) {
	r := make([]byte, l)
	err := db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(id)
		if err != nil {
			return err
		}
		val, err := item.Value()
		if err != nil {
			return err
		}
		copy(r, val)
		return nil
	})
	return r, err
}

// Delete deletes an entry
func delete(id []byte, db badger.DB) error {
	err := db.Update(func(txn *badger.Txn) error {
		err := txn.Delete(id)
		return err
	})
	return err
}

// Update updates updates an entry
func (a codeddata) Update(id []byte, db badger.DB, ttl bool) error {
	err := db.Update(func(txn *badger.Txn) error {
		var err error
		if ttl {
			err = txn.SetWithTTL(id, a, currentTTL)
		} else {
			err = txn.Set(id, a)
		}
		return err
	})
	return err
}
