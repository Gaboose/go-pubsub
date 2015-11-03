package peer

import (
	"time"

	ma "github.com/jbenet/go-multiaddr"
)

type Peerstore interface {
	AddrBook

	// Peers returns a list of all peer.IDs in this Peerstore
	Peers() []ID

	// PeerInfo returns a peer.PeerInfo struct for given peer.ID.
	// This is a small slice of the information Peerstore has on
	// that peer, useful to other services.
	PeerInfo(ID) PeerInfo
}

// AddrBook is an interface that fits the new AddrManager. I'm patching
// it up in here to avoid changing a ton of the codebase.
type AddrBook interface {

	// AddAddr calls AddAddrs(p, []ma.Multiaddr{addr}, ttl)
	AddAddr(p ID, addr ma.Multiaddr, ttl time.Duration)

	// AddAddrs gives AddrManager addresses to use, with a given ttl
	// (time-to-live), after which the address is no longer valid.
	// If the manager has a longer TTL, the operation is a no-op for that address
	AddAddrs(p ID, addrs []ma.Multiaddr, ttl time.Duration)

	// SetAddr calls mgr.SetAddrs(p, addr, ttl)
	SetAddr(p ID, addr ma.Multiaddr, ttl time.Duration)

	// SetAddrs sets the ttl on addresses. This clears any TTL there previously.
	// This is used when we receive the best estimate of the validity of an address.
	SetAddrs(p ID, addrs []ma.Multiaddr, ttl time.Duration)

	// Addresses returns all known (and valid) addresses for a given
	Addrs(p ID) []ma.Multiaddr
}

type peerstore struct {
	AddrManager
}

func NewPeerstore() Peerstore {
	return &peerstore{
		AddrManager: AddrManager{},
	}
}

func (ps *peerstore) Peers() []ID {
	set := map[ID]struct{}{}
	for _, p := range ps.AddrManager.Peers() {
		set[p] = struct{}{}
	}

	pps := make([]ID, 0, len(set))
	for p := range set {
		pps = append(pps, p)
	}
	return pps
}

func (ps *peerstore) PeerInfo(p ID) PeerInfo {
	return PeerInfo{
		ID:    p,
		Addrs: ps.AddrManager.Addrs(p),
	}
}
