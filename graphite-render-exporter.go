package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

const pollPeriod = time.Second * 10

var graphiteBaseURL = os.Getenv("GRAPHITE_URL")
var graphiteTargets = os.Getenv("TARGETS")
var graphitePollDepth = os.Getenv("POLL_DEPTH")
var graphiteHttpPassword = os.Getenv("HTTP_PASSWORD")
var graphiteHttpUser = os.Getenv("HTTP_USER")
var debug bool = false
var url string

var metricRequest = make(chan chan []Metric)

type Target struct {
	Target     string      `json:"target"`
	Datapoints []Datapoint `json:"datapoints"`
}

func (t Target) String() string {
	return fmt.Sprintf("target= %s, datapoints= %s", t.Target, t.Datapoints)
}

type Datapoint [2]*float64

func (g Datapoint) String() string {
	if g[0] == nil {
		return fmt.Sprintf("[null, %.0f]", *g[1])
	} else {
		return fmt.Sprintf("[%.1f, %.0f]", *g[0], *g[1])
	}
}

type Metric struct {
	Name   string
	Value  float64
	Labels map[string]string
}

func (m Metric) String() string {
	labelList := []string{}
	for k, v := range m.Labels {
		labelList = append(labelList, k+"=\""+v+"\"")
	}
	return m.Name + "{" + strings.Join(labelList, ",") + "}" + " " + strconv.FormatFloat(m.Value, 'f', -1, 64) + "\n"
}

var myClient = &http.Client{Timeout: 10 * time.Second}

func poller(channel chan []Metric) {
	if debug {
		log.Println("Start polling metrics from Graphite")
	}
	metrics := getMetrics(url)
	if debug {
		log.Println("Poller: Got metrics from Graphite", metrics)
	}
	channel <- metrics
	if debug {
		log.Println("Poll finished")
	}
}

func getMetrics(url string) []Metric {
	if debug {
		log.Println("Retrieving from: " + url)
	}

	data := []Target{}
	getJson(url, &data)
	if debug {
		log.Println(data)
	}
	m := []Metric{}
	for _, t := range data {
		metric := Metric{Name: strings.Replace(t.Target, ".", "_", -1), Value: getLastNonNullValue(t.Datapoints), Labels: getLabels()}
		m = append(m, metric)
	}
	return m
}

func getLabels() map[string]string {
	return map[string]string{"url": graphiteBaseURL}
}

func getLastNonNullValue(d []Datapoint) float64 {
	for i := len(d) - 1; i >= 0; i-- {
		if d[i][0] != nil {
			return *d[i][0]
		}
	}
	return 0
}

func getJson(url string, target interface{}) error {
	req, err := http.NewRequest("GET", url, nil)
	if graphiteHttpPassword != "" && graphiteHttpUser != "" {
		req.SetBasicAuth(graphiteHttpUser, graphiteHttpPassword)
	}
	r, err := myClient.Do(req)
	if err != nil {
		return err
	}
	return json.NewDecoder(r.Body).Decode(target)
}

func serveGraphite(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", 405)
		return
	}
	defer r.Body.Close()
	if debug {
		log.Println("Server: Got request")
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	var metricsOut = make(chan []Metric)
	defer close(metricsOut)
	metricRequest <- metricsOut
	metrics := <-metricsOut
	if debug {
		log.Println("Server: Received to handler", metrics)
	}
	c := 0
	for _, m := range metrics {
		if debug {
			log.Println("writing metric to response #", c, " - ", m)
		}
		fmt.Fprintf(w, m.String())
		c++
	}
	if debug {
		log.Println("Finished writing ", c, " metrics")
	}
}

func storage() {
	var metrics = []Metric{}
	var metricsIn = make(chan []Metric)
	var tick = time.NewTicker(pollPeriod).C
	go poller(metricsIn)
	for {
		select {
		case <-tick:
			go poller(metricsIn)
		case metrics = <-metricsIn:
			if debug {
				log.Println("Received metrics from poller", metrics)
			}
		case c := <-metricRequest:
			if debug {
				log.Println("Received request for metrics")
			}
			c <- metrics
			if debug {
				log.Println("Metrics sent to handler", metrics)
			}
		}
	}
}

func main() {
	if graphiteBaseURL == "" {
		graphiteBaseURL = "http://localhost:8080/render"
	}
	if os.Getenv("DEBUG") != "false" && os.Getenv("DEBUG") != "" {
		debug = true
	}
	if graphiteTargets == "" {
		graphiteTargets = "*.*"
	}
	if graphitePollDepth == "" {
		graphitePollDepth = "50s"
	}
	var graphiteParameters = "target=" + graphiteTargets + "&from=-" + graphitePollDepth + "&format=json"
	// stats_counts.*
	url = graphiteBaseURL + "?" + graphiteParameters
	go storage()

	http.HandleFunc("/", serveGraphite)
	log.Fatal(http.ListenAndServe(":8081", nil))
}
