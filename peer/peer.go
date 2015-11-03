package peer

import ma "github.com/jbenet/go-multiaddr"

// ID represents the identity of a peer.
type ID string

func IDFromString(s string) ID {
	return ID(s)
}

// PeerInfo is a small struct used to pass around a peer with
// a set of addresses. This is not meant to be a complete view
// of the system, but rather to model updates to the peerstore.
type PeerInfo struct {
	ID    ID
	Addrs []ma.Multiaddr
}
