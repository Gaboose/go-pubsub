package gway

import (
	"bytes"
	"io"
	"testing"

	"github.com/Gaboose/go-pubsub/topo"
	ma "github.com/jbenet/go-multiaddr"
)

func TestRouter(t *testing.T) {
	m, err := ma.NewMultiaddr("/ip4/127.0.0.1/tcp/8001")
	if err != nil {
		t.Fatal(err)
	}

	gw1 := NewGateway()
	defer gw1.Close()
	err = gw1.ListenAll([][]byte{m.Bytes()})
	if err != nil {
		t.Fatal(err)
	}
	pn1foo := gw1.NewProtoNet("/foo")
	defer pn1foo.Close()
	pn1bar := gw1.NewProtoNet("/bar")
	defer pn1bar.Close()

	acceptAndWrite := func(ln topo.Listener, bts []byte) {
		s, err := ln.Accept()
		if err != nil {
			t.Fatal(err)
		}
		s.Write(bts)
		s.Close()
	}

	toWriteFoo := []byte("hello")
	go acceptAndWrite(pn1foo.Listen(), toWriteFoo)

	toWriteBar := []byte("world")
	go acceptAndWrite(pn1bar.Listen(), toWriteBar)

	gw2 := NewGateway()
	defer gw2.Close()
	pn2foo := gw2.NewProtoNet("/foo")
	defer pn2foo.Close()
	pn2bar := gw2.NewProtoNet("/bar")
	defer pn2bar.Close()

	dialAndRead := func(pn *ProtoNet, p *PeerInfo) []byte {
		s, err := pn.Dial(p)
		if err != nil {
			t.Fatal(err)
		}

		var buf bytes.Buffer
		io.Copy(&buf, s)
		return buf.Bytes()
	}

	dest := &PeerInfo{
		ID:     "bob",
		MAddrs: [][]byte{m.Bytes()},
	}

	toReadFoo := dialAndRead(pn2foo, dest)
	toReadBar := dialAndRead(pn2bar, dest)

	if !bytes.Equal(toWriteFoo, toReadFoo) {
		t.Fatalf("written %v, but read %v", toWriteFoo, toReadFoo)
	}

	if !bytes.Equal(toWriteBar, toReadBar) {
		t.Fatalf("written %v, but read %v", toWriteBar, toReadBar)
	}
}

func TestBadProto(t *testing.T) {
	m, err := ma.NewMultiaddr("/ip4/127.0.0.1/tcp/8002")
	if err != nil {
		t.Fatal(err)
	}

	gw1 := NewGateway()
	defer gw1.Close()
	err = gw1.ListenAll([][]byte{m.Bytes()})
	if err != nil {
		t.Fatal(err)
	}
	pn1 := gw1.NewProtoNet("/foo")
	defer pn1.Close()

	gw2 := NewGateway()
	defer gw2.Close()
	pn2 := gw2.NewProtoNet("/bar")
	defer pn2.Close()

	dest := &PeerInfo{
		ID:     "bob",
		MAddrs: [][]byte{m.Bytes()},
	}

	_, err = pn2.Dial(dest)

	if err == nil {
		t.Fatal("expected error from ProtoNet.Dial, got nil")
	}
}
