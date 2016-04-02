package main

import (
	"fmt"
	"io/ioutil"

	"github.com/Gaboose/go-pubsub/pnet/gway"
	"github.com/Gaboose/mdns"
	ma "github.com/jbenet/go-multiaddr"
	manet "github.com/jbenet/go-multiaddr-net"
)

const ServiceTag = "discovery.github.com.Gaboose.go-pubsub"

func init() {
	mdns.Log.SetOutput(ioutil.Discard)
}

func PublishMDNS(host string, port int) {
	info := fmt.Sprintf("go-pubsub(%s)", host)
	service, _ := mdns.NewMDNSService(host, ServiceTag, "", "", port, nil, []string{info})

	// Create and run the mDNS server, don't shudown
	mdns.NewServer(&mdns.Config{Zone: service})
}

func LookupMDNS() []*gway.PeerInfo {
	// Collect channel messages to a slice
	prs := []*gway.PeerInfo{}
	ch := make(chan *mdns.ServiceEntry, 4)
	go func() {
		for e := range ch {
			p := &gway.PeerInfo{
				ID:     e.Info,
				MAddrs: MultiaddrsFromServiceEntry(e),
			}
			prs = append(prs, p)
		}
	}()

	// Start the lookup
	mdns.Lookup(ServiceTag, ch)
	close(ch)
	return prs
}

func MultiaddrsFromServiceEntry(en *mdns.ServiceEntry) [][]byte {
	var addrs []ma.Multiaddr

	// Parse IP addresses
	addr, err := manet.FromIP(en.AddrV4)
	if err == nil {
		addrs = append(addrs, addr)
	}
	addr, err = manet.FromIP(en.AddrV6)
	if err == nil {
		addrs = append(addrs, addr)
	}

	var bytes [][]byte
	for _, addr := range addrs {
		// Append port
		prt, err := ma.NewMultiaddr(fmt.Sprintf("/tcp/%d", en.Port))
		if err != nil {
			continue
		}
		addr = addr.Encapsulate(prt)

		bytes = append(bytes, addr.Bytes())
	}

	return bytes
}
