package main

import (
	"fmt"
	"net"
	"testing"
)

func respondWith(t *testing.T, server net.Listener, responseCode string) {
	for {
		conn, err := server.Accept()
		if err != nil {
			t.Fatalf("Error reading socket: %v", err)
		}
		fmt.Fprintln(conn, responseCode)
	}
}

func TestMakeTCPPackageIndexClient(t *testing.T) {
	portWithNobodyListeningTo := 8089
	client, err := MakeTCPPackageIndexClient("portisntopen", "127.0.0.1", portWithNobodyListeningTo)

	if err == nil {
		t.Errorf("Expected connection to [%d] to raise error as there's no server, got %v", portWithNobodyListeningTo, client)
	}
}

func TestSend(t *testing.T) {
	goodPort := 8088
	goodServer, err := net.Listen("tcp", fmt.Sprintf(":%d", goodPort))
	defer goodServer.Close()

	if err != nil {
		t.Fatalf("Error opening test server: %v", err)
	}

	go respondWith(t, goodServer, "OK")

	client, err := MakeTCPPackageIndexClient("goodPort", "127.0.0.1",goodPort)
	if err != nil {
		t.Fatalf("Error connecting to server: %v", err)
	}

	responseCode, err := client.Send("A")

	if err != nil {
		t.Errorf("Error sending message to server: %v", err)
	}

	if responseCode == FAIL {
		t.Errorf("Expected responseCode to be 1, got %v", responseCode)
	}

	badPort := 8090
	badServer, err := net.Listen("tcp", fmt.Sprintf(":%d", badPort))
	defer badServer.Close()

	if err != nil {
		t.Fatalf("Error opening test server: %v", err)
	}

	go respondWith(t, badServer, "banana")

	client, err = MakeTCPPackageIndexClient("badPort", "127.0.0.1", badPort)
	if err != nil {
		t.Fatalf("Error connecting to server: %v", err)
	}

	responseCode, err = client.Send("B")

	if err == nil {
		t.Errorf("No error returned for bad responseCode from server: %#v", responseCode)
	}
}
