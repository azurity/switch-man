package main

import (
	"embed"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strconv"
	"sync"
	"sync/atomic"

	"github.com/azurity/go-onefile"
)

//go:embed frontend
var frontend embed.FS

type Dispatcher struct {
	Id          int32
	Serve       *httputil.ReverseProxy
	FromBaseURL *url.URL
	ToBaseURL   *url.URL
}

func newDispatcher(desc DispatcherSerial) (*Dispatcher, error) {
	if desc.From[len(desc.From)-1] != '/' {
		desc.From = fmt.Sprintf("%s/", desc.From)
	}
	from, err := url.Parse(desc.From)
	if err != nil {
		return nil, err
	}
	to, err := url.Parse(desc.To)
	if err != nil {
		return nil, err
	}
	return &Dispatcher{
		Id:          desc.Id,
		Serve:       httputil.NewSingleHostReverseProxy(to),
		FromBaseURL: from,
		ToBaseURL:   to,
	}, nil
}

type DispatcherSerial struct {
	Id   int32  `json:"id"`
	From string `json:"from"`
	To   string `json:"to"`
}

func load() ([]*Dispatcher, *atomic.Int32) {
	list := []DispatcherSerial{}
	data, err := os.ReadFile("config.json")
	if err != nil {
		log.Panicln(err)
	}
	err = json.Unmarshal(data, &list)
	if err != nil {
		log.Panicln(err)
	}
	ret := []*Dispatcher{}
	for index, it := range list {
		it.Id = int32(index)
		dispatcher, err := newDispatcher(it)
		if err != nil {
			continue
		}
		ret = append(ret, dispatcher)
	}
	id := new(atomic.Int32)
	id.Store(int32(len(list)))
	return ret, id
}

func save(list []*Dispatcher) error {
	ret := []DispatcherSerial{}
	for _, it := range list {
		ret = append(ret, DispatcherSerial{
			Id:   it.Id,
			From: it.FromBaseURL.String(),
			To:   it.ToBaseURL.String(),
		})
	}
	data, err := json.Marshal(ret)
	if err != nil {
		return err
	}
	return os.WriteFile("config.json", data, 0666)
}

func main() {
	port := flag.Int("mainPort", 80, "entry serve port")
	manPort := flag.Int("managePort", 8080, "manager serve port")
	flag.Parse()

	if *port == *manPort || *port <= 0 || *port > 32767 || *manPort <= 0 || *manPort > 32767 {
		log.Panicln("invalid port")
	}

	dispatchers, id := load()
	mtx := sync.RWMutex{}

	main := http.NewServeMux()
	manager := http.NewServeMux()

	main.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		mtx.RLock()
		for _, it := range dispatchers {
			if r.Host == it.FromBaseURL.Hostname() {
				basePath := it.FromBaseURL.Path
				basePath = basePath[:len(basePath)-1]
				if basePath == r.URL.Path[:len(basePath)] {
					mtx.RUnlock()
					r.URL.Path = r.URL.Path[len(basePath):]
					it.Serve.ServeHTTP(w, r)
					return
				}
			}
		}
		mtx.RUnlock()
		w.WriteHeader(http.StatusNotFound)
	})

	frontendFS, _ := fs.Sub(frontend, "frontend")
	handle := onefile.New(frontendFS, nil, "")
	manager.Handle("/", handle)

	manager.HandleFunc("/list", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			mtx.RLock()
			defer mtx.RUnlock()
			list := []DispatcherSerial{}
			for _, it := range dispatchers {
				list = append(list, DispatcherSerial{
					Id:   it.Id,
					From: it.FromBaseURL.String(),
					To:   it.ToBaseURL.String(),
				})
			}
			data, _ := json.Marshal(list)
			w.WriteHeader(http.StatusOK)
			w.Write(data)
		} else if r.Method == "POST" {
			data, _ := io.ReadAll(r.Body)
			var desc DispatcherSerial
			err := json.Unmarshal(data, &desc)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			desc.Id = id.Add(1)
			dispatcher, err := newDispatcher(desc)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			mtx.Lock()
			mtx.Unlock()
			dispatchers = append(dispatchers, dispatcher)
			w.WriteHeader(http.StatusOK)
			save(dispatchers)
		} else if r.Method == "DELETE" {
			delId, err := strconv.ParseInt(r.URL.Query().Get("id"), 10, 32)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			mtx.Lock()
			mtx.Unlock()
			for index, it := range dispatchers {
				if it.Id == int32(delId) {
					dispatchers = append(dispatchers[:index], dispatchers[index+1:]...)
					w.WriteHeader(http.StatusOK)
					save(dispatchers)
					return
				}
			}
			w.WriteHeader(http.StatusBadRequest)
			return
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	})

	go http.ListenAndServe(fmt.Sprintf(":%d", *port), main)
	http.ListenAndServe(fmt.Sprintf(":%d", *manPort), manager)
}
