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
	count        int
	meanPingtime time.Duration
}

func pingURL(url string, ch chan<- pingPoint) {
	pinger, err := ping.NewPinger(url)
	if err != nil {
		panic(err)
	}
	pinger.SetPrivileged(true)
	pinger.Interval, _ = time.ParseDuration("100ms")
	pinger.Timeout, _ = time.ParseDuration("1s")
	pinger.Run()
	ch <- pingPoint{time.Now(), url, pinger.PacketsRecv, pinger.Statistics().AvgRtt}
}

func getLocalIPs() []string {
	baseIP := "192.168.1."
	var localIPS = make([]string, 256)
	for i := 0; i < 256; i++ {
		localIPS[i] = fmt.Sprintf("%v%v", baseIP, i)
	}
	return localIPS
}

func getRemoteURLs() []string {
	remoteUrls := make([]string, 3)
	remoteUrls[0] = "www.google.com"
	remoteUrls[1] = "www.amazon.com"
	remoteUrls[2] = "www.apple.com"
	return remoteUrls
}

func main() {
	ch := make(chan pingPoint)
	hosts := getRemoteURLs()

	hosts = append(hosts, getLocalIPs()...)

	for _, url := range hosts {
		go pingURL(url, ch)
		time.Sleep(5 * time.Millisecond)
	}

	for range hosts {
		pinged := <-ch
		if pinged.count > 0 {
			fmt.Printf("%v: %v - %v\n", pinged.timestamp, pinged.url, pinged.meanPingtime)
		}
	}

}
