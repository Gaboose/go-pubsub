package topo

import "io"

type Peer interface {
	Id() interface{}
	Get(string) interface{}
	Put(string, interface{})
}

type ProtNet interface {
	Dial(Peer) (io.ReadWriteCloser, error)
	Listen() Listener
}

type Listener interface {
	Accept() (io.ReadWriteCloser, error)
	Close() error
}
