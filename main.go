package main

import (
	"bufio"
	"log"
	"net"
	"strings"
	"sync"
)

const (
	ResponseError = "ERROR"
)

// PackageStore is the interface for storing packages
type PackageStore interface {
	Get(string) (*Package, bool)
	Delete(string)
	Put(*Package)
	Size() int
}

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

func (p *PackageIndexer) Add(name string, pkg *Package) bool {
	p.m.Lock()
	defer p.m.Unlock()
	if _, ok := p.store.Get(name); ok {
		return true
	}
	p.store.Put(pkg)
	p.addDependents(mapKeys(pkg.dependencies), name)
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

func main() {
	p := &PackageIndexer{
		conChan: make(chan net.Conn, 10),
		m:       &sync.RWMutex{},
		store:   NewMapStore(),
	}
	p.listenAndServe()
}

type PackageIndexer struct {
	m       *sync.RWMutex
	conChan chan net.Conn
	store   PackageStore
}

func (p *PackageIndexer) listenAndServe() {
	ln, err := net.Listen("tcp", ":8080")

	if err != nil {
		log.Fatalf("could not start tcp server: %s\n", err.Error())
	}
	go func() {
		for {
			conn := <-p.conChan
			go p.handleRequest(conn)
		}
	}()

	for {
		conn, err := ln.Accept()
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
		request, _ := bufio.NewReader(conn).ReadString('\n')

		Request, success := parseRequestString(request)

		if !success {
			conn.Write([]byte(ResponseError + "\n"))
			continue
		}

		if Request.command == "INDEX" {
			if len(Request.dependencies) > 0 && !p.find(Request.dependencies...) {
				//could not find all dependencies
				conn.Write([]byte("FAIL\n"))
				continue
			}
			p.Add(Request.pkg, &Package{name: Request.pkg, dependencies: sliceToMap(Request.dependencies), dependents: make(map[string]interface{})})
			_, err := conn.Write([]byte("OK\n"))

			if err != nil {
				//TODO do something
			}
			continue
		}

		if Request.command == "QUERY" {
			if p.Query(Request.pkg) {
				_, err := conn.Write([]byte("OK\n"))
				if err != nil {
					//TODO something
				}
				continue
			}
			conn.Write([]byte("FAIL\n"))
			continue
		}

		if Request.command == "REMOVE" {
			pkg, ok := p.Get(Request.pkg)
			if !ok {
				_, err := conn.Write([]byte("OK\n"))
				if err != nil {
				}
				continue
			}

			if len(pkg.dependents) > 0 && p.find(mapKeys(pkg.dependents)...) {
				_, err := conn.Write([]byte("FAIL\n"))
				if err != nil {
				}
				continue
			}

			if p.Remove(Request.pkg) {
				_, err := conn.Write([]byte("OK\n"))
				if err != nil {
				}
				continue
			}
			_, err := conn.Write([]byte("FAIL\n"))
			if err != nil {
			}
			continue
		}
	}

}

func (p *PackageIndexer) addDependents(packages []string, dependent string) {
	for _, pkg := range packages {
		currentPackage, ok := p.store.Get(pkg)
		if !ok {

		}
		currentPackage.dependents[dependent] = struct{}{}
	}
}

func (p *PackageIndexer) removeDependents(packages []string, dependent string) {
	for _, dep := range packages {
		pkg, ok := p.store.Get(dep)
		if !ok {
			//something went very wrong
		}
		delete(pkg.dependents, dependent)
	}
}

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

func parseRequestString(s string) (*Request, bool) {
	splitRequest := strings.Split(s, "|")
	if len(splitRequest) != 3 {
		return nil, false
	}

	command := splitRequest[0]
	if command != "INDEX" && command != "QUERY" && command != "REMOVE" {
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
