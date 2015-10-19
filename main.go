package main

import (
	"flag"
	"fmt"

	"github.com/Gaboose/go-pubsub/discovery"
	"github.com/Gaboose/go-pubsub/topology"
	"github.com/cryptix/mdns"
)

func main() {
	port := flag.Int("port", 8000, "an int")
	flag.Parse()
	fmt.Println("hello world")
	id := discovery.PublishMDNS(*port)

	ring := topology.NewRing(id, 2)

	// Make a channel for results and start listening
	entriesCh := make(chan *mdns.ServiceEntry, 4)
	go func() {
		for entry := range entriesCh {
			ring.NewPeer(entry)
		}
	}()
	discovery.LookupMDNS(entriesCh)
	select {}
}
