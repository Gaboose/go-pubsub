package ping

import (
	"testing"
	"time"

	"github.com/Gaboose/go-pubsub/topo/mock"
)

func TestSucceed(t *testing.T) {
	sw := mock.ProtoNetSwarm{}

	ping0 := &Ping{protonet: sw.DialListener("peer0")} // pinger
	ping1 := &Ping{protonet: sw.DialListener("peer1")} // ponger
	ping1.Serve()
	defer ping1.Stop()

	err := ping0.Ping(&mock.Peer{ID: "peer1"}, nil)
	if err != nil {
		t.Fatalf("expected no error, got '%v'", err)
	}
}

func TestTimeout(t *testing.T) {
	sw := mock.ProtoNetSwarm{}

	// ponger
	pn := sw.DialListener("peer0")
	ln := pn.Listen()
	defer ln.Close()

	// pinger
	ping := &Ping{protonet: sw.DialListener("peer1")}
	done := make(chan error)
	stop := make(chan bool)
	go func() {
		done <- ping.Ping(&mock.Peer{ID: "peer0"}, stop)
	}()
	
	// accept, but don't respond
	c, err := ln.Accept()
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	close(stop)

	err = <-done
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
}

func TestDelayedResponse(t *testing.T) {
	sw := mock.ProtoNetSwarm{}

	// ponger
	pn := sw.DialListener("peer0")
	ln := pn.Listen()
	defer ln.Close()

	// pinger
	ping := &Ping{protonet: sw.DialListener("peer1")}
	done := make(chan error)
	go func() {
		done <- ping.Ping(&mock.Peer{ID: "peer0"}, nil)
	}()

	// accept and respond after a delay
	c, err := ln.Accept()
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	time.Sleep(100*time.Millisecond)
	c.Write([]byte(msg))

	err = <-done
	if err != nil {
		t.Fatalf("expected no error, got '%v'", err)
	}
}

func TestFail(t *testing.T) {
	sw := mock.ProtoNetSwarm{}
	ping0 := &Ping{protonet: sw.DialListener("peer0")}

	err := ping0.Ping(&mock.Peer{ID: "peer1"}, nil)
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
}
