package manet

import (
	"fmt"
	"golang.org/x/net/websocket"
	"io"
	"net"
	"net/http"
	"runtime"
	"strings"
	"testing"
	"time"

	ma "github.com/jbenet/go-multiaddr"
)

// time to wait after ln.Close()
const toSleep = time.Millisecond

func TestDial(t *testing.T) {
	ms := []ma.Multiaddr{
		newMultiaddr(t, "/ip4/127.0.0.1/tcp/4324"),
		newMultiaddr(t, "/dns/localhost/tcp/4324"),
		newMultiaddr(t, "/ip6/::1/tcp/4325"),
		newMultiaddr(t, "/dns/localhost/tcp/4325"),
		newMultiaddr(t, "/ip4/127.0.0.1/tcp/4326/ws/foo"),
		newMultiaddr(t, "/ip4/127.0.0.1/tcp/4326/http/ws/bar"),
	}

	stop := make(chan struct{})
	defer close(stop)

	err := netecho("tcp", "127.0.0.1:4324", stop)
	if err != nil {
		t.Fatal(err)
	}

	err = netecho("tcp", "[::1]:4325", stop)
	if err != nil {
		t.Fatal(err)
	}

	err = wsecho("tcp", "127.0.0.1:4326", stop)
	if err != nil {
		t.Fatal(err)
	}

	for _, m := range ms {
		c, err := Dial(m)
		if err != nil {
			t.Errorf("Dial(%s) err: %s", m.String(), err)
			continue
		}
		assertEcho(t, c, m)
	}
}

func TestListen(t *testing.T) {
	time.Sleep(toSleep)

	lms := []ma.Multiaddr{
		newMultiaddr(t, "/ip4/0.0.0.0/tcp/4324"),
		newMultiaddr(t, "/ip6/::1/tcp/4325"),
		newMultiaddr(t, "/ip4/0.0.0.0/tcp/4326/http/ws/foo"),
		newMultiaddr(t, "/ip4/0.0.0.0/tcp/4326/http/ws/bar"), //reusing http server on the same port
		newMultiaddr(t, "/ip4/0.0.0.0/tcp/4326/ws/qux"),      //http part is optional
		newMultiaddr(t, "/dns/localhost/tcp/4327"),
	}

	dms := []ma.Multiaddr{
		newMultiaddr(t, "/ip4/127.0.0.1/tcp/4324"),
		newMultiaddr(t, "/ip6/::1/tcp/4325"),
		newMultiaddr(t, "/ip4/127.0.0.1/tcp/4326/ws/bar"),
		newMultiaddr(t, "/ip4/127.0.0.1/tcp/4326/ws/qux"),
		newMultiaddr(t, "/ip4/127.0.0.1/tcp/4326/ws/foo"),
		newMultiaddr(t, "/dns/localhost/tcp/4327"),
	}

	for _, m := range lms {
		ln, err := Listen(m)
		if err != nil {
			t.Errorf("Listen(%s) err: %s", m, err)
			continue
		}
		defer ln.Close()
		go serveecho(ln)
	}

	if t.Failed() {
		return
	}

	for _, m := range dms {
		c, err := Dial(m)
		if err != nil {
			t.Errorf("Dial(%s) err: %s", m, err)
			continue
		}
		assertEcho(t, c, m)
	}
}

func TestListenDuplicate(t *testing.T) {
	time.Sleep(toSleep)

	lms := []ma.Multiaddr{
		newMultiaddr(t, "/ip4/0.0.0.0/tcp/4324"),
		newMultiaddr(t, "/ip4/0.0.0.0/tcp/4325/http/ws/foo"),
		newMultiaddr(t, "/dns/localhost/tcp/4326"),
	}

	for _, m := range lms {
		ln, err := Listen(m)
		if err != nil {
			t.Errorf("Listen(%s) err: %s", m, err)
			continue
		}
		defer ln.Close()
	}

	if t.Failed() {
		return
	}

	for _, m := range lms {
		ln, err := Listen(m)
		if err == nil {
			t.Errorf("Listen(%s) expected an error", m)
			ln.Close()
		}
	}

	stop := make(chan struct{})
	defer close(stop)

	err := netecho("tcp", "127.0.0.1:4324", stop)
	if err == nil {
		t.Errorf("expected an error")
	}

	err = wsecho("tcp", "127.0.0.1:4325", stop)
	if err == nil {
		t.Errorf("expected an error")
	}
}

func TestListenFree(t *testing.T) {
	time.Sleep(toSleep)

	ms := []ma.Multiaddr{
		newMultiaddr(t, "/ip4/0.0.0.0/tcp/4324"),
		newMultiaddr(t, "/ip4/0.0.0.0/tcp/4325/http/ws/foo"),
		newMultiaddr(t, "/ip4/0.0.0.0/tcp/4325/http/ws/bar"),
		newMultiaddr(t, "/dns/localhost/tcp/4326"),
	}

	lns := []Listener{}

	for _, m := range ms {
		ln, err := Listen(m)
		if err != nil {
			t.Errorf("first Listen(%s) err: %s", m, err)
			continue
		}
		defer ln.Close()
		lns = append(lns, ln)
	}

	if t.Failed() {
		return
	}

	// see if we can close and open the listeners again
	for i, ln := range lns {
		ln.Close()

		time.Sleep(toSleep)

		l, err := Listen(ms[i])
		if err != nil {
			t.Errorf("second Listen(%s) err: %s", ms[i], err)
			continue
		}
		lns[i] = l
	}

	for _, ln := range lns {
		ln.Close()
	}

	time.Sleep(toSleep)

	stop := make(chan struct{})
	defer close(stop)

	// see if http listener freed everything up to the tcp socket
	err := netecho("tcp", "127.0.0.1:4325", stop)
	if err != nil {
		t.Fatal(err)
	}
}

func TestNumGoroutines(t *testing.T) {
	time.Sleep(toSleep)

	ms := []ma.Multiaddr{
		newMultiaddr(t, "/ip4/0.0.0.0/tcp/4324"),
		newMultiaddr(t, "/ip4/0.0.0.0/tcp/4325/http/ws/foo"),
		newMultiaddr(t, "/ip4/0.0.0.0/tcp/4325/http/ws/bar"),
		newMultiaddr(t, "/ip4/0.0.0.0/tcp/4325/ws/qux"),
		newMultiaddr(t, "/dns/localhost/tcp/4326"),
	}

	lns := []Listener{}

	baseNum := numGoroutines()

	// open listeners
	for _, m := range ms {
		ln, err := Listen(m)
		if err != nil {
			t.Errorf("Listen(%s) err: %s", m, err)
		}
		defer ln.Close()
		lns = append(lns, ln)
	}

	// open connections
	for _, m := range ms {
		_, err := Dial(m)
		if err != nil {
			t.Errorf("Dial(%s) err: %s", m, err)
		}
	}

	// close listeners
	for _, ln := range lns {
		ln.Close()
	}

	time.Sleep(toSleep)

	assertNumGoroutines(t, baseNum)
}

func newMultiaddr(t *testing.T, m string) ma.Multiaddr {
	maddr, err := ma.NewMultiaddr(m)
	if err != nil {
		t.Fatal("failed to construct multiaddr:", m, err)
	}
	return maddr
}

func serveecho(ln interface{}) error {
	var c net.Conn
	var err error
	for {
		switch l := ln.(type) {
		case net.Listener:
			c, err = l.Accept()
		case Listener:
			c, err = l.Accept()
		default:
			panic("bad listener argument type")
		}
		if err != nil {
			return err
		}
		echoOnce(c)
		c.Close()
	}
}

func netecho(nt, laddr string, stop <-chan struct{}) error {
	ln, err := net.Listen(nt, laddr)
	if err != nil {
		return err
	}

	go serveecho(ln)

	go func() {
		<-stop
		ln.Close()
	}()

	return nil
}

func wsecho(nt, laddr string, stop <-chan struct{}) error {
	ln, err := net.Listen(nt, laddr)
	if err != nil {
		return err
	}

	go http.Serve(ln, websocket.Handler(func(ws *websocket.Conn) {
		echoOnce(ws)
	}))

	go func() {
		<-stop
		ln.Close()
	}()

	return nil
}

func echoOnce(rw io.ReadWriter) error {
	buf := make([]byte, 256)
	n, err := rw.Read(buf)
	if err != nil {
		return err
	}
	_, err = rw.Write(buf[:n])
	return err
}

func assertEcho(t *testing.T, rwc io.ReadWriteCloser, m ma.Multiaddr) {
	str := "test string"
	done := make(chan struct{})
	defer rwc.Close()

	go func() {
		defer close(done)

		_, err := fmt.Fprint(rwc, str)
		if err != nil {
			t.Error(err)
			return
		}

		buf := make([]byte, 256)
		n, err := rwc.Read(buf)
		if err != nil {
			t.Error(err)
			return
		}
		got := string(buf[:n])

		if got != str {
			t.Errorf("expected \"%s\", got \"%s\"", str, got)
		}
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Errorf("assertEcho: %s timed out", m.String())
	}
}

func numGoroutines() int {
	buf := make([]byte, 1<<16)
	runtime.Stack(buf, true)
	s := fmt.Sprintf("%s", buf)
	return strings.Count(s, "created by")
}

func assertNumGoroutines(t *testing.T, baseNum int) {
	var num int
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	timeout := time.After(time.Second)
	for {
		select {
		case <-ticker.C:
			num = numGoroutines()
			if num == baseNum {
				return
			} else if num < baseNum {
				t.Fatalf("probably invalid base number of goroutines: started with %d, but now there's %d", baseNum, num)
			}
		case <-timeout:
			t.Fatalf("%d goroutines are leaking", num-baseNum)
		}
	}
}
