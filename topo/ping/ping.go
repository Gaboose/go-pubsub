package ping

import (
	"errors"

	"github.com/Gaboose/go-pubsub/topo"
)

const msg = "pong"

// Ping service is for checking peer availability.
//
// Originally written, because OpenShift has this bug, where waking "gears",
// for the first 10 seconds or so, accept websocket connections, but they're
// left there in limbo: no data is exchanged and the connection is left open
// indefinitely.
//
// Now they can be timed out when we wait too long on the "pong" message.
type Ping struct {
	protonet topo.ProtoNet
	ln       topo.Listener
}

// Ping returns nil if a predefined response is received,
// otherwise returns an error.
func (p *Ping) Ping(t topo.Peer, stop chan bool) error {
	c, err := p.protonet.Dial(t)
	if err != nil {
		return err
	}
	defer c.Close()

	if stop != nil {
		go func() {
			<-stop
			c.Close()
		}()
	}

	bs := make([]byte, 256)
	n, err := c.Read(bs)
	if err != nil {
		return err
	}
	if string(bs[:n]) != msg {
		return errors.New("invalid response")
	}

	return nil
}

// Serve starts listening for incoming pings.
func (p *Ping) Serve() {
	if p.ln != nil {
		panic(errors.New("ping is already serving"))
	}

	ln := p.protonet.Listen()
	go func() {
		c, err := ln.Accept()
		if err != nil {
			return
		}

		c.Write([]byte(msg))
		c.Close()
	}()

	p.ln = ln
}

// Stop closes the listener inside.
func (p *Ping) Stop() {
	p.ln.Close()
}
