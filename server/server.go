package server

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"strings"
	"sync"
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
}

// mapStore is an implementation of PackageStore using a standard library map
type mapStore struct {
	m map[string]*Package
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

func NewPackageIndexer(limit int, store PackageStore, port int) *PackageIndexer {
	return &PackageIndexer{
		conChan: make(chan net.Conn, limit),
		m:       &sync.RWMutex{},
		store:   store,
		port:    port,
	}
}

func (p *PackageIndexer) Add(pkg *Package) bool {
	p.m.Lock()
	defer p.m.Unlock()
	if len(pkg.dependencies) > 0 && !p.find(mapKeys(pkg.dependencies)...) {
		return false
	}

	if _, ok := p.store.Get(pkg.name); ok {
		return true
	}
	p.store.Put(pkg)
	p.addDependents(mapKeys(pkg.dependencies), pkg.name)
	return true
}

func (p *PackageIndexer) Get(name string) (*Package, bool) {
	p.m.RLock()
	defer p.m.RUnlock()
	var ok bool
	pkg, ok := p.store.Get(name)
	return pkg, ok
}

func (p *PackageIndexer) Remove(name string) bool {
	p.m.Lock()
	defer p.m.Unlock()
	pkg, ok := p.store.Get(name)
	if !ok {
		return true
	}
	//check dependents

	if p.find(mapKeys(pkg.dependents)...) {
		return false
	}
	p.store.Delete(name)
	p.removeDependents(mapKeys(pkg.dependencies), name)
	return true
}

func (p *PackageIndexer) Query(name string) bool {
	p.m.RLock()
	defer p.m.RUnlock()
	return p.find(name)
}

type Package struct {
	name         string
	dependents   map[string]interface{}
	dependencies map[string]interface{}
}

type Request struct {
	command      string
	pkg          string
	dependencies []string
}

type PackageIndexer struct {
	m       *sync.RWMutex
	conChan chan net.Conn
	store   PackageStore
	port    int
}

func (p *PackageIndexer) ListenAndServe() {
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", p.port))

	if err != nil {
		log.Fatalf("could not start tcp server: %s\n", err.Error())
	}
	// use a buffered channel to rate limit connections
	go func() {
		for {
			conn := <-p.conChan
			go p.handleRequest(conn)
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

func (p *PackageIndexer) handleRequest(conn net.Conn) {
	for {
		request, err := bufio.NewReader(conn).ReadString('\n')
		//METRICS: start request handle timer
		//METRICS: defer calculate total time for request

		// If we have an error, then we need to close the connection
		if err != nil {
			log.Printf("error reading from client %s", err.Error())
			//METRICS: increment connection closed count
			err := conn.Close()

			if err != nil {
				log.Printf("error closing connection %s", err.Error())
			}
			return
		}

		Request, success := parseRequestString(request)

		if !success {
			_, err := conn.Write([]byte(ResponseError + "\n"))
			if err != nil {
				log.Printf("error writing to connection %s", err.Error())
			}
			continue
		}

		// Handle valid commands
		if Request.command == CmdIndex {
			//METRICS: increment command index count
			if !p.Add(&Package{
				name:         Request.pkg,
				dependencies: sliceToMap(Request.dependencies),
				dependents:   make(map[string]interface{}),
			}) {
				conn.Write([]byte(fmt.Sprintf("%s\n", ResponseFail)))
				continue
			}
			_, err := conn.Write([]byte(fmt.Sprintf("%s\n", ResponseOK)))

			if err != nil {
				log.Printf("error writing to connection %s", err.Error())
			}
			continue
		}

		if Request.command == CmdQuery {
			if p.Query(Request.pkg) {
				_, err := conn.Write([]byte(fmt.Sprintf("%s\n", ResponseOK)))
				if err != nil {
					log.Printf("error writing to connection %s", err.Error())
				}
				continue
			}
			_, err := conn.Write([]byte(fmt.Sprintf("%s\n", ResponseFail)))
			if err != nil {
				log.Printf("error writing to connection %s", err.Error())
			}
			continue
		}

		if Request.command == CmdRemove {
			pkg, ok := p.Get(Request.pkg)
			if !ok {
				_, err := conn.Write([]byte(fmt.Sprintf("%s\n", ResponseOK)))
				if err != nil {
				}
				continue
			}

			if len(pkg.dependents) > 0 && p.find(mapKeys(pkg.dependents)...) {
				_, err := conn.Write([]byte(fmt.Sprintf("%s\n", ResponseFail)))
				if err != nil {
					log.Printf("error writing to connection %s", err.Error())
				}
				continue
			}

			if p.Remove(Request.pkg) {
				_, err := conn.Write([]byte(fmt.Sprintf("%s\n", ResponseOK)))
				if err != nil {
					log.Printf("error writing to connection %s", err.Error())
				}
				continue
			}
			_, err := conn.Write([]byte(fmt.Sprintf("%s\n", ResponseFail)))
			if err != nil {
				log.Printf("error writing to connection %s", err.Error())
			}
			continue
		}
	}
}

// for every package in 'packages', 'dependent' will not be a dependent
func (p *PackageIndexer) addDependents(packages []string, dependent string) {
	for _, pkg := range packages {
		currentPackage, ok := p.store.Get(pkg)
		if !ok {
			log.Fatalf("missing package %s", pkg)
		}
		currentPackage.dependents[dependent] = struct{}{}
	}
}

// for every package in 'packages', 'dependent' will no longer be a dependent
func (p *PackageIndexer) removeDependents(packages []string, dependent string) {
	for _, dep := range packages {
		pkg, ok := p.store.Get(dep)
		if !ok {
			log.Fatalf("missing package %s", pkg)
		}
		delete(pkg.dependents, dependent)
	}
}

// search function to find a package in the package store
func (p *PackageIndexer) find(pkgs ...string) bool {
	if len(pkgs) == 0 {
		return false
	}
	if p.store.Size() == 0 {
		return false
	}
	for _, pkg := range pkgs {
		if _, ok := p.store.Get(pkg); !ok {
			return false
		}
	}
	return true
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
