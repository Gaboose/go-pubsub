package impl

import (
	"fmt"
	"github.com/Gaboose/go-multiaddr-net/match"
	"net"
	"strconv"
	"strings"

	ma "github.com/jbenet/go-multiaddr"
)

type TCP struct{}

func (t TCP) Match(m ma.Multiaddr, side int) (int, bool) {
	ps := m.Protocols()

	if len(ps) < 1 {
		return 0, false
	}

	if ps[0].Name == "tcp" {
		return 1, true
	}

	return 0, false
}

func (t TCP) Apply(m ma.Multiaddr, side int, ctx match.Context) error {
	p := m.Protocols()[0]
	portstr, _ := m.ValueForProtocol(p.Code)

	port, err := strconv.Atoi(portstr)
	if err != nil {
		return err
	}

	mctx := ctx.Misc()
	sctx := ctx.Special()

	if len(mctx.IPs) == 0 {
		return fmt.Errorf("no ips in context")
	}

	switch side {

	case match.S_Client:
		var con net.Conn
		var err error
		if len(mctx.IPs) == 1 {
			con, err = t.Dial(mctx.IPs[0], port)
		} else {
			con, err = t.DialMany(mctx.IPs, port)
		}
		if err != nil {
			return err
		}

		sctx.NetConn = con
		sctx.CloseFn = con.Close
		return nil

	case match.S_Server:
		netln, err := t.Listen(mctx.IPs[0], port)
		if err != nil {
			return err
		}
		sctx.NetListener = netln
		sctx.CloseFn = netln.Close
		return nil

	}

	return fmt.Errorf("incorrect side constant")
}

// DialMany tries to connect to all ips, returns the first successful one
// and closes the others. If all fail, it returns an aggregated error.
func (t TCP) DialMany(ips []net.IP, port int) (*net.TCPConn, error) {
	firstCh := make(chan *net.TCPConn)
	doneCh := make(chan struct{})
	errCh := make(chan error)

	// launch parallel dialers
	for _, ip := range ips {
		go func(ip net.IP) {
			c, err := t.Dial(ip, port)

			if err != nil {
				select {
				case errCh <- err:
				case <-doneCh:
				}
			} else {
				select {
				case firstCh <- c:
					// DialMany will return this one
				case <-doneCh:
					// successful, but too late
					c.Close()
				}
			}
		}(ip)
	}

	var tcpcon *net.TCPConn
	errs := make([]string, 0, len(ips))
	var i int

Loop: // count errors, but break out immidiately on first successful connection
	for i < len(ips) {
		select {
		case tcpcon = <-firstCh:
			break Loop
		case err := <-errCh:
			errs = append(errs, err.Error())
			i++
		}
	}

	close(doneCh)

	if tcpcon == nil {
		return nil, fmt.Errorf(strings.Join(errs, "; "))
	}
	return tcpcon, nil
}

func (t TCP) Dial(ip net.IP, port int) (*net.TCPConn, error) {
	addr := &net.TCPAddr{IP: ip, Port: port}

	con, err := net.DialTCP("tcp", nil, addr)
	if err != nil {
		return nil, err
	}

	return con, nil
}

func (t TCP) Listen(ip net.IP, port int) (*net.TCPListener, error) {
	addr := &net.TCPAddr{IP: ip, Port: port}

	ln, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return nil, err
	}

	return ln, nil
}
