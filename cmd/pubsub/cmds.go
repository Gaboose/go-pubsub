package main

import (
	"bufio"
	"fmt"
	"io"
	"time"
	"github.com/Gaboose/go-pubsub/pnet/gway"
	"github.com/Gaboose/go-pubsub/svice/ping"
	maddr "github.com/jbenet/go-multiaddr"
)

func init() {
	// Commands that reference the command map can use this pointer
	// to bypass init reference loop checks.
	commandsPtr = &commands
}

var commandsPtr *CommandMap
var commands = CommandMap{

	"help": &LocalCommand{
		help: `Usage: pubsub help <command> - Print usage of a given command`,
		Run: func(args []string, stdio io.ReadWriter) byte {
			if len(args) > 0 {
				name := args[0]
				c, ok := (*commandsPtr)[name]
				if ok {
					fmt.Println(c.Help())
				} else {
					fmt.Printf("Unknown command: %s\n", name)
					return 1
				}
			} else {
				fmt.Println(`
Usage: pubsub [<flags>] <command> [ -help | <args> ] - p2p pubsub network

COMMANDS:
	daemon			Run a network-connected pubsub node
	pub <topic> <msg>	Publish a message
	sub <topic>		Listen for and receive messages

FLAGS:
	-apiport	int	- Port of the daemon API to connect to

Use 'pubsub <command> -help' for more information about a command.
`)
			}
			return 0
		},
	},

	"daemon": &LocalCommand{
		help: `
Usage: pubsub daemon [<flags>] - Run a network-connected PubSub node

FLAGS:
	-apiport	int	- Port for the daemon API to listen on
	-swarmport	int	- Port to listen for other nodes on
`,
		Run: startDaemon,
	},

	"status": &RemoteCommand{
		help: "Prints arbitrary status",
		Run: func(args []string, stdio io.ReadWriter) byte {
			fmt.Fprintln(stdio, "Hello Status")
			fmt.Fprintf(stdio, "daemon: %v\n", daemon)
			return 0
		},
	},
		
	"ping": &LocalCommand{
		help: "Usage: pubsub ping <maddr> - Try to contact a remote ping service",
		Run: func(args []string, stdio io.ReadWriter) byte {
			m, err := maddr.NewMultiaddr(args[0])
			if err != nil {
				fmt.Fprintln(stdio, err)
				return 1
			}
			
			gw := gway.NewGateway()
			png := ping.Ping{ProtoNet: gw.NewProtoNet("/ping")}
			p := &gway.PeerInfo{MAddrs: [][]byte{m.Bytes()}}
			
			stop := make(chan bool)
			done := make(chan error)
			
			go func() {
				done <- png.Ping(p, stop)
			}()
			
			select {
			case err = <-done:
				if err != nil {
					fmt.Fprintln(stdio, err)
					return 1
				}
				fmt.Fprintln(stdio, "successful contact")
				return 0
			case <-time.After(5*time.Second):
				close(stop)
				fmt.Fprintln(stdio, "timed out")
				return 1
			}
		},
	},

	"sub": &RemoteCommand{
		help: `Usage: pubsub sub <topic> - Receive/publish messages to/from stdout/stdin`,
		Run: func(args []string, stdio io.ReadWriter) byte {
			if len(args) != 1 {
				fmt.Fprintln(stdio, (*commandsPtr)["sub"].Help())
				return 1
			}

			done := make(chan bool)
			go func() {
				b := bufio.NewReader(stdio)
				for {
					s, err := b.ReadString('\n')
					if err != nil {
						done <- true
						return
					}

					// don't forget to remove the new line at the end of s
					daemon.Pub(s[:len(s)-1], args[0])
				}
			}()

			ch, unsub := daemon.Sub(args[0])
			for {
				select {
				case msg := <-ch:
					fmt.Fprintln(stdio, msg)
				case <-done:
					close(unsub)
					return 1
				}
			}
		},
	},

	"pub": &RemoteCommand{
		help: `Usage: pubsub pub <topic> <message> - Publish a message`,
		Run: func(args []string, stdio io.ReadWriter) byte {
			if len(args) != 2 {
				fmt.Fprintln(stdio, (*commandsPtr)["pub"].Help())
				return 1
			}

			daemon.Pub(args[1], args[0])
			return 0
		},
	},
}
