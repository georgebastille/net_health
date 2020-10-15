/*
net-health
Generate an html report of the general heatlh of the current network.
Run:
sudo setcap cap_net_raw+ep net-health
to give the executable permission to send ping requests.
*/
package main

import (
	"encoding/json"
	"fmt"
	"github.com/sparrc/go-ping"
	"os"
	"time"
)

type pingPoint struct {
	Timestamp    time.Time
	Url          string
	Count        int
	MeanPingtime time.Duration
}

func (p pingPoint) String() string {
	return fmt.Sprintf("%v: %v - %v", p.Timestamp, p.Url, p.MeanPingtime)
}

type activeURL struct {
	url    string
	active bool
}

func checkURL(url string, ch chan<- activeURL) {
	pinger, err := ping.NewPinger(url)
	if err != nil {
		panic(err)
	}
	pinger.SetPrivileged(true)
	pinger.Count = 1
	pinger.Timeout, _ = time.ParseDuration("1s")
	pinger.Run()
	ch <- activeURL{url, pinger.PacketsRecv > 0}
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

	filename := "responseTimes.json"

	f, err := os.OpenFile(filename, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		panic(err)
	}
	json_out := json.NewEncoder(f)

	defer f.Close()
	hosts := getRemoteURLs()
	hosts = append(hosts, getLocalIPs()...)
	ch := make(chan activeURL)
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
		if pinged.Count > 0 {
			fmt.Println(pinged)
			json_out.Encode(pinged)
		}
	}

	fmt.Println("...finished")
}
