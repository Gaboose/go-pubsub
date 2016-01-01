package main

import (
	"fmt"
	ma "github.com/jbenet/go-multiaddr"
	manet "github.com/jbenet/go-multiaddr-net"
	mux "github.com/jbenet/go-multicodec/mux"
	"io"
)

var APIAddr = "/ip4/127.0.0.1/tcp/5002"

var daemonCmd = &LocalCommand{
	Helptext: "Daemon help text",
	Run: func(args []string, stdio io.ReadWriter) byte {
		fmt.Fprintln(stdio, "I'm a daemon")

		m, err := ma.NewMultiaddr(APIAddr)
		if err != nil {
			fmt.Fprintln(stdio, err)
			return 1
		}

		ln, err := manet.Listen(m)
		if err != nil {
			fmt.Fprintln(stdio, err)
			return 1
		}
		defer ln.Close()

		fmt.Fprintf(stdio, "API server listening on %v\n", m)

		go apiServer(ln, stdio)

		select {}
	},
}

func apiServer(ln manet.Listener, stdio io.ReadWriter) {
	for {
		conn, err := ln.Accept()
		if err != nil {
			fmt.Fprintln(stdio, err)
		}
		go func(con manet.Conn) {
			defer con.Close()

			mx := mux.StandardMux()

			var args []string
			err = mx.Decoder(con).Decode(&args)
			if err != nil {
				fmt.Fprintln(stdio, err)
				return
			}

			cmdName := args[0]
			args = args[1:]

			v, ok := commands[cmdName]
			if !ok {
				fmt.Fprintf(stdio, "Unknown command: %s\n", cmdName)
				return
			}
			cmd, ok := v.(*RemoteCommand)
			if !ok {
				fmt.Fprintf(stdio, "%s is not a remote command\n", cmdName)
				return
			}

			remoteStdio, err := WithErrorCode(con)
			if err != nil {
				fmt.Fprintln(stdio, err)
				return
			}

			ec := cmd.Run(args, remoteStdio)
			remoteStdio.Close(ec)
		}(conn)
	}
}
