package main

import (
	"flag"
	"fmt"
	"time"

	ma "github.com/jbenet/go-multiaddr"
	manet "github.com/jbenet/go-multiaddr-net"
	"github.com/Gaboose/go-pubsub/discovery"
	"github.com/Gaboose/go-pubsub/net"
	"github.com/Gaboose/go-pubsub/peer"
)

func main() {
	port := flag.Int("port", 8000, "")
	flag.Parse()
	fmt.Println("greetings earth")

	// Populate peerstore from mDNS
	peers := discovery.LookupMDNS()
	peerstore := peer.NewPeerstore()
	for _, p := range peers {
		peerstore.AddAddrs(p.ID, p.Addrs, time.Second*60)
	}

	// Advertise yourself on mDNS
	id := discovery.PublishMDNS(*port)

	// Prepare listen addresses
	ifaceAddrs, _ := manet.InterfaceMultiaddrs()
	listenAddrs := []ma.Multiaddr{}
	prt, err := ma.NewMultiaddr(fmt.Sprintf("/tcp/%d", *port))
	if err != nil {
		fmt.Println(err)
		return
	}
	for _, a := range ifaceAddrs {
		listenAddrs = append(listenAddrs, a.Encapsulate(prt))
	}

	// Start the network
	net.NewNetwork(listenAddrs, peer.IDFromString(id), peerstore)

	// Wait
	select {}
}
