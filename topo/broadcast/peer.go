package broadcast

import (
	"fmt"
	"io"

	"github.com/Gaboose/go-pubsub-planet/topo"
)

// Name of a parameter
const bday = "bday"

// Package specific wrapper over the common Peer interface
type Peer struct {
	topo.Peer
	conn io.ReadWriteCloser
}

// GetBday returns a rough indication of this peer profile's "birthday".
//
// A peer profile with a greater Bday can be assumed to be verified more
// recently. And so it might be a better candidate for a neighbour.
//
// Bday parameter is set by the peer sampling service, e.g. Cyclon.
// The nature of Cyclon doesn't guarantee the output of peer profiles to be
// ordered by age, which makes the Bday parameter useful.
func (p Peer) GetBday() int64 {
	i := p.Get(bday)
	switch i := i.(type) {
	case int64:
		return i
	case int:
		return int64(i)
	default:
		return 0
	}
}

func (p Peer) String() string {
	return fmt.Sprintf("%v", p.Peer)
}
