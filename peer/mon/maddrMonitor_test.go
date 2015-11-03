package addrMon

import (
	"reflect"
	"testing"

	ma "github.com/jbenet/go-multiaddr"
	manet "github.com/jbenet/go-multiaddr-net"
)

func TestEmpty(t *testing.T) {
	// start an address monitor instance
	_, rqCh, _ := Start()

	// request an iterator
	itCh := make(chan IterResponse)
	rqCh <- IterRequest{ItFunc: AliveIterator, Ch: itCh}
	it := <-itCh
	defer close(it.Done)

	// expect to see nothing
	got := []Resource{}
	for m := range it.Out {
		got = append(got, m)
	}
	exp := []Resource{}
	if !reflect.DeepEqual(got, exp) {
		t.Errorf("Expected %d, got %d", exp, got)
	}
}

func TestDead(t *testing.T) {
	// start an address monitor instance
	pnCh, rqCh, upCh := Start()

	// send a multiaddr to the pending channel
	m, err := ma.NewMultiaddr("/ip4/127.0.0.1/tcp/8085")
	if err != nil {
		panic(err)
	}
	pnCh <- m

	// wait on the update channel, until the monitor checks the address
	<-upCh

	// get an iterator
	itCh := make(chan IterResponse)
	rqCh <- IterRequest{ItFunc: AliveIterator, Ch: itCh}
	it := <-itCh
	defer close(it.Done)

	// expect to see nothing
	got := []Resource{}
	for m := range it.Out {
		got = append(got, m)
	}
	exp := []Resource{}
	if !reflect.DeepEqual(got, exp) {
		t.Errorf("Expected %d, got %d", exp, got)
	}
}

func TestAlive(t *testing.T) {
	// set up a tcp listener
	m, err := ma.NewMultiaddr("/ip4/127.0.0.1/tcp/8085")
	if err != nil {
		panic(err)
	}
	l, err := manet.Listen(m)
	if err != nil {
		panic(err)
	}
	defer l.Close()

	// start an address monitor instance
	pnCh, rqCh, upCh := Start()

	// send the multiaddr to the pending channel
	pnCh <- m

	// wait on the update channel, until the monitor checks the address
	<-upCh

	// get an iterator
	itCh := make(chan IterResponse)
	rqCh <- IterRequest{ItFunc: AliveIterator, Ch: itCh}
	it := <-itCh
	defer close(it.Done)

	// expect to see the address
	got := []Resource{}
	for m := range it.Out {
		got = append(got, m)
	}
	exp := []Resource{Resource(m)}
	if !reflect.DeepEqual(got, exp) {
		t.Errorf("Expected %d, got %d", exp, got)
	}
}
