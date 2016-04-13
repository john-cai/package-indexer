package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"strings"
	"sync"
	"time"
)

const (
	ResponseError = "ERROR"
)

// PackageStore is the interface for storing packages
type PackageStore interface {
	Add(string, *Package) bool
	Get(string) (*Package, bool)
	Remove(string) bool
	find(...string) bool
}

func NewMapStore() *MapStore {
	return &MapStore{
		m:     &sync.RWMutex{},
		store: make(map[string]*Package),
	}
}

type MapStore struct {
	m     *sync.RWMutex
	store map[string]*Package
}

func (m *MapStore) Add(name string, pkg *Package) bool {
	m.m.Lock()
	defer m.m.Unlock()
	if _, ok := m.store[name]; ok {
		return ok
	}
	m.store[name] = pkg
	m.addDependents(mapKeys(pkg.dependencies), name)
	return true
}

func (m *MapStore) Get(name string) (*Package, bool) {
	m.m.RLock()
	defer m.m.RUnlock()
	p, ok := m.store[name]
	return p, ok
}

func (m *MapStore) Remove(name string) bool {
	m.m.Lock()
	defer m.m.Unlock()
	if _, ok := m.store[name]; !ok {
		return true
	}
	//check dependents
	p, _ := m.store[name]
	if m.find(mapKeys(p.dependents)...) {
		return false
	}
	delete(m.store, name)
	m.removeDependents(mapKeys(p.dependencies), name)
	return true
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
		store:   NewMapStore(),
		conChan: make(chan net.Conn, 10),
		reqChan: make(chan string),
	}
	p.listenAndServe()
}

type PackageIndexer struct {
	store   PackageStore
	conChan chan net.Conn
	reqChan chan string
}

//TODO rate limit

func (p *PackageIndexer) listenAndServe() {
	ln, err := net.Listen("tcp", ":8080")

	if err != nil {
		log.Fatalf("could not start tcp server: %s\n", err.Error())
	}
	go func() {
		for {
			conn := <-p.conChan
			fmt.Println("pulled off of the channel")
			go p.receiveRequest(conn)
		}
	}()

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Printf("error accepting connection: %s\n", err.Error())
		}
		fmt.Println("adding to the channel!!")
		p.conChan <- conn
		fmt.Println("added to the channel")
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

func (p *PackageIndexer) handleRequest(conn net.Conn, timer *time.Timer) {
	for {
		select {
		case request := <-p.reqChan:
			Request, success := parseRequestString(request)

			if !success {
				conn.Write([]byte(ResponseError + "\n"))
				continue
			}

			if Request.command == "INDEX" {
				if len(Request.dependencies) > 0 && !p.store.find(Request.dependencies...) {
					//could not find all dependencies
					conn.Write([]byte("FAIL\n"))
					continue
				}
				p.store.Add(Request.pkg, &Package{name: Request.pkg, dependencies: sliceToMap(Request.dependencies), dependents: make(map[string]interface{})})
				_, err := conn.Write([]byte("OK\n"))

				if err != nil {
					//TODO do something
				}
				continue
			}

			if Request.command == "QUERY" {
				if p.store.find(Request.pkg) {
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
				pkg, ok := p.store.Get(Request.pkg)
				if !ok {
					_, err := conn.Write([]byte("OK\n"))
					if err != nil {
					}
					continue
				}

				if len(pkg.dependents) > 0 && p.store.find(mapKeys(pkg.dependents)...) {
					_, err := conn.Write([]byte("FAIL\n"))
					if err != nil {
					}
					continue
				}

				if p.store.Remove(Request.pkg) {
					//REMOVE pkg as a dependent
					//dependencies := pkg.dependencies
					//p.removeDependents(mapKeys(dependencies), Request.pkg)
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
		case <-timer.C:
			conn.Close()
			return
		}
	}
}

func (p *PackageIndexer) receiveRequest(conn net.Conn) {
	timer := time.NewTimer(10 * time.Second)
	go p.handleRequest(conn, timer)
	for {
		request, _ := bufio.NewReader(conn).ReadString('\n')
		p.reqChan <- request
		timer.Reset(10 * time.Second)
	}
}

func (m *MapStore) addDependents(packages []string, dependent string) {
	for _, pkg := range packages {
		currentPackage, ok := m.store[pkg]
		if !ok {
			//something went very wrong
		}
		currentPackage.dependents[dependent] = struct{}{}
	}
}

func (m *MapStore) removeDependents(packages []string, dependent string) {
	for _, dep := range packages {
		pkg, ok := m.store[dep]
		if !ok {
			//something went very wrong
		}
		delete(pkg.dependents, dependent)
	}
}

func (m *MapStore) find(pkgs ...string) bool {
	if len(pkgs) == 0 {
		return false
	}
	if len(m.store) == 0 {
		return false
	}
	for _, pkg := range pkgs {
		if _, ok := m.store[pkg]; !ok {
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
