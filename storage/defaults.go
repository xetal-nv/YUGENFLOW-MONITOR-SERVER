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

var garbage struct {
	start       time.Time
	end         time.Time
	intervalMin time.Duration
}

var timeout int
var currentDB, statsDB *badger.DB     // databases
var once sync.Once                    // used fpr one time set-up function
var currentTTL time.Duration          // provides the TTL value to be used
var tagStart map[string][]int64       // map of definition for all series currently in use
var statsChanIn chan dbInChan         // channel for storing to the statistical DB
var currentChanIn chan dbInChan       // channel for storing to the current DB
var statsChanOut chan dbOutCommChan   // channel for reading to the statistical DB
var currentChanOut chan dbOutCommChan // channel for readin to the current DB
