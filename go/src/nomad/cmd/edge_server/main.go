package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"nomad/buildinfo"

	consul "github.com/hashicorp/consul/api"
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

type infoResponse struct {
	Upstreams  map[string]interface{}
	ServerInfo serverInfo
}

func main() {
	address := flag.String("address", "localhost:9999", "listen address")
	service := flag.String("service", "demo-server-main-main", "usptream service")
	flag.Parse()

	uc := &upstreamConfig{Service: *service}
	go func() {
		for {
			log.Printf("refreshing upstream services for service %q", uc.Service)
			if err := uc.refresh(); err != nil {
				log.Printf("warning: error while refreshing upstreams: %s", err)
			}
			time.Sleep(2 * time.Second)
		}
	}()

	handler := http.NewServeMux()
	handler.HandleFunc("/", logRequest(func(w http.ResponseWriter, r *http.Request) {
		rsp := infoResponse{
			Upstreams:  requestUpstreams(uc.getAddrs()),
			ServerInfo: getServerInfo(),
		}
		data, _ := json.MarshalIndent(rsp, "", "    ")
		fmt.Fprintln(w, string(data))
	}))

	server := &http.Server{Addr: *address, Handler: handler}
	log.Fatal(server.ListenAndServe())
}

func logRequest(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s", r.Method, r.URL.Path)
		h(w, r)
	}
}

// Comma-splits, trims, sorts.
func parseUpstreams(s string) []string {
	fields := strings.Split(s, ",")
	for i, f := range fields {
		fields[i] = strings.TrimSpace(f)
	}
	sort.Strings(fields)
	return fields
}

// Fetches "/info" from the web server at each address and returns the
// parsed responses as a map keyed by request URL.
func requestUpstreams(addrs []string) map[string]interface{} {
	fetch := func(url string) interface{} {
		client := &http.Client{Timeout: 5 * time.Second}
		rsp, err := client.Get(url)
		if err != nil {
			return fmt.Sprintf("request failed: %s", err)
		}
		data, err := ioutil.ReadAll(rsp.Body)
		rsp.Body.Close()
		if err != nil {
			return fmt.Sprintf("error reading response body: %s", err)
		}
		var result interface{}
		if err := json.Unmarshal(data, &result); err == nil {
			return result
		}
		return tryJSON(data)
	}
	type result struct {
		url      string
		response interface{}
	}
	ch := make(chan result)
	for _, addr := range addrs {
		go func(addr string) {
			url := fmt.Sprintf("http://%s/info", addr)
			ch <- result{url, fetch(url)}
		}(addr)
	}
	answer := make(map[string]interface{}, len(addrs))
	for range addrs {
		r := <-ch
		answer[r.url] = r.response
	}
	return answer
}

// If data can be parsed as JSON, returns the structured value.
// Otherwise returns string(data).
func tryJSON(data []byte) interface{} {
	var x interface{}
	if err := json.Unmarshal(data, &x); err == nil {
		return x
	}
	return string(data)
}

// Stores the current set of upstream hosts.
type upstreamConfig struct {
	Service string

	addrs []string
	mu    sync.Mutex
}

// Returns the upstream hosts as "address:port" strings.
func (c *upstreamConfig) getAddrs() []string {
	c.mu.Lock()
	answer := c.addrs
	c.mu.Unlock()
	return answer
}

// Asks consul for the current of upstream hosts.
func (c *upstreamConfig) refresh() error {
	config := consul.DefaultConfig()
	client, err := consul.NewClient(config)
	if err != nil {
		return err
	}
	options := &consul.QueryOptions{AllowStale: true, RequireConsistent: false}
	entries, _, err := client.Health().Service(c.Service, "", true, options)
	if err != nil {
		return err
	}
	newAddrs := make([]string, len(entries))
	for i, entry := range entries {
		// KLUDGE: use the node's address instead of the
		// service's address because Nomad will use the first
		// network interface it finds when we specify the listen
		// address as "0.0.0.0", and in Vagrant-land that is
		// invariably the wrong one to use.
		newAddrs[i] = fmt.Sprintf("%s:%d", entry.Node.Address, entry.Service.Port)
		log.Printf("DEBUG: entry.Node.Address=%v entry.Service.Address=%v entry.Service.Port=%v",
			entry.Node.Address, entry.Service.Address, entry.Service.Port)
	}
	c.mu.Lock()
	c.addrs = newAddrs
	c.mu.Unlock()
	return nil
}
