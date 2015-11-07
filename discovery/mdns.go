package discovery

import (
	"fmt"
	"io/ioutil"
	golog "log"
	"os"

	"github.com/Gaboose/mdns"
	ma "github.com/jbenet/go-multiaddr"
	manet "github.com/jbenet/go-multiaddr-net"
	"github.com/Gaboose/go-pubsub/peer"
)

const ServiceTag = "discovery.github.com.Gaboose.go-pubsub"

func PublishMDNS(port int) string {
	// TODO: dont let mdns use logging...
	golog.SetOutput(ioutil.Discard)

	// Setup our service export
	host, _ := os.Hostname()
	// Concatenating host with port allows to discover several instances on the same machine
	host = fmt.Sprintf("%s.%d", host, port)
	info := fmt.Sprintf("go-pubsub(%s)", host)
	service, _ := mdns.NewMDNSService(host, ServiceTag, "", "", port, nil, []string{info})

	// Create and run the mDNS server, don't shudown
	mdns.NewServer(&mdns.Config{Zone: service})

	return info
}

func LookupMDNS() []*peer.PeerInfo {
	// Collect channel messages to a slice
	prs := []*peer.PeerInfo{}
	ch := make(chan *mdns.ServiceEntry, 4)
	go func() {
		for e := range ch {
			p := &peer.PeerInfo{
				ID:    peer.IDFromString(e.Info),
				Addrs: MultiaddrsFromServiceEntry(e),
			}
			prs = append(prs, p)
		}
	}()

	// Start the lookup
	mdns.Lookup(ServiceTag, ch)
	close(ch)
	return prs
}

func MultiaddrsFromServiceEntry(en *mdns.ServiceEntry) []ma.Multiaddr {
	ads := []ma.Multiaddr{}

	// Parse IP addresses
	ad, err := manet.FromIP(en.AddrV4)
	if err == nil {
		ads = append(ads, ad)
	}
	ad, err = manet.FromIP(en.AddrV6)
	if err == nil {
		ads = append(ads, ad)
	}

	// Append ports
	for i, ad := range ads {
		prt, _ := ma.NewMultiaddr(fmt.Sprintf("/tcp/%d", en.Port))
		ads[i] = ad.Encapsulate(prt)
	}

	return ads
}
