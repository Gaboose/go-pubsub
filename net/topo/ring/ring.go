package ring

import "github.com/Gaboose/go-pubsub/net/topo"

type Ring struct {
	network  topo.Network
	me       topo.Peer
	membersN int
	members  []topo.Peer
	shutdown []func()
}

func NewRing(network topo.Network, me topo.Peer, membersN int) *Ring {
	r := new(Ring)
	r.network = network
	r.me = me
	r.membersN = membersN

	// subscribe to newly alive peers
	ch, done := network.AlivePeerNotifier()
	r.shutdown = append(r.shutdown, done)
	go func(ch <-chan topo.Peer) {
		for p := range ch {
			r.considerPeer(p)
		}
	}(ch)

	// consider currently alive peers
	ch = network.AlivePeerIterator()
	for p := range ch {
		r.considerPeer(p)
	}

	return r
}

func (r *Ring) considerPeer(p topo.Peer) {

	// find where to insert p
	i := r.potentialIndex(p)
	if i == 0 && p.ModCompare(r.me) > 0 {
		i = len(r.members)
	}

	// make room
	r.members = append(r.members, p)

	// shift the right side by one and insert (i, p)
	copy(r.members[i+1:], r.members[i:len(r.members)-1])
	r.members[i] = p

	// keep the slice of members from overflowing
	j := -1
	var dp topo.Peer
	if len(r.members) > r.membersN {
		// remove the furthest peer index-wise
		if r.potentialIndex(r.me) > r.membersN/2 {
			j, dp = 0, r.members[0]
			r.members = r.members[1:]
		} else {
			j, dp = r.membersN, r.members[r.membersN]
			r.members = r.members[:r.membersN]
		}
	}

	// update the network
	if i != j {
		r.network.ConnectMember(p)
		if dp != nil {
			r.network.DisconnectMember(dp)
		}
	}
}

// Find where p would go in the sorted r.members
func (r *Ring) potentialIndex(p topo.Peer) int {
	// Iterate over r.members until we find elements i-1 and i, which are
	// lesser and greater than p respectively.
	//
	// Beware that even though r.members is sorted, it's conceptually a
	// circle and we might start iterating in the 'greater than p' region.
	// Let's just skip over it.
	inGreater := true
	for i, el := range r.members {
		if inGreater {
			if el.ModCompare(p) <= 0 {
				inGreater = false
			}
		} else {
			if el.ModCompare(p) > 0 {
				return i
			}
		}
	}
	// okay then, p can go either at the start or the end of r.members
	return 0
}

func (r *Ring) Shutdown() {
	for _, f := range r.shutdown {
		f()
	}
}
