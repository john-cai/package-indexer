package server

import (
	"bufio"
	"fmt"
	"log"
	"net"
)

type Worker struct {
	id         string
	store      PackageStore
	workerChan chan *Worker
}

func (w *Worker) handleRequest(conn net.Conn) {
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
			if !w.Add(&Package{
				name:         Request.pkg,
				dependencies: Request.dependencies,
				dependents:   make(map[string]interface{}),
			}) {
				conn.Write([]byte(fmt.Sprintf("%s\n", ResponseFail)))
				continue
			}
			_, err := conn.Write([]byte(fmt.Sprintf("%s\n", ResponseOK)))

			if err != nil {
				log.Printf("error writing to connection %s", err.Error())
			}
			log.Printf("added %v", Request.pkg)
			continue
		}

		if Request.command == CmdQuery {
			if w.Query(Request.pkg) {
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
			_, ok := w.Get(Request.pkg)
			if !ok {
				_, err := conn.Write([]byte(fmt.Sprintf("%s\n", ResponseOK)))
				if err != nil {
				}
				continue
			}

			if w.Remove(Request.pkg) {
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

func (w *Worker) Add(pkg *Package) bool {
	w.store.Lock()
	defer w.store.Unlock()
	if len(pkg.dependencies) > 0 && !w.find(pkg.dependencies...) {
		return false
	}

	if _, ok := w.store.Get(pkg.name); ok {
		return true
	}
	w.store.Put(pkg)
	w.addDependents(pkg.dependencies, pkg.name)
	return true
}

func (w *Worker) Get(name string) (*Package, bool) {
	w.store.RLock()
	defer w.store.RUnlock()
	var ok bool
	pkg, ok := w.store.Get(name)
	return pkg, ok
}

func (w *Worker) Remove(name string) bool {
	w.store.Lock()
	defer w.store.Unlock()

	pkg, ok := w.store.Get(name)
	if !ok {
		return true
	}
	if len(pkg.dependents) > 0 && w.find(mapKeys(pkg.dependents)...) {
		return false
	}
	w.store.Delete(name)
	w.removeDependents(pkg.dependencies, name)

	return true
}

func (w *Worker) Query(name string) bool {
	w.store.RLock()
	defer w.store.RUnlock()
	return w.find(name)
}

// for every package in 'packages', 'dependent' will not be a dependent
func (w *Worker) addDependents(packages []string, dependent string) {
	for _, pkg := range packages {
		currentPackage, ok := w.store.Get(pkg)
		if !ok {
			log.Fatalf("missing package %s", pkg)
		}
		currentPackage.dependents[dependent] = struct{}{}
		w.store.Put(currentPackage)
	}
}

// for every package in 'packages', 'dependent' will no longer be a dependent
func (w *Worker) removeDependents(packages []string, dependent string) {
	for _, dep := range packages {
		pkg, ok := w.store.Get(dep)
		if !ok {
			log.Fatalf("missing package %s", pkg)
		}
		delete(pkg.dependents, dependent)
		w.store.Put(pkg)
	}
}

// search function to find a package in the package store
func (w *Worker) find(pkgs ...string) bool {
	if len(pkgs) == 0 {
		return false
	}
	if w.store.Size() == 0 {
		return false
	}
	for _, pkg := range pkgs {
		if _, ok := w.store.Get(pkg); !ok {
			return false
		}
	}
	return true
}
