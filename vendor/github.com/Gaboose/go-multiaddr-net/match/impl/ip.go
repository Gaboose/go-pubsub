package impl

import (
	"fmt"
	"github.com/Gaboose/go-multiaddr-net/match"
	ma "github.com/jbenet/go-multiaddr"
	"net"
)

type IP struct{}

func (_ IP) Match(m ma.Multiaddr, side int) (int, bool) {
	ps := m.Protocols()

	if len(ps) < 1 {
		return 0, false
	}

	name := ps[0].Name
	if name == "ip" || name == "ip4" || name == "ip6" {
		return 1, true
	}

	return 0, false
}

func (_ IP) Apply(m ma.Multiaddr, side int, ctx match.Context) error {
	p := m.Protocols()[0]
	name := p.Name
	s, _ := m.ValueForProtocol(p.Code)

	mctx := ctx.Misc()
	ip := net.ParseIP(s)
	if ip == nil {
		return fmt.Errorf("incorrect ip %s", m)
	}

	switch name {
	case "ip4":
		if ip = ip.To4(); ip == nil {
			return fmt.Errorf("incorrect ip4 %s", m)
		}
	case "ip6":
		if ip = ip.To16(); ip == nil {
			return fmt.Errorf("incorrent ip6 %s", m)
		}
	}

	mctx.IPs = append(mctx.IPs, ip)
	return nil
}

// FromIP converts a net.IP type to a Multiaddr.
func FromIP(ip net.IP) (ma.Multiaddr, error) {
	switch {
	case ip.To4() != nil:
		return ma.NewMultiaddr("/ip4/" + ip.String())
	case ip.To16() != nil:
		return ma.NewMultiaddr("/ip6/" + ip.String())
	default:
		return nil, fmt.Errorf("incorrect network addr conversion")
	}
}
