package server

import (
	"fmt"
	"log"
	"net"
	"strings"
	"sync"

	"github.com/pborman/uuid"
)

const (
	ResponseError = "ERROR"
	ResponseOK    = "OK"
	ResponseFail  = "FAIL"

	CmdIndex  = "INDEX"
	CmdQuery  = "QUERY"
	CmdRemove = "REMOVE"
)

// PackageStore is the interface for storing packages
type PackageStore interface {
	Get(string) (*Package, bool)
	Delete(string)
	Put(*Package)
	Size() int
	Lock()
	Unlock()
	RLock()
	RUnlock()
}

// mapStore is an implementation of PackageStore using a standard library map
type mapStore struct {
	l sync.RWMutex
	m map[string]*Package
}

func (m *mapStore) Lock() {
	m.l.Lock()
}

func (m *mapStore) Unlock() {
	m.l.Unlock()
}

func (m *mapStore) RLock() {
	m.l.RLock()
}

func (m *mapStore) RUnlock() {
	m.l.RUnlock()
}

func (m *mapStore) Get(p string) (*Package, bool) {
	pkg, ok := m.m[p]
	return pkg, ok
}

func (m *mapStore) Delete(p string) {
	delete(m.m, p)
}

func (m *mapStore) Put(p *Package) {
	m.m[p.name] = p
}

func (m *mapStore) Size() int {
	return len(m.m)
}

func NewMapStore() *mapStore {
	return &mapStore{
		m: make(map[string]*Package),
	}
}

func NewPackageIndexer(rateLimit, numWorkers int, store PackageStore, port int) *PackageIndexer {

	p := &PackageIndexer{
		conChan:    make(chan net.Conn, rateLimit),
		workerChan: make(chan *Worker, numWorkers),
		port:       port,
	}
	for i := 0; i < numWorkers; i++ {
		p.workerChan <- &Worker{id: uuid.New(), store: store, workerChan: p.workerChan}
	}
	return p
}

type Package struct {
	name         string
	dependents   map[string]interface{}
	dependencies []string
}

type Request struct {
	command      string
	pkg          string
	dependencies []string
}

type PackageIndexer struct {
	conChan    chan net.Conn
	port       int
	workers    []Worker
	workerChan chan *Worker
}

func (p *PackageIndexer) ListenAndServe() {
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", p.port))

	if err != nil {
		log.Fatalf("could not start tcp server: %s\n", err.Error())
	}
	// use a buffered channel to rate limit connections
	go func() {
		for {
			worker := <-p.workerChan
			conn := <-p.conChan
			go func() {
				worker.handleRequest(conn)
				p.workerChan <- worker
			}()
		}
	}()

	for {
		conn, err := ln.Accept()
		//METRIC: increment total connections count
		if err != nil {
			log.Printf("error accepting connection: %s\n", err.Error())
		}
		p.conChan <- conn
	}
}

func sliceToMap(s []string) map[string]interface{} {
	m := make(map[string]interface{})

	for _, v := range s {
		m[v] = struct{}{}
	}
	return m
}

func mapKeys(m map[string]interface{}) []string {
	s := make([]string, 0)
	for k, _ := range m {
		s = append(s, k)
	}
	return s
}

// parses the request string
func parseRequestString(s string) (*Request, bool) {
	splitRequest := strings.Split(s, "|")
	if len(splitRequest) != 3 {
		return nil, false
	}

	command := splitRequest[0]
	if command != CmdIndex && command != CmdQuery && command != CmdRemove {
		//invalid command
		return nil, false
	}

	pkg := splitRequest[1]

	if pkg == "" {
		return nil, false
	}

	deps := strings.TrimSpace(splitRequest[2])
	dependencies := make([]string, 0)
	if deps != "" {
		dependencies = strings.Split(deps, ",")
	}

	return &Request{
		command:      command,
		pkg:          pkg,
		dependencies: dependencies,
	}, true
}
