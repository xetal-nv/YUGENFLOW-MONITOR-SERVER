package storage

import (
	"encoding/binary"
	"errors"
	"fmt"
	"gateserver/support"
	"github.com/dgraph-io/badger"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"time"
)

// set-ups database
func TimedIntDBSSetUp(fd bool) error {
	// fd is used for testing or bypass the configuration file also in its absence
	if v, e := strconv.Atoi(os.Getenv("DBSTO")); e != nil {
		timeout = 5
	} else {
		timeout = v
	}
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
			optsCurr.Truncate = true
			optsCurr.Dir = "dbs/current"
			optsCurr.ValueDir = "dbs/current"
			optsStats := badger.DefaultOptions
			optsStats.Truncate = true
			optsStats.Dir = "dbs/statsDB"
			optsStats.ValueDir = "dbs/statsDB"
			currentDB, err = badger.Open(optsCurr)
			if err == nil {
				statsDB, err = badger.Open(optsStats)
				if err != nil {
					_ = currentDB.Close()
				} else {
					bufsize := 50
					if v, e := strconv.Atoi(os.Getenv("DBSBUFFSIZE")); e == nil {
						bufsize = v
					}
					statsChanIn = make(chan dbInChan, bufsize)
					currentChanIn = make(chan dbInChan, bufsize)
					go dbUpdateDriver(statsChanIn, *statsDB, false)
					go dbUpdateDriver(currentChanIn, *currentDB, true)
					statsChanOut = make(chan dbOutCommChan, bufsize)
					currentChanOut = make(chan dbOutCommChan, bufsize)
					go dbReadDriver(statsChanOut, *statsDB)
					go dbReadDriver(currentChanOut, *currentDB)
				}
			}
		})
	} else {
		log.Printf("storage.TimedIntDBSClose: Databases not enables for current Debug mode\n")
	}
	return err
}

// closes database, it is used to support defer closure
func TimedIntDBSClose() {
	if support.Debug < 3 {
		//noinspection GoUnhandledErrorResult
		currentDB.Close()
		//noinspection GoUnhandledErrorResult
		statsDB.Close()
	}
}

// External functions/API
// set-ups a series and/or retrieve its definition from the database
func SetSeries(tag string, step int, sDB bool) (found bool, err error) {
	found = true
	//var db badger.DB
	//if sDB {
	//	db = *statsDB
	//} else {
	//	db = *currentDB
	//}
	if _, ok := tagStart[tag]; !ok {
		// if not initialised it creates a new series
		// sets the entry in tagStart
		//fmt.Println(nt, step, sDB)
		if c, e := ReadHeader(tag, sDB); e != nil {
			nt := []byte(tag + "header")
			found = false
			a := Headerdata{}
			a.fromRst = uint64(support.Timestamp())
			a.step = uint32(step)
			a.lastUpdt = a.fromRst
			a.created = a.fromRst
			b := a.Marshal()
			//err = update(b, nt, db, false)
			if sDB {
				select {
				case statsChanIn <- dbInChan{nt, b, true}:
				case <-time.After(time.Duration(timeout) * time.Second):
					return false, errors.New("series " + tag + " stats time out")
				}
			} else {
				select {
				case currentChanIn <- dbInChan{nt, b, true}:
				case <-time.After(time.Duration(timeout) * time.Second):
					return false, errors.New("series " + tag + " current time out")
				}
			}
			tagStart[tag] = []int64{int64(a.fromRst), int64(a.step)}
			log.Printf("register.SetSeries: new series %v:%v added\n", tag, step)
		} else {
			tagStart[tag] = []int64{int64(c.fromRst), int64(c.step)}
			log.Printf("register.SetSeries: existing series %v:%v loaded\n", tag, step)
		}
	}
	//fmt.Println(support.Timestamp() - tagStart[tag][0])
	//os.Exit(1)
	return found, err
}

// reads the header of a series
func ReadHeader(tag string, sDB bool) (hd Headerdata, err error) {
	//type dbOutCommChan struct {
	//	id     []byte
	//	l      int;
	//	offset []int;
	//	co     chan struct {
	//		r   []byte
	//		err error
	//	}
	//}
	//if _, ok := tagStart[tag]; !ok {
	//	err = errors.New("Header not found")
	//} else {
	co := make(chan dbOutChan)
	//var db badger.DB
	if sDB {
		//db = *statsDB
		select {
		case statsChanOut <- dbOutCommChan{[]byte(tag + "header"), 28, []int{}, co}:
		case <-time.After(time.Duration(timeout) * time.Second):
			return hd, errors.New("ReadHeader " + tag + " stats time out")
		}
	} else {
		//db = *currentDB
		select {
		case currentChanOut <- dbOutCommChan{[]byte(tag + "header"), 28, []int{}, co}:
		case <-time.After(time.Duration(timeout) * time.Second):
			return hd, errors.New("ReadHeader " + tag + " current time out")
		}
	}
	select {
	case ans := <-co:
		if ans.err == nil {
			err = hd.Unmarshal(ans.r)
		} else {
			err = ans.err
		}
	case <-time.After(time.Duration(timeout) * time.Second):
		err = errors.New("ReadHeader timeout")
	}
	//}
	//if _, ok := tagStart[tag]; !ok {
	//	err = errors.New("Header not found")
	//} else {
	//if c, e := read([]byte(tag+"header"), 28, []int{}, db); e != nil {
	//	err = e
	//} else {
	//	err = hd.Unmarshal(c)
	//}
	//}
	return
}

// updates the header of a series
func updateHeader(tag string, sDB bool, gts ...int64) (err error) {
	var ts int64
	var hd Headerdata
	//var db badger.DB
	//if sDB {
	//	db = *statsDB
	//} else {
	//	db = *currentDB
	//}
	if len(gts) != 1 {
		ts = support.Timestamp()
	} else {
		ts = gts[0]
	}
	if hd, err = ReadHeader(tag, sDB); err == nil {
		hd.lastUpdt = uint64(ts)
		b := hd.Marshal()
		//err = update(b, []byte(tag+"0"), db, false)
		if sDB {
			select {
			case statsChanIn <- dbInChan{[]byte(tag + "0"), b, true}:
			case <-time.After(time.Duration(timeout) * time.Second):
				return errors.New("updateHeader " + tag + " stats time out")
			}
		} else {
			select {
			case currentChanIn <- dbInChan{[]byte(tag + "0"), b, true}:
			case <-time.After(time.Duration(timeout) * time.Second):
				return errors.New("updateHeader " + tag + " current time out")
			}
		}
	}
	return
}

// stores a sample, optionally it updates the header
func StoreSample(d SampleData, sDB bool, updatehead ...bool) (err error) {
	ts := d.Ts()
	tag := d.Tag()
	val := d.Marshal()
	//var db badger.DB
	//sampleMutex.Lock()
	//if sDB {
	//	db = *statsDB
	//} else {
	//	db = *currentDB
	//}
	if st, ok := tagStart[tag]; ok {
		i := (ts - st[0]) / (st[1] * 1000)
		lab := tag + strconv.Itoa(int(i))
		if sDB {
			//err = update(val, []byte(lab), *statsDB, false)
			select {
			case statsChanIn <- dbInChan{[]byte(lab), val, false}:
			case <-time.After(time.Duration(timeout) * time.Second):
				return errors.New("StoreSample " + tag + " stats time out")
			}
		} else {
			//err = update(val, []byte(lab), *currentDB, false)
			select {
			case currentChanIn <- dbInChan{[]byte(lab), val, false}:
			case <-time.After(time.Duration(timeout) * time.Second):
				return errors.New("StoreSample " + tag + " current time out")
			}
		}
		//err = update(val, []byte(lab), db, false)
	} else {
		err = errors.New("Serie " + tag + " not found")
	}
	if len(updatehead) == 1 {
		if updatehead[0] {
			err = updateHeader(d.Tag(), sDB, ts)
		}
	}
	//sampleMutex.Unlock()
	return err
}

// reads a series, all values between the timestamos included in s0 and s1 are reads
// values are returnes as a set of values
// tag: identified of the series
// rts: list of timestamps
// rt: list of the series values ordered according to rts
// err: reports the error is any
// TODO behaves weirdly on timeout and change of gates
func ReadSeries(s0, s1 SampleData, sDB bool) (tag string, rts []int64, rt [][]byte, err error) {
	// returns all values between s1 and s2, extremes included
	if s0.MarshalSize() == 0 && len(s0.MarshalSizeModifiers()) != 2 {
		err = errors.New("storage.ReadSeries: type not supporter: " + reflect.TypeOf(s0).String())
		return
	}
	//var db badger.DB
	//if sDB {
	//	db = *statsDB
	//} else {
	//	db = *currentDB
	//}
	//fmt.Println(s0, s1)
	tag = s0.Tag()
	ts0 := s0.Ts()
	ts1 := s1.Ts()
	if st, ok := tagStart[tag]; ok {
		if ts1 != st[0] {
			if ts0 <= (st[0] + st[1]*1000) {
				ts0 = st[0] + st[1]*1000 // offset to skip the header
			}
			i := (ts0 - st[0]) / (st[1] * 1000)
			i1 := (ts1 - st[0]) / (st[1] * 1000)
			for i <= i1 {
				lab := []byte(tag + strconv.Itoa(int(i)))
				co := make(chan dbOutChan)
				//var db badger.DB
				if sDB {
					//db = *statsDB
					select {
					case statsChanOut <- dbOutCommChan{lab, s0.MarshalSize(), s0.MarshalSizeModifiers(), co}:
					case <-time.After(time.Duration(timeout) * time.Second):
						return tag, rts, rt, errors.New("ReadSeries " + tag + " stats time out")
					}
				} else {
					//db = *currentDB
					//fmt.Println("here")
					select {
					case currentChanOut <- dbOutCommChan{lab, s0.MarshalSize(), s0.MarshalSizeModifiers(), co}:
					case <-time.After(time.Duration(timeout) * time.Second):
						return tag, rts, rt, errors.New("ReadSeries " + tag + " current time out")
					}
				}
				//fmt.Println("going for receive", co)
				select {
				case ans := <-co:
					//fmt.Print("received", ans)
					if ans.err == nil {
						nts := st[0] + i*st[1]*1000
						rt = append(rt, ans.r)
						rts = append(rts, nts)
					}
				case <-time.After(time.Duration(timeout) * time.Second):
					return tag, rts, rt, errors.New("ReadSeries " + tag + " receive time out")
				}
				//if v, e := read(lab, s0.MarshalSize(), s0.MarshalSizeModifiers(), db); e == nil {
				//	nts := st[0] + i*st[1]*1000
				//	rt = append(rt, v)
				//	rts = append(rts, nts)
				//}
				i += 1
			}
		}
	} else {
		err = errors.New("Serie " + tag + " not found")
	}
	return tag, rts, rt, err
}

// It returns the values for the last N timestamps
// values are returnes as a set of values
// tag: identified of the series
// rts: list of timestamps
// rt: list of the series values ordered according to rts
// err: reports the error is any
// TODO make it with channels!
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

// returns the stored definition of a series with identified tag
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

// View read an entry, when l==0, it assumes it is a variable length element
// and it retrieves the length from the element first (maximum number of fields in 16 bit)
// this version acts as a buffered thread to a database, requires time-out and closure on the receicing end
func dbReadDriver(ch chan dbOutCommChan, db badger.DB) {
	defer func() {
		if e := recover(); e != nil {
			go func() {
				support.DLog <- support.DevData{"storage.dbUpdateDriver: recovering server",
					support.Timestamp(), "", []int{1}, true}
			}()
			fmt.Println("recovering")
			go dbReadDriver(ch, db)
		}
	}()

	read := func(id []byte, lb int, db badger.DB) (r []byte, err error) {
		if lb > 0 {
			r = make([]byte, lb)
			// locks clean-up to avoid database corruption
			support.CleanupLock.RLock()
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
			support.CleanupLock.RUnlock()
		} else {
			err = errors.New("storage.DBS: error storing data with length <= 0")
		}
		return
	}

	for {
		//fmt.Println("dbReadDriver waiting")
		req := <-ch
		//go func(req dbOutCommChan, db badger.DB) {
		//fmt.Println("dbReadDriver starting", req)
		var v []byte
		var e error
		if req.l > 0 {
			v, e = read(req.id, req.l, db)
			//select {
			//case req.co <- dbOutChan{v, e}:
			//case <-time.After(time.Duration(timeout) * time.Second):
			//}
		} else {
			var r []byte
			if req.l == 0 && len(req.offset) == 2 {
				r, e = read(req.id, 2, db)
				if e == nil {
					vs := int(binary.LittleEndian.Uint16(r))*req.offset[0] + 2 + req.offset[1]
					if r, e = read(req.id, vs, db); e == nil {
						v = make([]byte, len(r))
						copy(v, r)
						//select {
						//case req.co <- dbOutChan{v, e}:
						//case <-time.After(time.Duration(timeout) * time.Second):
						//}
					}
				}
			}
		}
		select {
		case req.co <- dbOutChan{v, e}:
		case <-time.After(time.Duration(timeout) * time.Second):
		}
		//fmt.Println("dbReadDriver ending", req)
		//}(req, db)
	}
}

// Delete deletes an entry
//func delEntry(id []byte, db badger.DB) error {
//	err := db.Update(func(txn *badger.Txn) error {
//		err := txn.Delete(id)
//		return err
//	})
//	return err
//}

// update updates updates an entry as a function
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

// update updates updates an entry via goroutines
func dbUpdateDriver(c chan dbInChan, db badger.DB, ttl bool) {
	defer func() {
		if e := recover(); e != nil {
			go func() {
				support.DLog <- support.DevData{"storage.dbUpdateDriver: recovering server",
					support.Timestamp(), "", []int{1}, true}
			}()
			go dbUpdateDriver(c, db, ttl)
		}
	}()
	for {
		data := <-c
		go func(data dbInChan, db badger.DB, ttl bool) {
			// locks clean-up to avoid database corruption
			support.CleanupLock.RLock()
			err := db.Update(func(txn *badger.Txn) error {
				var err error
				if ttl && (!data.oride) {
					err = txn.SetWithTTL(data.id, data.val, currentTTL)
				} else {
					err = txn.Set(data.id, data.val)
				}
				return err
			})
			support.CleanupLock.RUnlock()
			if err != nil {
				log.Printf("Error writing at address %v: %v\n", data.id, err)
			}
		}(data, db, ttl)
	}
}
