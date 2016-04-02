package ponger

import (
	"github.com/Gaboose/go-pubsub/pnet/gway"
    "github.com/Gaboose/go-pubsub/svice/ping"
)

func NewNetwork(me *gway.PeerInfo) error {
	gw := gway.NewGateway()
	err := gw.ListenAll(me.MAddrs)
	if err != nil {
		return err
	}
    
    (&ping.Ping{ProtoNet: gw.NewProtoNet("/ping")}).Serve()

	return nil
}
