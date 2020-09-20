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
	"time"

	"github.com/sparrc/go-ping"
)

func pingURL(url string) time.Duration {
	pinger, err := ping.NewPinger(url)
	if err != nil {
		panic(err)
	}
	pinger.Count = 3
	pinger.SetPrivileged(true)
	pinger.Run()
	return pinger.Statistics().AvgRtt
}

func main() {
	urls := make([]string, 3)
	urls[0] = "www.google.com"
	urls[1] = "www.amazon.com"
	urls[2] = "www.apple.com"

	for _, url := range urls {
		fmt.Println(url, pingURL(url))
	}

}
