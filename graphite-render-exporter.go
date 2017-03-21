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
var graphiteBaseURL = os.Getenv("GRAPHITE_URL")
//"http://localhost:8080/render"
var graphiteParameters = "target="+os.Getenv("TARGETS")+"&from=-50s&format=json"
// stats_counts.*
var url = graphiteBaseURL+"?"+graphiteParameters
var  metrics = []Metric{}
var  tick = time.NewTicker(pollPeriod).C
var shutdownPoller = make(chan bool)

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


func poll(url string) {
  for {
    select {
      case <- tick :
      log.Println("Start polling metrics from Graphite")
        metrics = getMetrics(url)
      log.Println("Poll finished")
      case <- shutdownPoller:
        return
    }
  }
}

func getMetrics(url string) []Metric {
  println("Retrieving from: "+url)

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
 // log.Println("Got request")
  w.Header().Set("Content-Type", "text/plain; charset=utf-8")
  c := 0
  for i, m := range metrics {
    //log.Println("writing metric to response #", i ," - " , m)
    fmt.Fprintf(w, m.String() )
    c++
  }
 // log.Println("Finished writing ", c, " metrics")
}


func main() {

  go poll(url)
  http.HandleFunc("/", serveGraphite)
  log.Fatal(http.ListenAndServe(":8081", nil))
}