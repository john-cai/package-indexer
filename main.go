package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/john-cai/package-indexer/server"
)

const (
	ConnectionLimit        = "PACKAGE_INDEXER_CONNECTION_LIMIT"
	Port                   = "PACKAGE_INDEXER_PORT"
	ConnectionLimitDefault = 100
	PortDefault            = 8080
)

func main() {
	// parse environment variables
	connectionLimitString := os.Getenv(ConnectionLimit)
	portString := os.Getenv(Port)

	// set default values
	connectionLimit := ConnectionLimitDefault
	port := PortDefault

	if connectionLimitString != "" {
		i, err := strconv.Atoi(connectionLimitString)
		if err == nil {
			connectionLimit = i
		} else {
			fmt.Printf("%s not a valid value, using default %d", connectionLimitString, ConnectionLimitDefault)
		}
	}
	if portString != "" {
		i, err := strconv.Atoi(portString)
		if err == nil {
			port = i
		} else {
			fmt.Printf("%s not a valid value, using default %d", portString, PortDefault)
		}
	}

	p := server.NewPackageIndexer(connectionLimit, server.NewMapStore(), port)
	p.ListenAndServe()
}
