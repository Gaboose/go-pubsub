package broadcast

import (
	"fmt"
	"testing"
	//	"time"

	"github.com/Gaboose/go-pubsub-planet/topo"
	"github.com/Gaboose/go-pubsub-planet/topo/mock"
)

//TODO: broadcaster replenish neighbours on faulty connections
//TODO: backwards connections

func TestSmth(t *testing.T) {
	sw := mock.ProtNetSwarm{}
	b0 := New("p0", 2, sw.DialListener("p0"))
	b1 := New("p1", 2, sw.DialListener("p1"))

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
	case <-time.After(time.Second):
		t.Fatal("timeout")
	}
}

//func TestSmth2(t *testing.T) {
//	sw := mock.ProtNetSwarm{}
//	b0 := New("p0", 2, sw.DialListener("p0"))
//	b1 := New("p1", 2, sw.DialListener("p1"))
//	b2 := New("p2", 2, sw.DialListener("p2"))
//
//	fmt.Println("about to start")
//
//	b0.Start(nil)
//	b1.Start(nil)
//	b2.Start(nil)
//
//	fmt.Println("about to add neighbours")
//
//	b0.AddNeighbours([]topo.Peer{
//		&mock.Peer{ID: "p1"},
//	})
//	b1.AddNeighbours([]topo.Peer{
//		&mock.Peer{ID: "p0"},
//		&mock.Peer{ID: "p2"},
//	})
//	b2.AddNeighbours([]topo.Peer{
//		&mock.Peer{ID: "p0"},
//		&mock.Peer{ID: "p1"},
//	})
//
//	fmt.Println("about to greet world")
//	fmt.Println(b0.neighbs)
//
//	b0.In() <- "hello world"
//	fmt.Println(<-b1.Out())
//	fmt.Println(<-b2.Out())
//}
