package broadcast

import (
	"fmt"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Gaboose/go-pubsub/topo"
	"github.com/Gaboose/go-pubsub/topo/mock"
)

const timeToWait = time.Millisecond

func TestNumGoroutine(t *testing.T) {
	baseNum := numGoroutine()
	var num int

	sw := mock.ProtNetSwarm{}
	b := New(2, time.Minute, sw.DialListener("p0"))
	b.Start(nil, 0)
	b.Stop()

	ticker := time.NewTicker(timeToWait / 100)
	defer ticker.Stop()
	timeout := time.After(timeToWait)
	for {
		select {
		case <-ticker.C:
			num = numGoroutine()
			if num == baseNum {
				return
			}
		case <-timeout:
			t.Fatalf("%d goroutines are leaking", num-baseNum)
		}
	}
}

func TestMsg(t *testing.T) {
	sw := mock.ProtNetSwarm{}
	b0 := New(2, time.Minute, sw.DialListener("p0"))
	b1 := New(2, time.Minute, sw.DialListener("p1"))

	ps0, ps1 := make(chan topo.Peer), make(chan topo.Peer)
	b0.Start(ps0, 0)
	b1.Start(ps1, 0)
	ps0 <- &mock.Peer{ID: "p1"}
	b0.In() <- "hello world"

	select {
	case msg := <-b1.Out():
		if msg != "hello world" {
			t.Fatalf("expected \"hello world\", got %s", msg)
		}
	case <-time.After(timeToWait):
		t.Fatal("timeout")
	}
}

func TestMsgForward(t *testing.T) {
	sw := mock.ProtNetSwarm{}

	numNodes := 4
	b, ch := make([]*Broadcast, numNodes), make([]chan topo.Peer, numNodes)

	for i, _ := range b {
		name := fmt.Sprintf("p%d", i)
		b[i] = New(1, time.Minute, sw.DialListener(name))
		b[i].Str = name
		ch[i] = make(chan topo.Peer)
		b[i].Start(ch[i], 2)
		defer b[i].Stop()
	}

	ch[0] <- &mock.Peer{ID: "p1"}
	ch[1] <- &mock.Peer{ID: "p2"}
	ch[2] <- &mock.Peer{ID: "p0"}
	ch[3] <- &mock.Peer{ID: "p2"}

	b[0].In() <- "hello world"

	testReceive(t,
		map[*Broadcast]int{b[0]: 0, b[1]: 1, b[2]: 1, b[3]: 1},
		timeToWait,
	)
}

func TestNeighbourDiscard(t *testing.T) {
	sw := mock.ProtNetSwarm{}

	numNodes := 4
	b, ch := make([]*Broadcast, numNodes), make([]chan topo.Peer, numNodes)

	for i, _ := range b {
		name := fmt.Sprintf("p%d", i)
		b[i] = New(1, time.Minute, sw.DialListener(name))
		b[i].Str = name
		ch[i] = make(chan topo.Peer)
		b[i].Start(ch[i], 2)
		defer b[i].Stop()
	}

	ch[0] <- &mock.Peer{"p2", map[string]interface{}{bday: 2}} // keep
	ch[0] <- &mock.Peer{"p1", map[string]interface{}{bday: 0}} // discard
	ch[0] <- &mock.Peer{"p3", map[string]interface{}{bday: 1}} // keep

	b[0].In() <- "hello world"

	testReceive(t,
		map[*Broadcast]int{b[0]: 0, b[1]: 0, b[2]: 1, b[3]: 1},
		timeToWait,
	)
}

func TestNeighbourBackup(t *testing.T) {
	sw := mock.ProtNetSwarm{}

	numNodes := 4
	b, ch := make([]*Broadcast, numNodes), make([]chan topo.Peer, numNodes)

	for i, _ := range b {
		name := fmt.Sprintf("p%d", i)
		b[i] = New(1, time.Minute, sw.DialListener(name))
		b[i].Str = name
		ch[i] = make(chan topo.Peer)
		b[i].Start(ch[i], 2)
		defer b[i].Stop()
	}

	ch[0] <- &mock.Peer{"p3", map[string]interface{}{bday: 0}} // backup
	ch[0] <- &mock.Peer{"p1", map[string]interface{}{bday: 2}} // fail
	ch[0] <- &mock.Peer{"p2", map[string]interface{}{bday: 1}} // keep

	b[1].Stop()

	time.Sleep(timeToWait)

	b[0].In() <- "hello world"

	testReceive(t,
		map[*Broadcast]int{b[0]: 0, b[1]: 0, b[2]: 1, b[3]: 1},
		timeToWait,
	)
}

func numGoroutine() int {
	buf := make([]byte, 1<<16)
	runtime.Stack(buf, true)
	s := fmt.Sprintf("%s", buf)
	return strings.Count(s, "created by github.com")
}

func testReceive(t *testing.T, expect map[*Broadcast]int, dur time.Duration) {
	bs := make([]*Broadcast, 0, len(expect))
	for b, _ := range expect {
		bs = append(bs, b)
	}
	result := receive(bs, dur)

	for b, i := range expect {
		if len(result[b]) != i {
			t.Fatalf("expected to receive %v, got %v", expect, result)
		}
	}
}

func receive(bs []*Broadcast, dur time.Duration) map[*Broadcast][]string {
	mutex := sync.Mutex{}
	msgs := map[*Broadcast][]string{}
	wg := sync.WaitGroup{}

	wg.Add(len(bs))

	listener := func(b *Broadcast) {
		timeout := time.After(dur)
		select {
		case m := <-b.Out():
			mutex.Lock()
			msgs[b] = append(msgs[b], m)
			mutex.Unlock()
		case <-timeout:
		}
		wg.Done()
	}

	for _, b := range bs {
		go listener(b)
	}

	wg.Wait()
	return msgs
}
