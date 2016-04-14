package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"strings"
	"time"
)

//ResponseCode is the code returned by the sever as a response to our requests
type ResponseCode string

const (
	//OK code
	OK = "OK"

	//FAIL code
	FAIL = "FAIL"

	//ERROR code
	ERROR = "ERROR"

	//UNKNOWN code
	UNKNOWN = "UNKNOWN"
)

//PackageIndexerClient sends messages to a running server.
type PackageIndexerClient interface {
	Name() string
	Close() error
	Send(msg string) (ResponseCode, error)
}

// TCPPackageIndexerClient connects to the running server via TCP
type TCPPackageIndexerClient struct {
	name string
	conn net.Conn
}

//Name return this client's name.
func (client *TCPPackageIndexerClient) Name() string {
	return client.name
}

//Close closes the connection to the server.
func (client *TCPPackageIndexerClient) Close() error {
	log.Printf("%s disconnecting", client.Name())
	return client.conn.Close()
}

//Send sends amessage to the server using its line-oriented protocol
func (client *TCPPackageIndexerClient) Send(msg string) (ResponseCode, error) {
	extendTimoutFor(client.conn)
	_, err := fmt.Fprintln(client.conn, msg)

	if err != nil {
		return UNKNOWN, fmt.Errorf("Error sending message to server: %v", err)
	}

	extendTimoutFor(client.conn)
	responseMsg, err := bufio.NewReader(client.conn).ReadString('\n')
	if err != nil {
		return UNKNOWN, fmt.Errorf("Error reading response code from server: %v", err)
	}

	returnedString := strings.TrimRight(responseMsg, "\n")

	if returnedString == OK {
		return OK, nil
	}

	if returnedString == FAIL {
		return FAIL, nil
	}

	if returnedString == ERROR {
		return ERROR, nil
	}

	return UNKNOWN, fmt.Errorf("Error parsing message from server [%s]: %v", responseMsg, err)
}

// MakeTCPPackageIndexClient returns a new instance of the client
func MakeTCPPackageIndexClient(name string, ip string, port int) (PackageIndexerClient, error) {
	host := fmt.Sprintf("%s:%d", ip, port)
	log.Printf("%s connecting to [%s]", name, host)
	conn, err := net.Dial("tcp", host)

	if err != nil {
		return nil, fmt.Errorf("Failed to open connection to [%s]: %#v", host, err)
	}

	return &TCPPackageIndexerClient{
		name: name,
		conn: conn,
	}, nil
}

func extendTimoutFor(conn net.Conn) {
	whenWillThisConnectionTimeout := time.Now().Add(time.Second * 10)
	conn.SetDeadline(whenWillThisConnectionTimeout)
}
