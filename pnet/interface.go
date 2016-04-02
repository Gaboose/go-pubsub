package pnet

import "io"

// Peer implementations should export all fields that need to be
// transmitted over the network, because encoders can't see them otherwise.
// E.g. use []byte type for addresses instead of ma.Multiaddr, because
// ma.Multiaddr is implemented by a struct with no exported fields).
type Peer interface {
	Id() interface{}
	Get(string) interface{}
	Put(string, interface{})
}

// ProtoNet is used by topo and svice packages to connect to other nodes of the
// same protocol. Implementations of ProtoNet should isolate networks for
// different packages and protocols, by muxing streams with go-multistream,
// for example.
type ProtoNet interface {
	Dial(Peer) (io.ReadWriteCloser, error)
	Listen() Listener
}

// Listener is returned by the ProtoNet interface. It's similar to net.Listener.
type Listener interface {
	Accept() (io.ReadWriteCloser, error)
	Close() error
}
