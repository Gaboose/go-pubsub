package impl

import (
	"fmt"
	"github.com/Gaboose/go-multiaddr-net/match"
	ma "github.com/jbenet/go-multiaddr"
	"net"
)

func init() {
	ma.AddProtocol(ma.Protocol{42, -1, "dns", ma.CodeToVarint(42)})
}

type DNS struct{}

func (_ DNS) Match(m ma.Multiaddr, side int) (int, bool) {
	ps := m.Protocols()

	if len(ps) > 0 && ps[0].Name == "dns" {
		return 1, true
	}

	return 0, false
}

func (_ DNS) Apply(m ma.Multiaddr, side int, ctx match.Context) error {
	p := m.Protocols()[0]
	host, _ := m.ValueForProtocol(p.Code)

	ips, err := net.LookupIP(host)
	if err != nil {
		return err
	}

	if len(ips) == 0 {
		return fmt.Errorf("failed to resolve domain")
	}

	mctx := ctx.Misc()
	mctx.Host = host
	mctx.IPs = ips

	return nil
}
