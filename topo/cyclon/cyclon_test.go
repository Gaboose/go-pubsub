package cyclon

import (
	"encoding/gob"
	"fmt"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/Gaboose/go-pubsub/topo"
	"github.com/Gaboose/go-pubsub/topo/mock"
)

func TestNumGoroutine(t *testing.T) {
	baseNum := numGoroutine()
	var num int

	sw := mock.ProtoNetSwarm{}
	p := &mock.Peer{ID: "peer0"}
	c := New(p, 20, 10, sw.DialListener(p))
	c.Start(time.Second)
	c.Stop()

	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	timeout := time.After(time.Second)
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

func TestNoAnswer(t *testing.T) {
	sw := mock.ProtoNetSwarm{}
	p := &mock.Peer{ID: "peer0"}
	c := New(p, 20, 10, sw.DialListener(p))

	gob.Register(&mock.Peer{})
	c.Add(&mock.Peer{ID: "peer1"})
	c.Shuffle()

	if len(c.neighbs) > 0 {
		t.Fatalf("expected no neighbours, got %v", c.neighbs)
	}
}

func TestAgeSelect(t *testing.T) {
	sw := mock.ProtoNetSwarm{}
	p := &mock.Peer{ID: "peer0"}
	c := New(p, 20, 10, sw.DialListener(p))

	c.neighbs = PeerSet{
		"peer1": &mock.Peer{"peer1", map[string]interface{}{age: 2}},
		"peer2": &mock.Peer{"peer2", map[string]interface{}{age: 4}},
		"peer3": &mock.Peer{"peer3", map[string]interface{}{age: 3}},
	}

	c.Shuffle()

	expected := []interface{}{"peer1", "peer3"}
	got := make([]interface{}, 0, 3)
	for p, _ := range c.neighbs {
		got = append(got, p)
	}
	if !equalSets(expected, got) {
		t.Fatalf("expected %v, got %v", expected, got)
	}
}

func TestReverseEdge(t *testing.T) {
	sw := mock.ProtoNetSwarm{}
	p0 := &mock.Peer{ID: "peer0"}
	p1 := &mock.Peer{ID: "peer1"}
	c0 := New(p0, 20, 10, sw.DialListener(p0.Id()))
	c1 := New(p1, 20, 10, sw.DialListener(p1.Id()))

	c0.Add(p1)

	c1.Start(0)
	defer c1.Stop()
	c0.Shuffle()

	select {
	case p := <-c1.Out():
		if p.Id() != p0.Id() {
			t.Fatalf("expected %v, got %v", p0, p)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out")
	}
}

func TestConservation(t *testing.T) {
	sw := mock.ProtoNetSwarm{}
	p0 := &mock.Peer{ID: "p0"}
	p1 := &mock.Peer{ID: "p1"}
	c0 := New(p0, 5, 3, sw.DialListener(p0.Id()))
	c1 := New(p1, 5, 3, sw.DialListener(p1.Id()))

	c0.neighbs = PeerSet{
		"p2": &mock.Peer{"p2", map[string]interface{}{age: 2}},
		"p3": &mock.Peer{"p3", map[string]interface{}{age: 2}},
		"p4": &mock.Peer{"p4", map[string]interface{}{age: 2}},
		"p5": &mock.Peer{"p5", map[string]interface{}{age: 2}},
		"p6": &mock.Peer{"p6", map[string]interface{}{age: 2}},
	}
	c1.neighbs = PeerSet{
		"p0":  &mock.Peer{"p0", map[string]interface{}{age: 3}},
		"p7":  &mock.Peer{"p7", map[string]interface{}{age: 2}},
		"p8":  &mock.Peer{"p8", map[string]interface{}{age: 2}},
		"p9":  &mock.Peer{"p9", map[string]interface{}{age: 2}},
		"p10": &mock.Peer{"p10", map[string]interface{}{age: 2}},
	}

	c0.Start(0)
	defer c0.Stop()
	c1.Shuffle()

	// We expect p0 to be selected for shuffling, thus removed from the pool.
	// p1 should be added to p0's cache, because p1 is initiating the shuffle.
	// All other peer profiles should be conserved.
	expected := []interface{}{"p1", "p2", "p3", "p4",
		"p5", "p6", "p7", "p8", "p9", "p10"}

	got := make([]interface{}, 0, 10)
	for p, _ := range c0.neighbs {
		got = append(got, p)
	}
	for p, _ := range c1.neighbs {
		got = append(got, p)
	}
	if !equalSets(expected, got) {
		t.Fatalf("expected %v, got %v", expected, got)
	}
}

func TestIncreaseAge(t *testing.T) {
	sw := mock.ProtoNetSwarm{}
	p0 := &mock.Peer{ID: "p0"}
	p1 := &mock.Peer{ID: "p1"}
	c0 := New(p0, 3, 2, sw.DialListener(p0.Id()))
	c1 := New(p1, 3, 2, sw.DialListener(p0.Id()))

	c0.neighbs = PeerSet{
		"p2": &mock.Peer{"p2", map[string]interface{}{age: 2}},
		"p3": &mock.Peer{"p3", map[string]interface{}{age: 2}},
		"p4": &mock.Peer{"p4", map[string]interface{}{age: 2}},
	}
	c1.neighbs = PeerSet{
		"p0": &mock.Peer{"p0", map[string]interface{}{age: 3}},
		"p5": &mock.Peer{"p5", map[string]interface{}{age: 2}},
		"p6": &mock.Peer{"p6", map[string]interface{}{age: 2}},
	}

	c0.Start(0)
	defer c0.Stop()
	c1.Shuffle()

	union := make(map[interface{}]topo.Peer)
	for s, p := range c0.neighbs {
		union[s] = p
	}
	for s, p := range c1.neighbs {
		union[s] = p
	}
	expected := map[string]int{"p1": 0, "p2": 2, "p3": 2, "p4": 2, "p5": 3, "p6": 3}
	for s, a := range expected {
		if union[s].Get(age) != a {
			t.Fatalf("expected %s.age %d, got %d", s, a, union[s].Get(age))
		}
	}
}

func TestBday(t *testing.T) {
	sw := mock.ProtoNetSwarm{}
	c0 := New(&mock.Peer{ID: "p0"}, 3, 2, sw.DialListener("p0"))
	c1 := New(&mock.Peer{ID: "p1"}, 3, 2, sw.DialListener("p1"))

	c0.Start(0)
	defer c0.Stop()
	c1.Start(0)
	defer c1.Stop()
	c0.neighbs = PeerSet{
		"p1": &mock.Peer{"p1", map[string]interface{}{age: 10}},
		"p2": &mock.Peer{"p2", map[string]interface{}{age: 0}},
	}
	c1.neighbs = PeerSet{
		"p0": &mock.Peer{"p0", map[string]interface{}{age: 10}},
	}
	c0.Shuffle()
	c1.Shuffle()

	c0.neighbs = PeerSet{
		"p1": &mock.Peer{"p1", map[string]interface{}{age: 10}},
		"p3": &mock.Peer{"p3", map[string]interface{}{age: 0}},
	}
	c0.Shuffle()

	set := make(map[interface{}]int64)
	for i := 0; i < 3; i++ {
		p := <-c1.Out()
		set[p.Id()] = p.Get(bday).(int64)
	}

	diff := set["p3"] - set["p2"]
	if diff != 1 {
		t.Fatalf("expected birth to be 1 apart, got %v", diff)
	}
}

func equalSets(arr1, arr2 []interface{}) bool {
	set1 := map[interface{}]bool{}
	for _, s := range arr1 {
		set1[s] = true
	}
	set2 := map[interface{}]bool{}
	for _, s := range arr2 {
		set2[s] = true
	}
	if len(set1) != len(set2) {
		return false
	}
	for s, _ := range set1 {
		if _, has := set2[s]; !has {
			return false
		}
	}
	return true
}

func numGoroutine() int {
	buf := make([]byte, 1<<16)
	runtime.Stack(buf, true)
	s := fmt.Sprintf("%s", buf)
	return strings.Count(s, "created by github.com")
}
