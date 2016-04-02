package mock

import (
	"errors"
	"fmt"
	"github.com/Gaboose/go-pubsub/pnet"
	"io"
	"net"
)

type ProtoNetSwarm map[interface{}]*listener

func (sw ProtoNetSwarm) DialListener(peerid interface{}) *ProtoNet {
	return &ProtoNet{sw, peerid}
}

type ProtoNet struct {
	sw ProtoNetSwarm
	id interface{}
}

func (pn *ProtoNet) Dial(p pnet.Peer) (io.ReadWriteCloser, error) {
	ln, ok := pn.sw[p.Id()]
	if !ok {
		msg := fmt.Sprintf("%s isn't listening", p.Id())
		return nil, errors.New(msg)
	}
	conn1, conn2 := net.Pipe()
	ln.accept <- conn1
	return conn2, nil
}

func (pn *ProtoNet) Listen() pnet.Listener {
	accept := make(chan io.ReadWriteCloser)
	closeFn := func() error {
		close(accept)
		delete(pn.sw, pn.id)
		return nil
	}
	ln := &listener{accept, closeFn}
	pn.sw[pn.id] = ln
	return ln
}

type listener struct {
	accept chan io.ReadWriteCloser
	close  func() error
}

func (ln *listener) Accept() (io.ReadWriteCloser, error) {
	s, ok := <-ln.accept
	if !ok {
		return nil, errors.New("accept channel is closed")
	}
	return s, nil
}

func (ln *listener) Close() error {
	return ln.close()
}
