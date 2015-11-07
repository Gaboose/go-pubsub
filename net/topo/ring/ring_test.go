package ring

import (
	"testing"

	"github.com/Gaboose/go-pubsub/net/topo/mock"
)

func TestSingle(t *testing.T) {
	network := mock.NewNetwork()
	defer network.Shutdown()

	r := NewRing(network, mock.Peer{Id: 20}, 2)
	defer r.Shutdown()

	ch, done := network.NewMemberNotifier()
	defer done()

	network.SetPeer(mock.Peer{Id: 40}, mock.Status{Alive: true})

	<-ch
	network.AssertMembers(t, mock.Peer{Id: 40})
}

func TestMultiple(t *testing.T) {
	network := mock.NewNetwork()
	defer network.Shutdown()

	r := NewRing(network, mock.Peer{Id: 20}, 4)
	defer r.Shutdown()

	nCh, done := network.NewMemberNotifier()
	defer done()
	dCh, done := network.DeadMemberNotifier()
	defer done()

	network.SetPeer(mock.Peer{Id: 21}, mock.Status{Alive: true})
	network.SetPeer(mock.Peer{Id: 22}, mock.Status{Alive: true})
	network.SetPeer(mock.Peer{Id: 23}, mock.Status{Alive: true})
	network.SetPeer(mock.Peer{Id: 24}, mock.Status{Alive: true})
	network.SetPeer(mock.Peer{Id: 18}, mock.Status{Alive: true})

	for i := 0; i < 5; i++ {
		<-nCh
	}
	<-dCh

	network.AssertMembers(t,
		mock.Peer{Id: 18},
		mock.Peer{Id: 21},
		mock.Peer{Id: 22},
		mock.Peer{Id: 23})

	network.SetPeer(mock.Peer{Id: 19}, mock.Status{Alive: true})

	<-nCh
	<-dCh

	network.AssertMembers(t,
		mock.Peer{Id: 18},
		mock.Peer{Id: 19},
		mock.Peer{Id: 21},
		mock.Peer{Id: 22})
}

func TestWrap(t *testing.T) {
	network := mock.NewNetwork()
	defer network.Shutdown()

	r := NewRing(network, mock.Peer{Id: 20}, 4)
	defer r.Shutdown()

	ch, done := network.NewMemberNotifier()
	defer done()

	network.SetPeer(mock.Peer{Id: 19}, mock.Status{Alive: true})
	network.SetPeer(mock.Peer{Id: 21}, mock.Status{Alive: true})
	network.SetPeer(mock.Peer{Id: 22}, mock.Status{Alive: true})
	network.SetPeer(mock.Peer{Id: 98}, mock.Status{Alive: true})
	network.SetPeer(mock.Peer{Id: 99}, mock.Status{Alive: true})

	for i := 0; i < 5; i++ {
		<-ch
	}

	network.AssertMembers(t,
		mock.Peer{Id: 99},
		mock.Peer{Id: 19},
		mock.Peer{Id: 21},
		mock.Peer{Id: 22})
}
