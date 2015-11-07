package topo

// To use topologies, external packages will have to implement
// this interface.
type Network interface {
	// applies to every peer in peerstore
	AlivePeerIterator() <-chan Peer
	AlivePeerNotifier() (<-chan Peer, func())

	// applies only to peers, with which there was a topo connection
	DeadMemberNotifier() (<-chan Peer, func())

	//Send(PeerInfo, []bytes) // to communicate weights, component ids, etc.
	ConnectMember(Peer)    // establish a topo connection
	DisconnectMember(Peer) // remove a topo connection
}

type Peer interface {
	ModComparable
}

type ModComparable interface {
	ModCompare(ModComparable) int
}
