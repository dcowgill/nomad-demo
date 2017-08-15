// This web server handles the following paths:
//
//	/echo
//
//		Echos the request body in the response.
//
//
//	/exit/<code>
//
//		If <code> is in the range [0, 255], the server will
//		immediately call os.Exit with that value.
//
//	/info
//
//		Prints information about the server as JSON.
//
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync/atomic"
	"time"

	"nomad/buildinfo"
)

var started = time.Now()

type serverInfo struct {
	Build    buildinfo.Info
	Started  time.Time
	Hostname string
	PID      int
}

func getServerInfo() serverInfo {
	hostname, _ := os.Hostname()
	pid := os.Getpid()
	return serverInfo{
		Build:    buildinfo.Get(),
		Started:  started,
		Hostname: hostname,
		PID:      pid,
	}
}

func main() {
	portEnv := flag.String("port-env", "PORT", "env var with listen port")
	flag.Parse()

	// KLUDGE: we would prefer to get the full address from the
	// environment (i.e. including the IP address), but nomad will
	// not necessarily choose the right network interface.
	port := os.Getenv(*portEnv)
	if port == "" {
		port = "8888"
	}
	address := "0.0.0.0:" + port

	var exitCode int64 // server will exit with this value

	handler := http.NewServeMux()
	server := &http.Server{Addr: address, Handler: handler}

	handler.HandleFunc("/echo", logRequest(func(w http.ResponseWriter, r *http.Request) {
		data, err := ioutil.ReadAll(r.Body)
		r.Body.Close()
		if err != nil {
			http.Error(w, fmt.Sprintf("error reading response body: %s", err), 400)
			return
		}
		w.Write(data)
	}))

	handler.HandleFunc("/exit/", logRequest(func(w http.ResponseWriter, r *http.Request) {
		code, err := strconv.ParseInt(r.URL.Path[6:], 10, 64)
		if err != nil || code < 0 || code > 255 {
			http.Error(w, "exit code must be >=0 and <=255", 400)
			return
		}
		atomic.StoreInt64(&exitCode, code)
		go server.Shutdown(context.Background())
	}))

	handler.HandleFunc("/info", logRequest(func(w http.ResponseWriter, r *http.Request) {
		data, _ := json.MarshalIndent(getServerInfo(), "", "    ")
		fmt.Fprintln(w, string(data))
	}))

	if err := server.ListenAndServe(); err != nil {
		fmt.Fprintf(os.Stderr, "error from ListenAndServe: %s\n", err)
	}
	os.Exit(int(atomic.LoadInt64(&exitCode)))
}

func logRequest(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s", r.Method, r.URL.Path)
		h(w, r)
	}
}
