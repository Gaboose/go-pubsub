package gway

import (
	"errors"
	"fmt"
	"io"

	"github.com/Gaboose/go-pubsub/pnet"
	"github.com/ipfs/go-ipfs/thirdparty/multierr"
	ma "github.com/jbenet/go-multiaddr"
	manet "github.com/Gaboose/go-multiaddr-net"
	ps "github.com/jbenet/go-peerstream"
	yamux "github.com/jbenet/go-stream-muxer/yamux"
	ms "github.com/whyrusleeping/go-multistream"
)

var transport = yamux.DefaultTransport

type Gateway struct {
	conns  *ps.Swarm
	router *ms.MultistreamMuxer
}

func NewGateway() *Gateway {
	router := ms.NewMultistreamMuxer()

	conns := ps.NewSwarm(transport)
	conns.SetStreamHandler(func(s *ps.Stream) {
		go func() {
			router.Handle(s)
		}()
	})

	return &Gateway{
		conns:  conns,
		router: router,
	}
}

func (gw *Gateway) Dial(dest pnet.Peer, proto string) (io.ReadWriteCloser, error) {
	// See if we already have a connection with this peer.
	s, err := gw.conns.NewStreamWithGroup(dest.Id())
	if err != nil {
		// We don't, let's create a new connection.

		//TODO: Dial many/all addrs, use earliest. Like here:
		//https://github.com/ipfs/go-ipfs/blob/master/p2p/net/swarm/swarm_dial.go
		p, ok := dest.(*PeerInfo)
		if !ok {
			return nil, errors.New("Unknown pnet.Peer type")
		}

		m, err := ma.NewMultiaddrBytes(p.MAddrs[0])
		if err != nil {
			return nil, err
		}

		nc, err := manet.Dial(m)
		if err != nil {
			return nil, err
		}

		c, err := gw.conns.AddConn(nc)
		if err != nil {
			nc.Close()
			return nil, err
		}
		gw.conns.AddConnToGroup(c, p.Id())

		s, err = gw.conns.NewStreamWithGroup(p.Id())
		if err != nil {
			return nil, err
		}
	}

	// Select the protocol on the new stream.
	err = ms.SelectProtoOrFail(proto, s)
	if err != nil {
		s.Close()
	}

	return s, err
}

func (gw *Gateway) ListenAll(maddrs [][]byte) error {
	retErr := multierr.New()
	addErr := func(err error) {
		if retErr.Errors == nil {
			retErr.Errors = make([]error, 0)
		}
		retErr.Errors = append(retErr.Errors, err)
	}

	// listen on every address
	for _, bts := range maddrs {
		m, err := ma.NewMultiaddrBytes(bts)
		if err != nil {
			addErr(err)
			continue
		}

		err = gw.listen(m)
		if err != nil {
			addErr(err)
		}
	}

	if retErr.Errors != nil {
		return retErr
	}
	return nil
}

func (gw *Gateway) Close() error {
	return gw.conns.Close()
}

func (gw *Gateway) NewProtoNet(proto string) *ProtoNet {
	acceptCh := make(chan io.ReadWriteCloser)
	closeCh := make(chan bool)

	gw.router.AddHandler(proto, func(rwc io.ReadWriteCloser) error {
		select {
		case acceptCh <- rwc:
			return nil
		case <-closeCh:
			gw.router.RemoveHandler(proto)
			return errors.New("ProtoNet is closed")
		}
	})

	return &ProtoNet{proto, acceptCh, closeCh, gw}
}

func (gw *Gateway) listen(addr ma.Multiaddr) error {
	l, err := manet.NetListen(addr)
	if err != nil {
		return err
	}

	_, err = gw.conns.AddListener(l)
	if err == nil {
		fmt.Printf("Swarm listening on %v\n", addr)
	}
	return err
}
