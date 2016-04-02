package cyclon

import (
	"github.com/Gaboose/go-pubsub/pnet"
	"math/rand"
)

type PeerSet map[interface{}]pnet.Peer

func (s PeerSet) PopOldest() pnet.Peer {
	var oldest pnet.Peer
	for _, p := range s {
		if oldest == nil || oldest.Get(age).(int) < p.Get(age).(int) {
			oldest = p
		}
	}
	delete(s, oldest.Id())
	return oldest
}

// Return n random elements without removing them
func (s PeerSet) Sample(n int) []pnet.Peer {
	if n > len(s) {
		n = len(s)
	}
	p := rand.Perm(len(s))

	sl := make([]pnet.Peer, n)
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
