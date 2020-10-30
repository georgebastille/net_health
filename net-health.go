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
	"flag"
	"fmt"
	"github.com/sparrc/go-ping"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"
	"gonum.org/v1/plot/vg/draw"
	"image/color"
	"log"
	"net/http"
	"os"
	"time"
)

type pingPoint struct {
	Timestamp    time.Time
	URL          string
	Count        int
	MeanPingtime time.Duration
}

func (p pingPoint) String() string {
	return fmt.Sprintf("%v: %v - %v", p.Timestamp, p.URL, p.MeanPingtime)
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
	remoteURLs := make([]string, 4)
	remoteURLs[0] = "www.google.com"
	remoteURLs[1] = "www.amazon.com"
	remoteURLs[2] = "www.apple.com"
	remoteURLs[3] = "www.bbc.co.uk"

	return remoteURLs
}

type coordSeries struct {
	xs []float64
	ys []float64
}

func main() {
	ticker := time.NewTicker(10 * time.Second)
	go func() {
		for range ticker.C {
			collectTimingData()
			renderPlot()
		}
	}()
	servePlot()
}

func servePlot() {
	port := flag.String("p", "8000", "Port server listens to")
	flag.Parse()
	http.Handle("/", http.FileServer(http.Dir("./static")))
	log.Printf("Listening on port %v", *port)
	log.Fatal(http.ListenAndServe("0.0.0.0:"+*port, nil))
}
func renderPlot() {
	// TODO:
	// Create a plot per url, and then lets serve a basic template
	if _, err := os.Stat("./static/"); os.IsNotExist(err) {
		err := os.Mkdir("./static/", 0755)
		if err != nil {
			panic(err)
		}
	}

	filename := "responseTimes.json"

	f, err := os.Open(filename)
	if err != nil {
		panic(err)
	}

	dec := json.NewDecoder(f)
	series := make(map[string]coordSeries)
	// while the array contains values
	var m pingPoint
	for dec.More() {
		// decode an array value (Message)
		err := dec.Decode(&m)
		if err != nil {
			log.Fatal(err)
		}

		var xs = series[m.URL].xs
		xs = append(xs, float64(m.Timestamp.Unix()))
		var ys = series[m.URL].ys
		ys = append(ys, float64(m.MeanPingtime)/1e6)
		series[m.URL] = coordSeries{xs, ys}

	}
	xticks := plot.TimeTicks{Format: "2020-12-25\n15:04:32"}

	plotSeries := make(map[string]plotter.XYs)
	for url, values := range series {
		data := make(plotter.XYs, len(values.xs))
		for i := range data {
			data[i].X = values.xs[i]
			data[i].Y = values.ys[i]
		}
		plotSeries[url] = data
	}

	for url, data := range plotSeries {
		p, err := plot.New()
		if err != nil {
			log.Panic(err)
		}
		p.X.Tick.Marker = xticks
		p.Y.Label.Text = "Ping Time (ms)"
		p.Add(plotter.NewGrid())

		line, points, err := plotter.NewLinePoints(data)
		if err != nil {
			log.Panic(err)
		}
		// Hash url to get colour
		line.Color = color.RGBA{G: 255, A: 255}
		points.Shape = draw.CircleGlyph{}
		points.Color = color.RGBA{R: 255, A: 255}

		p.Add(line, points)
		p.Title.Text = fmt.Sprintf("Ping for %s", url)
		outImg := fmt.Sprintf("./static/%s.png", url)
		err = p.Save(20*vg.Centimeter, 7*vg.Centimeter, outImg)
		if err != nil {
			log.Panic(err)
		}
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

	activeURLs := make([]string, 0)
	for range hosts {
		checked := <-ch

		if checked.active {
			activeURLs = append(activeURLs, checked.url)
		}
	}

	fmt.Printf("Collecting ping Statistics for %v hosts...\n", len(activeURLs))
	ch2 := make(chan pingPoint)
	for _, url := range activeURLs {
		go pingURL(url, ch2)
		time.Sleep(25 * time.Millisecond)
	}

	for range activeURLs {
		pinged := <-ch2
		if pinged.Count > 0 {
			fmt.Println(pinged)
			jsonOut.Encode(pinged)
		}
	}

	fmt.Println("...finished")
}
