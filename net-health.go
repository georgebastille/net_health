/*
net-health
Generate an html report of the general heatlh of the current network.
Run:
sudo setcap cap_net_raw+ep net-health
to give the executable permission to send ping requests.
*/
package main

import (
	"fmt"
	"github.com/sparrc/go-ping"
	"time"
)

type pingPoint struct {
	timestamp    time.Time
	url          string
	meanPingtime time.Duration
}

func pingURL(url string, ch chan<- pingPoint) {
	pinger, err := ping.NewPinger(url)
	if err != nil {
		panic(err)
	}
	pinger.Count = 3
	pinger.SetPrivileged(true)
	pinger.Run()
	ch <- pingPoint{time.Now(), url, pinger.Statistics().AvgRtt}
}

func main() {
	ch := make(chan pingPoint)
	urls := make([]string, 3)
	urls[0] = "www.google.com"
	urls[1] = "www.amazon.com"
	urls[2] = "www.apple.com"

	for _, url := range urls {
		go pingURL(url, ch)
	}

	for range urls {
		pinged := <-ch
		fmt.Printf("%v: %v - %v\n", pinged.timestamp, pinged.url, pinged.meanPingtime)
	}
}
