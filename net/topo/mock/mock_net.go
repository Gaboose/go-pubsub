package mock

import (
	"reflect"
	"testing"

	"github.com/Gaboose/go-pubsub/net/topo"
	"github.com/briantigerchow/pubsub"
)

type Network struct {
	Peers         map[topo.Peer]Status
	Members       map[topo.Peer]bool
	Notifications *pubsub.PubSub
	shutdown      []func()
}

type Peer struct {
	Id int
}

const maxPeerId = 100

type Status struct {
	Alive bool
}

func NewNetwork() *Network {
	n := &Network{}
	n.Peers = make(map[topo.Peer]Status)
	n.Members = make(map[topo.Peer]bool)
	n.Notifications = pubsub.New(100)
	n.shutdown = append(n.shutdown, n.Notifications.Shutdown)
	return n
}

func (n *Network) SetPeer(p Peer, s Status) {
	n.Peers[p] = s
	if s.Alive {
		n.Notifications.Pub(p, "alivePeer")
	}
}

func (n *Network) AlivePeerIterator() <-chan topo.Peer {
	ch := make(chan topo.Peer, len(n.Peers))
	for p := range n.Peers {
		ch <- p
	}
	close(ch)
	return ch
}

func (n *Network) AlivePeerNotifier() (<-chan topo.Peer, func()) {
	return n.newNotifier("alivePeer")
}

func (n *Network) DeadMemberNotifier() (<-chan topo.Peer, func()) {
	return n.newNotifier("deadMember")
}

func (n *Network) NewMemberNotifier() (<-chan topo.Peer, func()) {
	return n.newNotifier("newMember")
}

func (n *Network) ConnectMember(p topo.Peer) {
	n.Members[p] = true
	n.Notifications.Pub(p, "newMember")
}

func (n *Network) DisconnectMember(p topo.Peer) {
	delete(n.Members, p)
	n.Notifications.Pub(p, "deadMember")
}

func (n *Network) AssertMembers(t *testing.T, peers ...Peer) {
	exp := map[int]bool{}
	for _, p := range peers {
		exp[p.Id] = true
	}
	got := map[int]bool{}
	for p := range n.Members {
		got[p.(Peer).Id] = true
	}
	if !reflect.DeepEqual(exp, got) {
		t.Errorf("Expected %v, got %v", exp, got)
	}
}

func (n *Network) Shutdown() {
	for _, f := range n.shutdown {
		f()
	}
}

func (n *Network) newNotifier(topic string) (chan topo.Peer, func()) {
	ch := n.Notifications.Sub(topic)
	peerCh := castToPeerCh(ch)
	done := func() { close(peerCh); n.Notifications.Unsub(ch) }
	return peerCh, done
}

func (p Peer) ModCompare(c topo.ModComparable) int {
	halfmod := maxPeerId / 2
	p2 := c.(Peer)
	dist := (p.Id-p2.Id+maxPeerId+halfmod)%maxPeerId - halfmod
	if dist > 0 {
		return 1
	} else if dist == 0 {
		return 0
	} else {
		return -1
	}
}

func castToPeerCh(ch chan interface{}) chan topo.Peer {
	peerCh := make(chan topo.Peer)
	go func() {
		for x := range ch {
			peerCh <- x.(topo.Peer)
		}
	}()
	return peerCh
}
