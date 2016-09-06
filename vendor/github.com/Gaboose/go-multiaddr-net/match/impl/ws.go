package impl

import (
	"errors"
	"fmt"
	"github.com/Gaboose/go-multiaddr-net/match"
	"io"
	"net"
	"net/http"

	"golang.org/x/net/websocket"

	ma "github.com/jbenet/go-multiaddr"
)

func init() {
	ma.AddProtocol(ma.Protocol{481, -1, "ws", ma.CodeToVarint(481)})
}

type WS struct{}

func (w WS) Match(m ma.Multiaddr, side int) (int, bool) {
	ps := m.Protocols()

	if len(ps) >= 1 && ps[0].Name == "ws" {
		return 1, true
	}

	// If we're a client, also match "/http/ws".
	if side == match.S_Client && len(ps) >= 2 &&
		ps[0].Name == "http" && ps[1].Name == "ws" {

		return 2, true
	}

	return 0, false
}

func (w WS) Apply(m ma.Multiaddr, side int, ctx match.Context) error {
	var path string
	// ws client matches /http/ws too, so /ws might not be the first protocol
	for _, p := range m.Protocols() {
		if p.Name == "ws" {
			pth, err := m.ValueForProtocol(p.Code)
			if err != nil {
				return err
			}
			path = pth
			break
		}
	}
	sctx := ctx.Special()
	mctx := ctx.Misc()

	switch side {

	case match.S_Client:
		// resolve url
		var url string
		if mctx.Host != "" {
			// this will make mctx.Host appear in http request headers,
			// cloud servers often require that
			url = fmt.Sprintf("ws://%s/%s", mctx.Host, path)
		} else {
			switch t := sctx.NetConn.(type) {
			case *net.TCPConn:
				url = fmt.Sprintf("ws://%s/%s", t.RemoteAddr().String(), path)
			default:
				url = fmt.Sprintf("ws://foo.bar/%s", path)
			}
		}

		wcon, err := w.Select(sctx.NetConn, url)
		if err != nil {
			return err
		}
		sctx.NetConn = wcon
		return nil

	case match.S_Server:
		if mctx.HTTPMux == nil {
			// help the user out if /http is missing before /ws
			HTTP{}.Apply(m, match.S_Server, ctx)
		}
		ln, err := w.Handle(mctx.HTTPMux, "/"+path)
		if err != nil {
			return err
		}
		sctx.NetListener = ln
		sctx.CloseFn = ConcatClose(ln.Close, sctx.CloseFn)
		return nil

	}

	return fmt.Errorf("incorrect side constant")
}

func (w WS) Select(netcon net.Conn, url string) (*websocket.Conn, error) {
	conf, err := websocket.NewConfig(url, url)
	if err != nil {
		return nil, err
	}

	wcon, err := websocket.NewClient(conf, netcon)
	if err != nil {
		return nil, err
	}

	return wcon, nil
}

func (w WS) Handle(mux *match.ServeMux, pattern string) (net.Listener, error) {

	closeCh := make(chan struct{})
	ln := &wslistener{
		make(chan net.Conn),
		closeCh,
	}

	var err error
	func() {
		defer recoverToError(&err, nil)
		mux.Handle(pattern, ln)
	}()
	if err != nil {
		return nil, err
	}

	go func() {
		<-closeCh
		mux.DeHandle(pattern)
	}()

	return ln, nil
}

type wslistener struct {
	acceptCh chan net.Conn
	closeCh  chan struct{}
}

func (ln wslistener) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	websocket.Handler(func(wcon *websocket.Conn) {

		// It appears we mustn't pass wcon to external users as is.
		// We'll pass a pipe instead, because the only way to know if a wcon
		// was closed remotely is to read from it until EOF.
		//
		// See below for why we need to know when wcon is closed remotely.

		ch := make(chan struct{})
		p1, p2 := net.Pipe()

		go func() {
			io.Copy(wcon, p1)
			wcon.Close()
		}()
		go func() {
			io.Copy(p1, wcon)
			p1.Close()

			close(ch)
		}()

		select {
		case ln.acceptCh <- p2:
		case <-ln.closeCh:
		}

		// As soon as we return from this function, websocket library will
		// close wcon. So we'll wait until p2 or wcon is closed.
		<-ch

	}).ServeHTTP(w, r)
}

func (ln wslistener) Accept() (net.Conn, error) {
	select {
	case c := <-ln.acceptCh:
		return c, nil
	case <-ln.closeCh:
		return nil, errors.New("listener is closed")
	}
}

func (ln wslistener) Close() error {
	var err error
	func() {
		defer recoverToError(
			&err,
			fmt.Errorf("listener is already closed"),
		)
		close(ln.closeCh)
	}()
	return err
}

func (ln wslistener) Addr() net.Addr { return nil }

func ConcatClose(f1, f2 func() error) func() error {
	return func() error {
		err := f1()
		err = f2()
		return err
	}
}

func recoverToError(maybeErr *error, err error) {
	if r := recover(); r != nil {
		if err != nil {
			*maybeErr = err
		} else {
			*maybeErr = fmt.Errorf("%s", r)
		}
	}
}
