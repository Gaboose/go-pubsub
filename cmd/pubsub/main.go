package main

import (
	"flag"
	"io"
	"math/rand"
	"os"
	"time"
)

func init() {
	rand.Seed(time.Now().UTC().UnixNano())
}

func main() {

	// Parse flags before the subcommand
	fs := flag.NewFlagSet("pubsub", flag.ExitOnError)
	port := fs.Int("apiport", 5002, "Port of the daemon API to connect to")
	fs.Parse(os.Args[1:])
	args := fs.Args()

	var cmdName string
	if len(args) < 1 {
		cmdName = "help"
	} else {
		cmdName = args[0]
		args = args[1:]

		// Parse a potential help flag after the subcommand
		fs := NewFlagSet("", flag.ContinueOnError)
		help := fs.Bool("help", false, "")
		fs.ParseDefined(args)
		args = fs.Undefined()

		if *help {
			args = []string{cmdName}
			cmdName = "help"
		}
	}

	stdio := struct {
		io.Reader
		io.Writer
	}{os.Stdin, os.Stdout}

	ec := commands.Call(cmdName, args, stdio, BuildAPIAddr(*port))

	os.Exit(int(ec))
}
