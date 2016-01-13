package net

import (
	"encoding/gob"
	"fmt"
	"strings"
	"time"

	"github.com/Gaboose/go-pubsub/gway"
	"github.com/Gaboose/go-pubsub/topo/broadcast"
	"github.com/Gaboose/go-pubsub/topo/cyclon"

	ps "github.com/briantigerchow/pubsub"
)

type Network struct {
	gw  *gway.Gateway
	cyc *cyclon.Cyclon
	bro *broadcast.Broadcast
	rtr *ps.PubSub
}

func NewNetwork(me *gway.PeerInfo) (*Network, error) {
	gw := gway.NewGateway()
	err := gw.ListenAll(me.MAddrs)
	if err != nil {
		return nil, err
	}

	gob.Register(&gway.PeerInfo{})

	c := cyclon.New(me, 30, 10, gw.NewProtoNet("/cyclon"))
	c.Start(time.Second)

	b := broadcast.New(2, time.Minute, gw.NewProtoNet("/broadcast"))
	b.Start(c.Out(), 30)

	r := ps.New(1)
	go route(b.Out(), r)

	return &Network{
		gw:  gw,
		cyc: c,
		bro: b,
		rtr: r,
	}, nil
}

func (n *Network) Connect(p *gway.PeerInfo) { n.cyc.Add(p) }

func (n *Network) Pub(msg string, topic string) {
	n.bro.In() <- fmt.Sprintf("/%s/%s", topic, msg)
}

func (n *Network) Sub(topic string) (<-chan interface{}, chan<- bool) {
	unsub := make(chan bool)
	ch := n.rtr.Sub(topic)
	go func() {
		<-unsub
		n.rtr.Unsub(ch)
	}()
	return ch, unsub
}

func route(in <-chan string, rtr *ps.PubSub) {
	for {
		msg := string(<-in)
		if strings.Count(msg, "/") < 2 || msg[0] != '/' {
			fmt.Println("Unspecified topic")
			continue
		}
		s := strings.SplitN(msg[1:], "/", 2)
		topic := s[0]
		msg = s[1]

		rtr.Pub(msg, topic)
	}
}
