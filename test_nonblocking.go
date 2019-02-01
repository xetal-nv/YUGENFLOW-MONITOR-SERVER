package main

import (
	"fmt"
	"math/rand"
	"playground/registers"
	"strconv"
	"time"
)

func testnonblocking() {

	ins := make([]chan int, 2)
	outs := make([]chan int, 2)

	for i := 0; i < 2; i++ {
		ins[i] = make(chan int)
		outs[i] = make(chan int)
	}

	if registers.IntBank("5sec_", ins, outs) {
		panic("error")
	}

	go func() {
		i := 0
		for {
			go func() { ins[0] <- i }()
			ins[1] <- i
			i += 1
			r := rand.Intn(10)
			time.Sleep(time.Duration(r) * time.Second)
		}
	}()
	for i := 1; i < 20; i++ {
		select {
		case val := <-outs[0]:
			fmt.Println("0: " + strconv.Itoa(val))
		default:
			//fmt.Println("A: not available")
			select {
			case val := <-outs[1]:
				fmt.Println("1: " + strconv.Itoa(val))
			default:
				fmt.Println("No channel available")
			}
		}
		r := rand.Intn(5)
		time.Sleep(time.Duration(r) * time.Second)
	}
}
