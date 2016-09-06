package manet

import (
	"fmt"
	"net"
	"strings"

	"github.com/Gaboose/go-multiaddr-net/match"
	"github.com/Gaboose/go-multiaddr-net/match/impl"
	ma "github.com/jbenet/go-multiaddr"
)

// Dial connects to a remote address
func Dial(remote ma.Multiaddr) (Conn, error) {

	matchers.Lock()
	defer matchers.Unlock()

	chain, split, err := matchers.buildChain(remote, match.S_Client)
	if err != nil {
		return nil, err
	}

	ctx := NewContext()
	sctx := ctx.Special()

	// apply context mutators
	for i, mch := range chain {

		err := mch.Apply(split[i], match.S_Client, ctx)
		if err != nil {
			if sctx.CloseFn != nil {
				sctx.CloseFn()
			}
			return nil, err
		}

		if sctx.PreAddr == nil {
			sctx.PreAddr = split[i]
		} else {
			sctx.PreAddr = sctx.PreAddr.Encapsulate(split[i])
		}

	}

	if sctx.NetConn == nil {
		if sctx.CloseFn != nil {
			sctx.CloseFn()
		}
		return nil, fmt.Errorf("insufficient address for a connection: %s", remote)
	}

	return &conn{
		Conn:    sctx.NetConn,
		raddr:   remote,
		closeFn: sctx.CloseFn,
	}, nil
}

func Listen(local ma.Multiaddr) (Listener, error) {
	return listen(local)
}

// Listen receives inbound connections on the local network address.
func listen(local ma.Multiaddr) (*listener, error) {
	matchers.Lock()
	defer matchers.Unlock()

	// resolve a chain of applicable MatchAppliers
	chain, split, err := matchers.buildChain(local, match.S_Server)
	if err != nil {
		return nil, err
	}

	ctx := NewContext()
	sctx := ctx.Special()

	// apply chain to empty context
	for i, mch := range chain {

		err := mch.Apply(split[i], match.S_Server, ctx)

		if err != nil {
			if sctx.CloseFn != nil {
				go sctx.CloseFn()
			}
			return nil, err
		}

		if sctx.PreAddr == nil {
			sctx.PreAddr = split[i]
		} else {
			sctx.PreAddr = sctx.PreAddr.Encapsulate(split[i])
		}

	}

	if sctx.NetListener == nil {
		if sctx.CloseFn != nil {
			go sctx.CloseFn()
		}
		return nil, fmt.Errorf("insufficient address for a listener: %s", local)
	}

	ln := &listener{
		Listener: sctx.NetListener,
		maddr:    local,
		closeFn:  sctx.CloseFn,
	}

	return ln, nil
}

type listener struct {
	net.Listener
	maddr   ma.Multiaddr
	closeFn func() error
}

func (l listener) Accept() (Conn, error) {
	netcon, err := l.Listener.Accept()
	if err != nil {
		return nil, fmt.Errorf("listener %s is closed", l.maddr.String())
	}

	return &conn{
		Conn:    netcon,
		laddr:   l.maddr,
		closeFn: netcon.Close,
	}, nil
}

func (l listener) Close() error {
	return l.closeFn()
}

func (l listener) Multiaddr() ma.Multiaddr { return l.maddr }

func NetListen(local ma.Multiaddr) (net.Listener, error) {
	ln, err := listen(local)
	return netlistener{ln}, err
}

type netlistener struct {
	*listener
}

func (l netlistener) Accept() (net.Conn, error) {
	return l.listener.Accept()
}

type conn struct {
	net.Conn
	laddr   ma.Multiaddr
	raddr   ma.Multiaddr
	closeFn func() error
}

func (c conn) Close() error {
	return c.closeFn()
}

func (c conn) LocalMultiaddr() ma.Multiaddr {
	if c.laddr != nil {
		return c.laddr
	}
	m, _ := FromNetAddr(c.LocalAddr())
	return m
}
func (c conn) RemoteMultiaddr() ma.Multiaddr {
	if c.raddr != nil {
		return c.raddr
	}
	m, _ := FromNetAddr(c.RemoteAddr())
	return m
}

func trimPrefix(m, prem ma.Multiaddr) (ma.Multiaddr, bool) {
	s := m.String()
	pres := prem.String()

	if !strings.HasPrefix(s, pres) {
		return nil, false
	}

	return ma.StringCast(strings.TrimPrefix(s, pres)), true
}

// FromNetAddr converts a net.Addr type to a Multiaddr.
func FromNetAddr(naddr net.Addr) (ma.Multiaddr, error) {
	if tcpaddr, ok := naddr.(*net.TCPAddr); ok {
		return FromTCPAddr(tcpaddr)
	} else {
		return nil, fmt.Errorf("unknown net.Addr")
	}
}

// FromTCPAddr converts a *net.TCPAddr type to a Multiaddr.
func FromTCPAddr(addr *net.TCPAddr) (ma.Multiaddr, error) {
	ipm, err := impl.FromIP(addr.IP)
	if err != nil {
		return nil, err
	}

	tcpm, err := ma.NewMultiaddr(fmt.Sprintf("/tcp/%d", addr.Port))
	if err != nil {
		return nil, err
	}

	return ipm.Encapsulate(tcpm), nil
}
