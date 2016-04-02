package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/Gaboose/go-pubsub/pnet/gway"
	psnet "github.com/Gaboose/go-pubsub/net/cycbro"

	ma "github.com/jbenet/go-multiaddr"
	manet "github.com/jbenet/go-multiaddr-net"
	mux "github.com/jbenet/go-multicodec/mux"
)

var daemon *psnet.Network

func startDaemon(args []string, stdio io.ReadWriter) byte {
	fs := flag.NewFlagSet("pubsub daemon", flag.ContinueOnError)
	apiport := fs.Int("apiport", 5002, "Port for the daemon API to listen on")
	swarmport := fs.Int("swarmport", 4002, "Port to listen for other nodes on")
	err := fs.Parse(args)
	if err != nil {
		return 1
	}

	fmt.Println("Initializing daemon...")

	err = apiServer(*apiport)
	if err != nil {
		fmt.Println(err)
		return 1
	}

	err = network(*swarmport)
	if err != nil {
		fmt.Println(err)
		return 1
	}

	select {}
}

func network(port int) error {
	// Concatenating id with port allows us to run several
	// daemons on the same machine
	id, _ := os.Hostname()
	id = fmt.Sprintf("%s.%d", id, port)

	ready := make(chan *psnet.Network)

	go func() {
		// Look for peers to bootstrap our network with
		peers := LookupMDNS()

		// Advertise yourself on mDNS
		PublishMDNS(id, port)

		n := <-ready
		if n == nil {
			return
		}

		for _, p := range peers {
			n.Connect(p)
		}
	}()

	// Create our own PeerInfo
	me := &gway.PeerInfo{
		ID:     id,
		MAddrs: BuildSwarmAddrs(port),
	}

	// Start the network
	n, err := psnet.NewNetwork(me)
	if err != nil {
		ready <- nil
		return err
	}

	daemon = n
	ready <- n

	return nil
}

func apiServer(port int) error {
	m := BuildAPIAddr(port)
	ln, err := manet.Listen(m)
	if err != nil {
		return err
	}

	fmt.Printf("API server listening on %v\n", m)

	// Go accept API requests.
	go func() {
		defer ln.Close()

		for {
			conn, err := ln.Accept()
			if err != nil {
				fmt.Println(err)
				return
			}

			// Go handle the new connection.
			go func(con manet.Conn) {
				defer con.Close()

				// Decode arguments.
				var args []string
				mx := mux.StandardMux()
				err = mx.Decoder(con).Decode(&args)
				if err != nil {
					fmt.Println(err)
					return
				}

				cmdName := args[0]
				args = args[1:]

				// Get the command object
				v, ok := (*commandsPtr)[cmdName]
				if !ok {
					fmt.Printf("Unknown command: %s\n", cmdName)
					return
				}
				cmd, ok := v.(*RemoteCommand)
				if !ok {
					fmt.Printf("%s is not a remote command\n", cmdName)
					return
				}

				// Multiplex connection
				remoteStdio, err := WithErrorCode(con)
				if err != nil {
					fmt.Println(err)
					return
				}

				// Run cmd. Provide remote connection as stdin/stdout.
				ec := cmd.Run(args, remoteStdio)
				remoteStdio.Close(ec)
			}(conn)
		}
	}()

	return nil
}

// Builds and returns the Multiaddrs for the swarm to listen on
func BuildSwarmAddrs(port int) [][]byte {
	// Get all interface addresses
	all, _ := manet.InterfaceMultiaddrs()

	// Filter out loopback and link-local addresses
	var filtered []ma.Multiaddr
	for _, m := range all {
		if manet.IsIPLoopback(m) {
			continue
		}
		if manet.IsIP6LinkLocal(m) {
			continue
		}
		filtered = append(filtered, m)
	}

	// Add tcp/<port> to each address and convert to byte representation
	prt, err := ma.NewMultiaddr(fmt.Sprintf("/tcp/%d", port))
	if err != nil {
		panic(err)
	}

	var listenAddrs [][]byte
	for _, a := range filtered {
		listenAddrs = append(listenAddrs, a.Encapsulate(prt).Bytes())
	}
	return listenAddrs
}

func BuildAPIAddr(port int) ma.Multiaddr {
	return manet.IP4Loopback.Encapsulate(
		ma.StringCast(fmt.Sprint("/tcp/", port)))
}
