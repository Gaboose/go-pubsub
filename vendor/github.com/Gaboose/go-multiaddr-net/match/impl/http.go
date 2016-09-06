package impl

import (
	"github.com/Gaboose/go-multiaddr-net/match"
	"net"
	"net/http"

	ma "github.com/jbenet/go-multiaddr"
)

type HTTP struct{}

func (p HTTP) Match(m ma.Multiaddr, side int) (int, bool) {
	if side != match.S_Server {
		return 0, false
	}

	ms := m.Protocols()
	if len(ms) >= 1 && ms[0].Name == "http" {
		return 1, true
	}

	return 0, false
}

func (p HTTP) Apply(m ma.Multiaddr, side int, ctx match.Context) error {
	mctx := ctx.Misc()
	sctx := ctx.Special()

	mctx.HTTPMux = p.Server(sctx.NetListener)

	ctx.Reuse(&httpreuser{ctx.Special().PreAddr})
	return nil
}

func (p HTTP) Server(ln net.Listener) *match.ServeMux {
	mux := match.NewServeMux()
	srv := &http.Server{Handler: mux}
	go srv.Serve(ln)
	return mux
}

type httpreuser struct {
	prefix ma.Multiaddr
}

func (h httpreuser) Match(m ma.Multiaddr, side int) (int, bool) {
	if side != match.S_Server {
		return 0, false
	}

	// check if h.prefix is a prefix of m
	ms := ma.Split(m)
	ps := ma.Split(h.prefix)

	if len(ms) < len(ps) {
		return 0, false
	}

	for i, p := range ps {
		if !p.Equal(ms[i]) {
			return 0, false
		}
	}

	// match an additional http protocol if it's there
	if len(ms) > len(ps) && ms[len(ps)].String() == "/http" {
		return len(ps) + 1, true
	}

	return len(ps), true
}
