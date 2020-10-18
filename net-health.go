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
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"
	"gonum.org/v1/plot/vg/draw"
	"image/color"
	"log"
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

	f, err := os.Open(filename)
	if err != nil {
		panic(err)
	}

	dec := json.NewDecoder(f)
	xs := make([]float64, 0)
	ys := make([]float64, 0)
	// while the array contains values
	for dec.More() {
		var m pingPoint
		// decode an array value (Message)
		err := dec.Decode(&m)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Println(m)
		xs = append(xs, float64(m.Timestamp.Unix()))
		ys = append(ys, float64(m.MeanPingtime))

	}
	xticks := plot.TimeTicks{Format: "2006-01-02\n15:04:32.132"}
	data := make(plotter.XYs, len(xs))
	for i := range data {
		data[i].X = xs[i]
		data[i].Y = ys[i]
	}

	p, err := plot.New()
	if err != nil {
		log.Panic(err)
	}
	p.Title.Text = "Time Series"
	p.X.Tick.Marker = xticks
	p.Y.Label.Text = "Number of Gophers\n(Billions)"
	p.Add(plotter.NewGrid())

	line, points, err := plotter.NewLinePoints(data)
	if err != nil {
		log.Panic(err)
	}
	line.Color = color.RGBA{G: 255, A: 255}
	points.Shape = draw.CircleGlyph{}
	points.Color = color.RGBA{R: 255, A: 255}

	p.Add(line, points)

	err = p.Save(20*vg.Centimeter, 7*vg.Centimeter, "timeseries.png")
	if err != nil {
		log.Panic(err)
	}
}

func collectTimingData() {

	filename := "responseTimes.json"

	f, err := os.OpenFile(filename, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		panic(err)
	}
	jsonOut := json.NewEncoder(f)

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
			jsonOut.Encode(pinged)
		}
	}

	fmt.Println("...finished")
}
