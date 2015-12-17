package cyclon

import (
	"github.com/Gaboose/go-pubsub/topo"
	"math/rand"
)

type PeerSet map[interface{}]topo.Peer

func (s PeerSet) PopOldest() topo.Peer {
	var oldest topo.Peer
	for _, p := range s {
		if oldest == nil || oldest.Get(age).(int) < p.Get(age).(int) {
			oldest = p
		}
	}
	delete(s, oldest.Id())
	return oldest
}

// Return n random elements without removing them
func (s PeerSet) Sample(n int) []topo.Peer {
	if n > len(s) {
		n = len(s)
	}
	p := rand.Perm(len(s))

	sl := make([]topo.Peer, n)
	i, j := 0, 0
	for _, v := range s {
		if p[i] < n {
			sl[j] = v
			j++
		}
		i++
	}

	return sl
}
