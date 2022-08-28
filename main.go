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

const (
	defaultPort  = "4000"
	defaultRoute = "/mterics"
)

type metricHandler struct {
	// data mutex
	m sync.Mutex
	// next to access mutex
	n sync.Mutex
	// low priority mutexes
	l sync.Mutex
	// for instance: methods['SHIT']['GET'][400] gives the number of requests with method=GET and status code 400 of app=shit
	methods map[string]map[string]map[string]int64
	//
	logs map[string][]string
}

func extractDataFromLine(line string) (string, string, string) {
	data := splitQoutes(line)
	if len(data) < 4 {
		return "", "", ""
	}

	status := data[1]
	method := strings.Fields(data[2])[0]
	bytesSent := data[3]

	return status, method, bytesSent
}

func (metric *metricHandler) processData() {

	for appName := range metric.methods {
		// appMethods := metric.methods[appName]
		appLogs := metric.logs[appName]

		for _, logPath := range appLogs {
			go func(logPath string) {
				t, err := tail.TailFile(logPath, tail.Config{Follow: true})
				if err != nil {
					panic(err)
				}
				for line := range t.Lines {
					status, method, _ := extractDataFromLine(line.Text)
					metric.l.Lock()
					metric.n.Lock()
					metric.m.Lock()
					metric.n.Unlock()
					if _, ok := metric.methods[appName][method]; ok {
						if _, ok := metric.methods[appName][method][status]; ok {
							metric.methods[appName][method][status]++
						}
					}
					metric.m.Unlock()
					metric.l.Unlock()
				}
			}(logPath)
		}

	}

}

func (metric *metricHandler) metricsHandler(w http.ResponseWriter, r *http.Request) {

	// here we inform our goroutine calculator to does not enter to our critical section if
	// handler wants to show the result to the client
	metric.n.Lock()
	metric.m.Lock()
	metric.n.Unlock()
	lines := ""
	lines += fmt.Sprintf("# HELP {name_space}_log_exporter_requests_total Number of request with specified status and method.\n")
	lines += fmt.Sprintf("# TYPE {name_space}_log_exporter_requests_total counter\n")
	for appName := range metric.methods {
		for method := range metric.methods[appName] {
			for status := range metric.methods[appName][method] {
				lines += fmt.Sprintf("%s_log_exporter_requests_total{method=\"%s\", status=\"%s\"} %d\n", appName, method, status, metric.methods[appName][method][status])
			}
		}
	}
	w.Write([]byte(lines))
	metric.m.Unlock()
}

func main() {

	// methods := map[string]map[string]map[string]int64{
	// 	"SHIT": {"GET": {
	// 		"400": 0,
	// 		"500": 0,
	// 		"200": 0,
	// 	},
	// 		"POST": {
	// 			"200": 0,
	// 			"400": 0,
	// 			"500": 0,
	// 		},
	// 	},
	// }
	methods, logs, listenPort, route := parseYml()

	mux := tinymux.NewTinyMux()

	metric := &metricHandler{
		methods: methods,
		logs:    logs,
	}

	fmt.Println(metric)

	fmt.Println("hello world!")

	go metric.processData()

	fmt.Println(route, listenPort)
	mux.GET(route, http.HandlerFunc(metric.metricsHandler))

	http.ListenAndServe(fmt.Sprintf(":%s", listenPort), mux)

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

func parseYml() (map[string]map[string]map[string]int64, map[string][]string, string, string) {
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
	parsedListen := defaultPort
	parsedRoute := defaultRoute
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

	methodsResult := make(map[string]map[string]map[string]int64)
	logsResult := make(map[string][]string)

	for appName := range apps.(map[string]interface{}) {
		app := apps.(map[string]interface{})[appName].(map[string]interface{})
		logs, ok := app["logs"]
		if !ok {
			panic("invalid yaml format")
		}
		methodsResult[appName] = make(map[string]map[string]int64)

		parsedLogs := make([]string, 0)
		switch t := logs.(type) {
		case []interface{}:
			for _, value := range t {
				castedValue, ok := value.(string)
				if !ok {
					panic("invalid yaml format")
				}
				parsedLogs = append(parsedLogs, castedValue)
				logsResult[appName] = append(logsResult[appName], castedValue)
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

		for methodName := range parsedMethods {
			methodsResult[appName][methodName] = make(map[string]int64)
			switch t := parsedMethods[methodName].(type) {
			case []interface{}:
				for _, status := range t {
					castedStatus, ok := status.(string)
					if !ok {
						panic("invalid yaml format")
					}
					methodsResult[appName][methodName][castedStatus] = 0
					// allMethdos[methodName] = append(allMethdos[methodName], castedValue)
				}
			default:
				panic("invalid yaml format")
			}
		}
	}

	fmt.Println(parsedListen)
	fmt.Println(parsedRoute)
	fmt.Println(methodsResult)
	fmt.Println(logsResult)
	return methodsResult, logsResult, parsedListen, parsedRoute
}
