package storage

import (
	"github.com/dgraph-io/badger"
	"sync"
	"time"
)

var currentDB, statsDB *badger.DB // databases
var once sync.Once                // used fpr one time set-up function
var currentTTL time.Duration      // provides the TTL value to be used
var tagStart map[string][]int64   // map of definition for all series currently in use
