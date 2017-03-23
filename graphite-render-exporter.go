package main
import (
  "os"
  "log"
  "fmt"
  "net/http"
  "encoding/json"
  "time"
  "strconv"
  "strings"
)



const pollPeriod = time.Second * 10

var url string 

var tick = time.NewTicker(pollPeriod).C
var metricsOut = make(chan []Metric)
var metricRequest = make(chan bool)

type Target struct {
  Target string `json:"target"`
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
  Name string
  Value float64
}
func (m Metric) String() string {
  return m.Name + " " + strconv.FormatFloat(m.Value, 'f', -1, 64) + "\n"
}

var myClient = &http.Client{Timeout: 10 * time.Second}


func poller(channel chan []Metric) {
//  log.Println("Start polling metrics from Graphite")
  metrics := getMetrics(url)
//  log.Println("Got metrics from Graphite", metrics)
  channel <- metrics
//  log.Println("Poll finished")
}

func getMetrics(url string) []Metric {
  log.Println("Retrieving from: "+url)

  data := []Target{}
  getJson(url, &data)
  log.Println(data)
  m := []Metric{}
  for _, t := range data {
    metric := Metric{Name: strings.Replace(t.Target, ".", "_", -1), Value: getLastNonNullValue(t.Datapoints)}
    m = append(m, metric)
  }
  return m
}


func getLastNonNullValue(d []Datapoint) float64 {
  for i:= len(d)-1 ; i>=0 ; i-- {
    if d[i][0] != nil { return *d[i][0] }
  }
  return 0
}

func getJson(url string, target interface{}) error {
    r, err := myClient.Get(url)
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
//  log.Println("Got request")
  w.Header().Set("Content-Type", "text/plain; charset=utf-8")
  metricRequest <- true
  metrics := <- metricsOut
//  log.Println("Received to handler", metrics)
  c := 0
  for _, m := range metrics {
//    log.Println("writing metric to response #", c ," - " , m)
    fmt.Fprintf(w, m.String() )
    c++
  }
//  log.Println("Finished writing ", c, " metrics")
}

func storage() {
  var metrics = []Metric{}
  var metricsIn = make(chan []Metric)
  go poller(metricsIn)
  for {
    select {
    case <- tick:
      go poller(metricsIn)
      case metrics = <- metricsIn:
//      log.Println("Received metrics from poller", metrics)
    case <- metricRequest:
//      log.Println("Received request for metrics")
      metricsOut <- metrics
//      log.Println("Metrics sent to handler", metrics)
    }
  }
}


func main() {
  var graphiteBaseURL = os.Getenv("GRAPHITE_URL")
  if (graphiteBaseURL == "") { graphiteBaseURL = "http://localhost:8080/render" }
  var graphiteTargets = os.Getenv("TARGETS")
  if (graphiteTargets == "") { graphiteTargets = "*.*" }
  var graphiteParameters = "target="+graphiteTargets+"&from=-50s&format=json"
  // stats_counts.*
  url = graphiteBaseURL+"?"+graphiteParameters
  go storage()
  
  http.HandleFunc("/", serveGraphite)
  log.Fatal(http.ListenAndServe(":8081", nil))
}