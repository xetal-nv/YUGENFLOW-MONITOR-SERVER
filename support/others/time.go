package others

import (
	"math/rand"
	"time"
)

var seed = time.Now().UnixNano()

func WaitRandom(reference int) {
	wait := rand.New(rand.NewSource(seed)).Intn(reference)
	time.Sleep(time.Duration(wait) * time.Second)
}
