package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"

	tinymux "github.com/alinowrouzii/tiny-mux"
	"github.com/hpcloud/tail"
	"gopkg.in/yaml.v3"
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

type YamlConfig struct {
	Main struct {
		Listen int    `yaml:"listen"`
		Route  string `yaml:"route"`
	} `yaml:"main"`
	Apps []struct {
		AppName interface{} `yaml:"app-name"`
		Logs    []string    `yaml:"logs"`
		Methods struct {
			GET  []int `yaml:"GET"`
			POST []int `yaml:"POST"`
		} `yaml:"methods"`
	} `yaml:"apps"`
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
	h.n.Lock()
	h.m.Lock()
	h.n.Unlock()
	lines := ""
	lines += fmt.Sprintf("# HELP my_nginx_log_exporter_requests_total Number of request with specified status and method.\n")
	lines += fmt.Sprintf("# TYPE my_nginx_log_exporter_requests_total counter\n")
	for method := range *h.methods {
		for status := range (*h.methods)[method] {
			// fmt.Printf("method=%s   status=%s    value=%d\n\n", method, status, (*h.methods)[method][status])
			lines += fmt.Sprintf("my_nginx_log_exporter_requests_total{method=\"%s\", status=\"%s\"} %d\n", method, status, (*h.methods)[method][status])
		}
	}
	w.Write([]byte(lines))
	h.m.Unlock()
}

func main() {

	methods := map[string]map[string]int64{
		"GET": {
			"400": 0,
			"500": 0,
			"200": 0,
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
			status, method, _ := proccessData(line.Text)
			if _, ok := (*metric.methods)[method]; ok {
				if _, ok := (*metric.methods)[method][status]; ok {

					metric.l.Lock()
					metric.n.Lock()
					metric.m.Lock()
					metric.n.Unlock()

					(*metric.methods)[method][status]++

					metric.m.Unlock()
					metric.l.Unlock()
				}
			}
		}
	}()

	mux.GET("/metrics", http.HandlerFunc(metric.metricsHandler))

	http.ListenAndServe(":8888", mux)

	// parseYml()
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

var data = `
a: Easy!
b:
  c: 2
  d: [3, 4]
`

type App struct {
	Logs    []string `yaml:"logs"`
	Methods struct {
		GET  []int `yaml:"GET"`
		POST []int `yaml:"POST"`
	} `yaml:"methods"`
}

func parseYml() {
	parsedYml := make(map[interface{}]interface{})
	data, err := ioutil.ReadFile("./config.yml")

	if err != nil {
		panic(err)
	}
	err = yaml.Unmarshal(data, &parsedYml)
	if err != nil {
		panic(err)
	}

	fmt.Println(parsedYml["apps"])

	apps, ok := parsedYml["apps"]

	if !ok {
		panic("invalid yml format!")
	}

	main, ok := parsedYml["main"]
	parsedListen := "8888"
	parsedRoute := "metrics"
	if ok {
		parsedMain, ok := main.(map[string]interface{})
		if !ok {
			panic("invalid yaml format")
		}

		listen, ok := parsedMain["listen"]
		if ok {
			parsedListen, ok = listen.(string)
			if !ok {
				panic("invalid yaml format")
			}
		}

		route, ok := parsedMain["route"]
		if ok {
			parsedRoute, ok = route.(string)
			if !ok {
				panic("invalid yaml format")
			}
		}
	}

	for appName := range apps.(map[string]interface{}) {

		app := apps.(map[string]interface{})[appName].(map[string]interface{})
		logs, ok := app["logs"]
		if !ok {
			panic("invalid yaml format")
		}

		parsedLogs := make([]string, 0)
		switch t := logs.(type) {
		case []interface{}:
			for _, value := range t {
				castedValue, ok := value.(string)
				if !ok {
					panic("invalid yaml format")
				}
				parsedLogs = append(parsedLogs, castedValue)
			}
		default:
			panic("invalid yaml format")
		}

		fmt.Println("parsedLogs: ", parsedLogs)
		methods, ok := app["methods"]
		if !ok {
			panic("invalid yaml format")
		}

		parsedMethods, ok := methods.(map[string]interface{})
		if !ok {
			panic("invalid yaml format")
		}

		allMethdos := make(map[string][]string)
		for methodName := range parsedMethods {
			switch t := parsedMethods[methodName].(type) {
			case []interface{}:
				for _, value := range t {
					castedValue, ok := value.(string)
					if !ok {
						panic("invalid yaml format")
					}
					allMethdos[methodName] = append(allMethdos[methodName], castedValue)
				}
			default:
				panic("invalid yaml format")
			}
		}

		fmt.Println(allMethdos)
		fmt.Println(parsedLogs)
	}

	fmt.Println(parsedListen)
	fmt.Println(parsedRoute)

}
