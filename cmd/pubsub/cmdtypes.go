package main

import (
	"fmt"
	"io"

	ma "github.com/jbenet/go-multiaddr"
	manet "github.com/Gaboose/go-multiaddr-net"
	mux "github.com/jbenet/go-multicodec/mux"
)

type CommandMap map[string]Helper

type Helper interface {
	Help() string
}

type LocalCommand command
type RemoteCommand command

type command struct {
	help
	Run func([]string, io.ReadWriter) byte
}

type help string

func (h help) Help() string { return string(h) }

func (cm CommandMap) Call(name string, args []string, stdio io.ReadWriter, apiaddr ma.Multiaddr) byte {
	cmd, ok := cm[name]
	if !ok {
		fmt.Printf("Unknown command: %s\n", name)
		return 1
	}

	switch cmd := cmd.(type) {

	case *LocalCommand:
		return cmd.Run(args, stdio)

	case *RemoteCommand:

		// Connect to daemon
		conn, err := manet.Dial(apiaddr)
		if err != nil {
			fmt.Println(err)
			return 1
		}
		defer conn.Close()

		// Send arguments
		mx := mux.StandardMux()
		err = mx.Encoder(conn).Encode(append([]string{name}, args...))
		if err != nil {
			fmt.Println(err)
			return 1
		}

		// Multiplex connection
		remoteStdio, err := WithErrorCode(conn)
		if err != nil {
			fmt.Println(err)
			return 1
		}

		// Join stdout and stdin with remote ones
		go io.Copy(remoteStdio, stdio)
		_, err = io.Copy(stdio, remoteStdio)
		if err != nil {
			fmt.Println(err)
			return 1
		}

		return <-remoteStdio.ErrorCodeCh()

	default:
		fmt.Printf("Unknown type of command: %s\n", name)
		return 1
	}
}
