package main

import (
	"io"
	"math/rand"
	"net"

	smux "github.com/jbenet/go-stream-muxer"
	ymux "github.com/jbenet/go-stream-muxer/yamux"
	sroute "github.com/whyrusleeping/go-multistream"
)

var transport = ymux.DefaultTransport

type ErrorCodeRWC interface {
	io.Reader
	io.Writer
	Close(ec byte) error
	ErrorCodeCh() <-chan byte
}

// WithErrorCode wraps a connection and enables you to close it with an error
// code on one side and read it on the other after the connection is closed.
//
// Both ends of a connection must wrap it with this function for it to work.
func WithErrorCode(whole net.Conn) (ErrorCodeRWC, error) {

	// Throw dice (write and read a random byte) to determine who's going
	// to be multiplex server and client.
	writeErrCh := make(chan error)
	var myDice byte
	var theirDice [1]byte
	for {
		myDice = byte(rand.Intn(256))
		go func() {
			_, err := whole.Write([]byte{myDice})
			writeErrCh <- err
		}()

		_, err := whole.Read(theirDice[:])
		if err != nil {
			return nil, err
		}

		err = <-writeErrCh
		if err != nil {
			return nil, err
		}

		if myDice != theirDice[0] {
			break
		}
	}

	if myDice > theirDice[0] {
		return client(whole)
	} else {
		return server(whole)
	}
}

func (ecr *errorCodeRWC) Read(p []byte) (int, error) {
	return ecr.mainPeel.Read(p)
}
func (ecr *errorCodeRWC) Write(p []byte) (int, error) {
	return ecr.mainPeel.Write(p)
}

func (ecr *errorCodeRWC) Close(ec byte) error {
	_, err := ecr.ecPeel.Write([]byte{ec})
	if err != nil {
		return err
	}
	return ecr.whole.Close()
}

func (ecr *errorCodeRWC) ErrorCodeCh() <-chan byte {
	return ecr.ecCh
}

type errorCodeRWC struct {
	whole    net.Conn
	mainPeel io.ReadWriteCloser
	ecPeel   io.ReadWriteCloser
	ecCh     chan byte
}

func client(whole net.Conn) (ErrorCodeRWC, error) {
	sconn, _ := transport.NewConn(whole, false)
	go sconn.Serve(func(smux.Stream) {})

	mainPeel, err := sconn.OpenStream()
	if err != nil {
		return nil, err
	}

	err = sroute.SelectProtoOrFail("/main", mainPeel)
	if err != nil {
		return nil, err
	}

	ecPeel, err := sconn.OpenStream()
	if err != nil {
		return nil, err
	}

	err = sroute.SelectProtoOrFail("/ec", ecPeel)
	if err != nil {
		return nil, err
	}

	ecCh := make(chan byte)
	go readToCh(ecPeel, ecCh)

	return &errorCodeRWC{whole, mainPeel, ecPeel, ecCh}, nil
}

func server(whole net.Conn) (ErrorCodeRWC, error) {

	mainPeelCh := make(chan io.ReadWriteCloser)
	ecPeelCh := make(chan io.ReadWriteCloser)
	ecCh := make(chan byte)

	rt := sroute.NewMultistreamMuxer()
	rt.AddHandler("/main", func(mainPeel io.ReadWriteCloser) error {
		mainPeelCh <- mainPeel
		return nil
	})
	rt.AddHandler("/ec", func(ecPeel io.ReadWriteCloser) error {
		ecPeelCh <- ecPeel
		go readToCh(ecPeel, ecCh)
		return nil
	})

	sc, err := transport.NewConn(whole, true)
	if err != nil {
		return nil, err
	}

	go sc.Serve(func(s smux.Stream) {
		rt.Handle(s)
	})

	return &errorCodeRWC{whole, <-mainPeelCh, <-ecPeelCh, ecCh}, nil
}

func readToCh(rc io.ReadCloser, out chan<- byte) {
	var b [1]byte
	n, _ := rc.Read(b[:])
	if n == 1 {
		out <- b[0]
	}
	rc.Close()
	close(out)
}
