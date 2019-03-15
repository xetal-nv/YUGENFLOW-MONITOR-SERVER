package storage

import (
	"countingserver/support"
	"encoding/binary"
	"errors"
	"github.com/dgraph-io/badger"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"time"
)

func TimedIntDBSSetUp(fd bool) error {
	// fd is used for testing or bypass the configuration file also in its absence
	force := false
	_ = os.MkdirAll("dbs", os.ModePerm)
	if !fd {
		if os.Getenv("FORCEDBS") == "1" {
			force = true
		}
	} else {
		force = true
	}
	if force {
		log.Printf("storage.TimedIntDBSClose: Force mode on, deleting lock files if present\n")
		_ = os.Remove("dbs/current/LOCK")
		_ = os.Remove("dbs/statsDB/LOCK")

		if files, e := filepath.Glob("dbs/current/*.vlog"); e == nil {
			for _, f := range files {
				_ = os.Remove(f)
			}
		}
		if files, e := filepath.Glob("dbs/statsDB/*.vlog"); e == nil {
			for _, f := range files {
				_ = os.Remove(f)
			}
		}
	}
	var err error
	tagStart = make(map[string][]int64)
	currentTTL = time.Hour * 24 * 14
	if v := os.Getenv("EXPTIME"); v != "" {
		if vd, er := strconv.Atoi(v); er != nil {
			currentTTL = time.Hour * 24 * time.Duration(vd)
		}
	}
	log.Printf("storage.TimedIntDBSSetUp: current TTL set to %v\n", currentTTL)
	if support.Debug < 3 {
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
				if err != nil {
					currentDB.Close()
				}
			}
		})
	} else {
		log.Printf("storage.TimedIntDBSClose: Databases not enables for current Debug mode\n")
	}
	return err
}

func TimedIntDBSClose() {
	if support.Debug < 3 {
		//noinspection GoUnhandledErrorResult
		currentDB.Close()
		//noinspection GoUnhandledErrorResult
		statsDB.Close()
	}
}

// External functions/API
func SetSeries(tag string, step int, sDB bool) (found bool, err error) {
	found = true
	var db badger.DB
	if sDB {
		db = *statsDB
	} else {
		db = *currentDB
	}
	if _, ok := tagStart[tag]; !ok {
		// if not initialised it creates a new series
		// sets the entry in tagStart
		nt := []byte(tag + "0")
		if c, e := read(nt, 28, []int{}, db); e != nil {
			found = false
			a := Headerdata{}
			a.fromRst = uint64(support.Timestamp())
			a.step = uint32(step)
			a.lastUpdt = a.fromRst
			a.created = a.fromRst
			b := a.Marshal()
			err = update(b, nt, db, false)
			tagStart[tag] = []int64{int64(a.fromRst), int64(a.step)}
			log.Printf("register.SetSeries: new series %v:%v added\n", tag, step)
		} else {
			//if c, e := read(nt, 28, db); e != nil {
			//	err = e
			//} else {
			var a Headerdata
			if e := a.Unmarshal(c); e != nil {
				err = e
			} else {
				tagStart[tag] = []int64{int64(a.fromRst), int64(a.step)}
				log.Printf("register.SetSeries: existing series %v:%v loaded\n", tag, step)
			}

		}
		//}
	}
	return found, err
}

func ReadHeader(tag string, sDB bool) (hd Headerdata, err error) {
	var db badger.DB
	if sDB {
		db = *statsDB
	} else {
		db = *currentDB
	}
	//if _, ok := tagStart[Stag]; !ok {
	//	err = errors.New("Header not found")
	//} else {
	if c, e := read([]byte(tag+"0"), 28, []int{}, db); e != nil {
		err = e
	} else {
		err = hd.Unmarshal(c)
	}
	//}
	return
}

func updateHeader(tag string, sDB bool, gts ...int64) (err error) {
	var ts int64
	var hd Headerdata
	var db badger.DB
	if sDB {
		db = *statsDB
	} else {
		db = *currentDB
	}
	if len(gts) != 1 {
		ts = support.Timestamp()
	} else {
		ts = gts[0]
	}
	if hd, err = ReadHeader(tag, sDB); err == nil {
		hd.lastUpdt = uint64(ts)
		b := hd.Marshal()
		err = update(b, []byte(tag+"0"), db, false)
	}
	return
}

func StoreSample(d SampleData, sDB bool, updatehead ...bool) (err error) {
	ts := d.Ts()
	tag := d.Tag()
	val := d.Marshal()
	var db badger.DB
	if sDB {
		db = *statsDB
	} else {
		db = *currentDB
	}
	if st, ok := tagStart[tag]; ok {
		i := (ts - st[0]) / (st[1] * 1000)
		lab := tag + strconv.Itoa(int(i))
		err = update(val, []byte(lab), db, false)
	} else {
		err = errors.New("Serie " + tag + " not found")
	}
	if len(updatehead) == 1 {
		if updatehead[0] {
			err = updateHeader(d.Tag(), sDB, ts)
		}
	}
	return err
}

func ReadSeries(s0, s1 SampleData, sDB bool) (tag string, rts []int64, rt [][]byte, err error) {
	// returns all values between s1 and s2, extremes included
	if s0.MarshalSize() == 0 && len(s0.MarshalSizeModifiers()) != 2 {
		err = errors.New("storage.ReadSeries: type not supporter: " + reflect.TypeOf(s0).String())
		return
	}
	var db badger.DB
	if sDB {
		db = *statsDB
	} else {
		db = *currentDB
	}
	tag = s0.Tag()
	ts0 := s0.Ts()
	ts1 := s1.Ts()
	if st, ok := tagStart[tag]; ok {
		if ts1 != st[0] {
			if ts0 <= st[0] {
				ts0 = st[0] + st[1]*1000 // offset to skip the header
			}
			i := (ts0 - st[0]) / (st[1] * 1000)
			i1 := (ts1 - st[0]) / (st[1] * 1000)
			for i <= i1 {
				lab := []byte(tag + strconv.Itoa(int(i)))
				if v, e := read(lab, s0.MarshalSize(), s0.MarshalSizeModifiers(), db); e == nil {
					nts := st[0] + i*st[1]*1000
					rt = append(rt, v)
					rts = append(rts, nts)
				}
				i += 1
			}
		}
	} else {
		err = errors.New("Serie " + tag + " not found")
	}
	return tag, rts, rt, err
}

func ReadLastN(head SampleData, ns int, sDB bool) (tag string, rts []int64, rt [][]byte, err error) {
	if head.MarshalSize() == 0 && len(head.MarshalSizeModifiers()) != 2 {
		err = errors.New("storage.ReadSeries: type not supporter: " + reflect.TypeOf(head).String())
		return
	}
	var db badger.DB
	if sDB {
		db = *statsDB
	} else {
		db = *currentDB
	}
	tag = head.Tag()
	ts1 := head.Ts()
	if st, ok := tagStart[tag]; ok {
		if ts1 <= st[0] {
			err = errors.New("storage.ReadLastN: illegal end series point provided")
		} else {
			i1 := (ts1 - st[0]) / (st[1] * 1000)
			for j := ns; j > 0; j-- {
				i := i1 - int64(j)
				if i > 0 {
					lab := []byte(tag + strconv.Itoa(int(i)))
					if v, e := read(lab, head.MarshalSize(), head.MarshalSizeModifiers(), db); e == nil {
						nts := st[0] + i*st[1]*1000
						rt = append(rt, v)
						rts = append(rts, nts)
					}
				}
			}
		}
	} else {
		err = errors.New("storage.ReadLastN: serie " + tag + " not found")
	}
	return
}

func GetDefinition(tag string) []int64 {
	return tagStart[tag]
}

// View read an entry, when l==0, it assumes it is a variable length element
// and it retrieves the length from the element first (maximum number of fields in 16 bit)
func read(id []byte, l int, offset []int, db badger.DB) (v []byte, err error) {

	read := func(id []byte, lb int, db badger.DB) (r []byte, err error) {
		if lb > 0 {
			r = make([]byte, lb)
			err = db.View(func(txn *badger.Txn) error {
				item, err := txn.Get(id)
				if err != nil {
					return err
				}
				val, err := item.ValueCopy(nil)
				if err != nil {
					return err
				}
				copy(r, val)
				return nil
			})
		} else {
			err = errors.New("storage.DBS: error storing data with length <= 0")
		}
		return
	}

	if l > 0 {
		return read(id, l, db)
	} else {
		if l == 0 && len(offset) == 2 {
			var r []byte
			r, err = read(id, 2, db)
			if err == nil {
				vs := int(binary.LittleEndian.Uint16(r))*offset[0] + 2 + offset[1]
				if r, err = read(id, vs, db); err == nil {
					v = make([]byte, len(r))
					copy(v, r)
				}

			}
		}
	}
	return
}

// Delete deletes an entry
//func delEntry(id []byte, db badger.DB) error {
//	err := db.Update(func(txn *badger.Txn) error {
//		err := txn.Delete(id)
//		return err
//	})
//	return err
//}

// update updates updates an entry
func update(a []byte, id []byte, db badger.DB, ttl bool) (err error) {
	err = db.Update(func(txn *badger.Txn) error {
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
