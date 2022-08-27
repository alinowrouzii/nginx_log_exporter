package main

import (
	"fmt"
	"net/http"
	"strings"
	"sync"

	tinymux "github.com/alinowrouzii/tiny-mux"
	"github.com/hpcloud/tail"
)

type MetricHandler struct {
	// data mutex
	m sync.Mutex
	// next to access mutext
	n sync.Mutex
	// low priority mutext
	l sync.Mutex
	// for instance: methods['GET'][400] gives the number of requests with method=GET and status code 400
	methods *map[string]map[string]int64
}

func proccessData(line string) (string, string, string) {
	data := splitQoutes(line)
	if len(data) < 4 {
		return "", "", ""
	}

	status := data[1]
	method := strings.Fields(data[2])[0]
	bytesSent := data[3]

	// fmt.Printf("'%s' '%s' '%s'\n\n", status, method, bytesSent)
	return status, method, bytesSent
}

func (h *MetricHandler) metricsHandler(w http.ResponseWriter, r *http.Request) {

	// here we inform our goroutine calculator to does not enter to our critical section if
	// handler wants to show the result to the client
	h.l.Lock()
	h.n.Lock()
	h.m.Lock()
	h.n.Unlock()
	// fmt.Println("Locking process for 10 seconds")
	// time.Sleep(time.Second * 10)
	h.m.Unlock()
	h.l.Unlock()
	w.Write([]byte("helllllo ali"))
}

func main() {
	methods := map[string]map[string]int64{
		"GET": {
			"400": 0,
			"500": 0,
		},
		"POST": {
			"200": 0,
			"400": 0,
			"500": 0,
		},
	}

	mux := tinymux.NewTinyMux()

	metric := &MetricHandler{
		methods: &methods,
	}

	fmt.Println(metric)

	fmt.Println("hello world!")

	go func() {
		t, _ := tail.TailFile("/var/log/nginx/access.log", tail.Config{Follow: true})
		for line := range t.Lines {
			fmt.Println("Im stock here")
			status, method, _ := proccessData(line.Text)
			if _, ok := (*metric.methods)[method]; ok {
				if _, ok := (*metric.methods)[method][status]; ok {

					metric.l.Lock()
					metric.n.Lock()
					metric.m.Lock()
					metric.n.Unlock()
					// fmt.Println("before log")
					// // time.Sleep(time.Second * 10)
					// fmt.Println("after log")
					// fmt.Println("Sleeping for ten seconds...")
					(*metric.methods)[method][status]++

					metric.m.Unlock()
					metric.l.Unlock()
				}
			}

			fmt.Println((*metric.methods)["GET"]["500"])
		}
	}()

	mux.GET("/metrics", http.HandlerFunc(metric.metricsHandler))

	http.ListenAndServe(":8888", mux)
}

func splitQoutes(s string) []string {
	insideQoute := false
	out := strings.FieldsFunc(s, func(r rune) bool {
		if r == '"' {
			insideQoute = !insideQoute
		}
		return r == '"' || (!insideQoute && r == ' ')
	})
	return out
}
