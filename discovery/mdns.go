package discovery

import (
	"fmt"
	"io/ioutil"
	golog "log"
	"os"

	"github.com/cryptix/mdns"
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

	// Create the mDNS server, defer shutdown
	mdns.NewServer(&mdns.Config{Zone: service})
	//server, _ := mdns.NewServer(&mdns.Config{Zone: service})
	//defer server.Shutdown()

	return info
}

func LookupMDNS(entriesCh chan *mdns.ServiceEntry) {

	// Start the lookup
	mdns.Lookup(ServiceTag, entriesCh)
	close(entriesCh)
}
