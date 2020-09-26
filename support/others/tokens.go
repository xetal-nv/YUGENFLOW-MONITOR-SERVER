package others

import "time"

func ChannelEmptier(ch chan interface{}, stop chan bool, delay int) {
	for {
		select {
		case <-stop:
			return
		case <-time.After(time.Duration(delay) * time.Second):
			select {
			case <-stop:
				return
			case <-ch:
			}
		}
	}
}
