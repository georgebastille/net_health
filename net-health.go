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

type activeUrl struct {
	url    string
	active bool
}

func checkURL(url string, ch chan<- activeUrl) {
	pinger, err := ping.NewPinger(url)
	if err != nil {
		panic(err)
	}
	pinger.SetPrivileged(true)
	pinger.Count = 1
	pinger.Timeout, _ = time.ParseDuration("1s")
	pinger.Run()
	ch <- activeUrl{url, pinger.PacketsRecv > 0}
}

func pingURL(url string, ch chan<- pingPoint) {
	pinger, err := ping.NewPinger(url)
	if err != nil {
		panic(err)
	}
	pinger.SetPrivileged(true)
	pinger.Interval, _ = time.ParseDuration("100ms")
	pinger.Timeout, _ = time.ParseDuration("2s")
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
	remoteUrls := make([]string, 4)
	remoteUrls[0] = "www.google.com"
	remoteUrls[1] = "www.amazon.com"
	remoteUrls[2] = "www.apple.com"
	remoteUrls[3] = "www.bbc.co.uk"

	return remoteUrls
}

func main() {
	hosts := getRemoteURLs()
	hosts = append(hosts, getLocalIPs()...)
	ch := make(chan activeUrl)
	fmt.Printf("Testing %v local and remote hosts...\n", len(hosts))
	for _, url := range hosts {
		go checkURL(url, ch)
		time.Sleep(1 * time.Millisecond)
	}

	activeUrls := make([]string, 0)
	for range hosts {
		checked := <-ch

		if checked.active {
			activeUrls = append(activeUrls, checked.url)
		}
	}

	fmt.Printf("Collecting ping Statistics for %v hosts...\n", len(activeUrls))
	ch2 := make(chan pingPoint)
	for _, url := range activeUrls {
		go pingURL(url, ch2)
		time.Sleep(25 * time.Millisecond)
	}

	for range activeUrls {
		pinged := <-ch2
		if pinged.count > 0 {
			fmt.Printf("%v: %v - %v\n", pinged.timestamp, pinged.url, pinged.meanPingtime)
		}
	}

	fmt.Println("...finished")
}
