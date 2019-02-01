package main

import (
	"fmt"
	"math/rand"
	"playground/support"
	"time"
)

func test(a string) {
	fmt.Println(a)
	r := rand.Intn(5)
	time.Sleep(time.Duration(r) * time.Second)
	panic("panicking")
}

func testrecover() {
	support.RunWithRecovery(func() { test("waiting") }, func() { fmt.Println("recovering") })
}
