package net

import (
	"net"

	"github.com/ipfs/go-ipfs/thirdparty/multierr"
	ma "github.com/jbenet/go-multiaddr"
	manet "github.com/jbenet/go-multiaddr-net"
	ps "github.com/jbenet/go-peerstream"
	psy "github.com/jbenet/go-stream-muxer/yamux"
	"github.com/Gaboose/go-pubsub/peer"
)

type Network struct {
	swarm *ps.Swarm
	local peer.ID
	peers peer.Peerstore
}

func NewNetwork(listenAddrs []ma.Multiaddr, local peer.ID,
	peers peer.Peerstore) (*Network, error) {

	n := &Network{
		swarm: ps.NewSwarm(psy.DefaultTransport),
		local: local,
		peers: peers,
	}

	return n, n.listenMulti(listenAddrs)
}

func (n *Network) listenMulti(addrs []ma.Multiaddr) error {
	retErr := multierr.New()

	// listen on every address
	for i, addr := range addrs {
		err := n.listen(addr)
		if err != nil {
			if retErr.Errors == nil {
				retErr.Errors = make([]error, len(addrs))
			}
			retErr.Errors[i] = err
		}
	}

	if retErr.Errors != nil {
		return retErr
	}
	return nil
}

func (n *Network) listen(addr ma.Multiaddr) error {
	lnet, lnaddr, err := manet.DialArgs(addr)
	if err != nil {
		return err
	}

	l, err := net.Listen(lnet, lnaddr)
	if err != nil {
		return err
	}

	_, err = n.swarm.AddListener(l)
	return err
}
