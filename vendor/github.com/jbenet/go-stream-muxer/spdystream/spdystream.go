package peerstream_spdystream

import (
	"errors"
	"net"
	"net/http"

	smux "github.com/jbenet/go-stream-muxer"
	ss "github.com/jbenet/go-stream-muxer/Godeps/_workspace/src/github.com/docker/spdystream"
)

var ErrUseServe = errors.New("not implemented, use Serve")

// stream implements smux.Stream using a ss.Stream
type stream ss.Stream

func (s *stream) spdyStream() *ss.Stream {
	return (*ss.Stream)(s)
}

func (s *stream) Read(buf []byte) (int, error) {
	return s.spdyStream().Read(buf)
}

func (s *stream) Write(buf []byte) (int, error) {
	return s.spdyStream().Write(buf)
}

func (s *stream) Close() error {
	// Reset is spdystream's full bidirectional close.
	// We expose bidirectional close as our `Close`.
	// To close only half of the connection, and use other
	// spdystream options, just get the stream with:
	//  ssStream := (*ss.Stream)(stream)
	return s.spdyStream().Reset()
}

// Conn is a connection to a remote peer.
type conn struct {
	sc *ss.Connection

	closed chan struct{}
}

func (c *conn) spdyConn() *ss.Connection {
	return c.sc
}

func (c *conn) Close() error {
	err := c.spdyConn().CloseWait()
	if !c.IsClosed() {
		close(c.closed)
	}
	return err
}

func (c *conn) IsClosed() bool {
	select {
	case <-c.closed:
		return true
	case <-c.sc.CloseChan():
		return true
	default:
		return false
	}
}

// OpenStream creates a new stream.
func (c *conn) OpenStream() (smux.Stream, error) {
	s, err := c.spdyConn().CreateStream(http.Header{
		":method": []string{"GET"}, // this is here for HTTP/SPDY interop
		":path":   []string{"/"},   // this is here for HTTP/SPDY interop
	}, nil, false)
	if err != nil {
		return nil, err
	}

	// wait for a response before writing. for some reason
	// spdystream does not make forward progress unless you do this.
	s.Wait()
	return (*stream)(s), nil
}

// AcceptStream accepts a stream opened by the other side.
func (c *conn) AcceptStream() (smux.Stream, error) {
	return nil, ErrUseServe
}

// Serve starts listening for incoming requests and handles them
// using given StreamHandler
func (c *conn) Serve(handler smux.StreamHandler) {
	c.spdyConn().Serve(func(s *ss.Stream) {

		// Flow control and backpressure of Opening streams is broken.
		// I believe that spdystream has one set of workers that both send
		// data AND accept new streams (as it's just more data). there
		// is a problem where if the new stream handlers want to throttle,
		// they also eliminate the ability to read/write data, which makes
		// forward-progress impossible. Thus, throttling this function is
		// -- at this moment -- not the solution. Either spdystream must
		// change, or we must throttle another way. go-peerstream handles
		// every new stream in its own goroutine.
		err := s.SendReply(http.Header{}, false)
		if err != nil {
			// this _could_ error out. not sure how to handle this failure.
			// don't return, and let the caller handle a broken stream.
			// better than _hiding_ an error.
			// return
		}
		go handler((*stream)(s))
	})
}

type transport struct{}

// Transport is a go-peerstream transport that constructs
// spdystream-backed connections.
var Transport = transport{}

func (t transport) NewConn(nc net.Conn, isServer bool) (smux.Conn, error) {
	sc, err := ss.NewConnection(nc, isServer)
	return &conn{sc: sc, closed: make(chan struct{})}, err
}
