package storage

import (
	"github.com/dgraph-io/badger"
	"sync"
	"time"
)

var currentDB, statsDB *badger.DB
var once sync.Once
var currentTTL time.Duration
var tagStart map[string][]int64

var DataMap map[string]GenericData
