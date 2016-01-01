package main

import (
	"fmt"
	"io"
	"math/rand"
	"net"
	"os"
	"time"
	//"github.com/Gaboose/go-pubsub/net"
	//"github.com/Gaboose/go-pubsub/topo"
	ma "github.com/jbenet/go-multiaddr"
	manet "github.com/jbenet/go-multiaddr-net"
	mux "github.com/jbenet/go-multicodec/mux"
)

type command struct {
	Helptext string
	Run      func(args []string, rwc io.ReadWriter) byte
}

type LocalCommand command
type RemoteCommand command

type CommandMap map[string]interface{}

var commands = CommandMap{
	"hello-world": &RemoteCommand{
		Helptext: "Greets you",
		Run: func(args []string, stdio io.ReadWriter) byte {
			fmt.Fprintln(stdio, "Hello Pluto")
			return 0
		},
	},
}

func init() {
	rand.Seed(time.Now().UTC().UnixNano())

	// daemonCmd is set separately to bypass init loop checks
	commands["daemon"] = daemonCmd
}

func (cm CommandMap) Call(name string, args []string, stdio io.ReadWriter) byte {
	cmd, ok := cm[name]
	if !ok {
		fmt.Fprintf(stdio, "Unknown command: %s\n", name)
		return 1
	}

	switch cmd := cmd.(type) {

	case *LocalCommand:
		return cmd.Run(args, stdio)

	case *RemoteCommand:
		m, err := ma.NewMultiaddr(APIAddr)
		if err != nil {
			fmt.Fprintln(stdio, err)
			return 1
		}

		conn, err := manet.Dial(m)
		if err != nil {
			fmt.Fprintln(stdio, err)
			return 1
		}
		defer conn.Close()

		mx := mux.StandardMux()
		err = mx.Encoder(conn).Encode(append([]string{name}, args...))
		if err != nil {
			fmt.Fprintln(stdio, err)
			return 1
		}

		remoteStdio, err := WithErrorCode(conn)
		if err != nil {
			fmt.Fprintln(stdio, err)
			return 1
		}

		go io.Copy(remoteStdio, stdio)
		_, err = io.Copy(stdio, remoteStdio)
		if err != nil {
			fmt.Fprintln(stdio, err)
			return 1
		}

		return <-remoteStdio.ErrorCodeCh()

	default:
		fmt.Fprintf(stdio, "Unknown type of command: %s\n", name)
		return 1
	}
}

func printHelp() {
	fmt.Println("don't ask me")
}

func main() {
	if len(os.Args) < 2 || os.Args[1] == "help" {
		printHelp()
		return
	}

	cmdName := os.Args[1]
	args := os.Args[2:]

	c1, c2 := net.Pipe()
	done := make(chan byte)
	go func() {
		ec := commands.Call(cmdName, args, c2)
		c2.Close()
		done <- ec
	}()
	io.Copy(os.Stdout, c1)

	//why not just ec := commands.Call(cmdName, args, os.Stdout)?

	os.Exit(int(<-done))
}
