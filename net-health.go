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
)

func pingURL(url string, ch chan<- string) {
	pinger, err := ping.NewPinger(url)
	if err != nil {
		panic(err)
	}
	pinger.Count = 3
	pinger.SetPrivileged(true)
	pinger.Run()
	ch <- fmt.Sprintf("%v: %v", url, pinger.Statistics().AvgRtt)
}

func main() {
	ch := make(chan string)
	urls := make([]string, 3)
	urls[0] = "www.google.com"
	urls[1] = "www.amazon.com"
	urls[2] = "www.apple.com"

	for _, url := range urls {
		go pingURL(url, ch)
	}

	for range urls {
		fmt.Println(<-ch)
	}
}
