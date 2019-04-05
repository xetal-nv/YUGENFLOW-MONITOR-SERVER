package storage

import (
	"github.com/dgraph-io/badger"
	"sync"
	"time"
)

type dbInChan struct {
	id    []byte
	val   []byte
	oride bool
}

type dbOutChan struct {
	r   []byte
	err error
}

type dbOutCommChan struct {
	id     []byte
	l      int
	offset []int
	co     chan dbOutChan
}

var timeout int
var currentDB, statsDB *badger.DB // databases
var once sync.Once                // used fpr one time set-up function
var currentTTL time.Duration      // provides the TTL value to be used
var tagStart map[string][]int64   // map of definition for all series currently in use
//var sampleMutex = &sync.Mutex{}   // mutex for badger update function, to be checked if necessary apart from suppressing race warnings
var statsChanIn chan dbInChan         // channel for storing to the statistical DB
var currentChanIn chan dbInChan       // channel for storing to the current DB
var statsChanOut chan dbOutCommChan   // channel for reading to the statistical DB
var currentChanOut chan dbOutCommChan // channel for readin to the current DB
