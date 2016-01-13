package gway

import (
	"errors"
	"io"

	"github.com/Gaboose/go-pubsub/topo"
)

type ProtoNet struct {
	proto    string
	acceptCh chan io.ReadWriteCloser
	closeCh  chan bool
	gw       *Gateway
}

func (pn *ProtoNet) Dial(p topo.Peer) (io.ReadWriteCloser, error) {
	return pn.gw.Dial(p, pn.proto)
}

func (pn *ProtoNet) Listen() topo.Listener {
	return &listener{
		acceptCh: pn.acceptCh,
		closeCh:  make(chan bool),
	}
}

func (pn *ProtoNet) Close() error {
	var err error
	func() {
		defer recoverError(&err, errors.New("ProtNet already closed"))
		close(pn.closeCh)
	}()
	return err
}

type listener struct {
	acceptCh chan io.ReadWriteCloser
	closeCh  chan bool
}

func (ln *listener) Accept() (io.ReadWriteCloser, error) {
	select {
	case s, ok := <-ln.acceptCh:
		if !ok {
			return nil, errors.New("ProtoNet is closed")
		}
		return s, nil
	case <-ln.closeCh:
		return nil, errors.New("Listener is closed")
	}
}

func (ln *listener) Close() error {
	var err error
	func() {
		defer recoverError(&err, errors.New("Listener already closed"))
		close(ln.closeCh)
	}()
	return err
}

func recoverError(maybeErr *error, err error) {
	r := recover()
	if r != nil {
		*maybeErr = err
	}
}
