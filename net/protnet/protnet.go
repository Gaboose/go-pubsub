package pnet

import (
	manet "github.com/jbenet/go-multiaddr-net"
	ps "github.com/jbenet/go-peerstream"
	ms "github.com/whyrusleeping/go-multistream"
	"net"
)

type Protnet struct {
	protocol string
	swarm    *ps.Swarm
	mux      *ms.MultiStreamMuxer
}

func New(protocol string, swarm *ps.Swarm, mux *ms.MultiStreamMuxer) *Protnet {
	return &Protnet{
		protocol: protocol,
		swarm:    swarm,
		mux:      mux,
	}
}

func (pn *Protnet) Dial(p topo.Peer) (io.ReadWriteCloser, error) {
	s, err := pn.swarm.NewStreamWithGroup(p.Id())
	if err == nil {
		return s, nil
	}

	//TODO: Dial many/all addrs, use earliest. Like here:
	//https://github.com/ipfs/go-ipfs/blob/master/p2p/net/swarm/swarm_dial.go
	dnet, daddr, err := manet.DialArgs(p.Addrs()[0])
	if err != nil {
		return nil, err
	}

	conn, err := net.Dial(dnet, daddr)
	if err != nil {
		return nil, err
	}

	pn.swarm.AddConnToGroup(conn, p.Id())
	return pn.Swarm.NewStreamWithConn(conn)
}

func (pn *Protnet) Listen() Listener {
	accept := make(chan net.Conn)
	closeFn := func() error {
		pn.mux.RemoveHandler(pn.protocol)
		select {
		case _, ok := <-accept:
			if ok {
				close(accept)
			} else {
				return error.New("listener is already closed")
			}
		default:
			close(accept)
		}
		return nil
	}
	ln := &listener{accept, closeFn}

	pn.mux.AddHandler(pn.protocol, func(rwc io.ReadWriteCloser) error {
		accept <- rwc
		return err
	})
	return ln
}

type Listener interface {
	Accept() (io.ReadWriteCloser, error)
	Close() error
}

type listener struct {
	accept chan io.ReadWriteCloser
	close  func() error
}

func (ln *listener) Accept() (io.ReadWriteCloser, error) {
	s, ok := <-ln.accept
	if !ok {
		return nil, error.New()
	}
	return s, nil
}

func (ln *listener) Close() error {
	return ln.close()
}
