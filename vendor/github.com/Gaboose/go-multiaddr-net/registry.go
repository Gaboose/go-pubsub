package manet

import (
	"fmt"
	"sync"

	"github.com/Gaboose/go-multiaddr-net/match"
	"github.com/Gaboose/go-multiaddr-net/match/impl"
	ma "github.com/jbenet/go-multiaddr"
)

// matchreg is a holds MatchAppliers used by Dial and Listen functions
type matchreg struct {
	sync.Mutex

	// standard MatchAppliers for both dialing and listening
	protocols []match.MatchApplier

	// running listeners available for reuse (e.g. /http with a ServeMux)
	reusable []match.MatchApplier
}

var matchers = &matchreg{
	protocols: []match.MatchApplier{
		impl.IP{},
		impl.DNS{},
		impl.TCP{},
		impl.HTTP{},
		impl.WS{},
	},
}

type reusableContext struct {
	match.Matcher
	match.Context
	underClose func() error
	usecount   int
}

func (rc *reusableContext) Apply(m ma.Multiaddr, side int, ctx match.Context) error {
	rc.Context.CopyTo(ctx)
	ctx.Special().CloseFn = rc.Close
	rc.usecount++
	return nil
}

func (rc *reusableContext) Close() error {
	matchers.Lock()
	defer matchers.Unlock()

	rc.usecount--
	if rc.usecount == 0 {

		// remove rc from matchers.reusable
		mr := matchers.reusable
		for i, mch := range mr {
			if mch == rc {
				mr[i] = mr[len(mr)-1] // override with the last element
				mr[len(mr)-1] = nil   // remove duplicate ref
				mr = mr[:len(mr)-1]   // decrease length by one
				break
			}
		}
		matchers.reusable = mr

		return rc.underClose()
	}
	return nil
}

// buildChain returns two parralel slices. One holds a sequence of MatchAppliers,
// which is capable to handle the given Multiaddr with the side constant.
// The second: full Multiaddr split into one or more protocols a piece, which
// is what each MatchApplier.Apply expects as its m argument.
//
// Allowed values for side are match.S_Server and match.S_Client.
func (mr matchreg) buildChain(m ma.Multiaddr, side int) ([]match.MatchApplier, []ma.Multiaddr, error) {
	tail := m
	chain := []match.MatchApplier{}
	split := []ma.Multiaddr{}

	for tail.String() != "" {
		mch, n, err := mr.matchPrefix(tail, side)
		if err != nil {
			return nil, nil, err
		}

		spl := ma.Split(tail)
		head := ma.Join(spl[:n]...)
		tail = ma.Join(spl[n:]...)

		split = append(split, head)
		chain = append(chain, mch)
	}

	return chain, split, nil
}

// matchPrefix finds a MatchApplier (in mr.protocols or mr.reusable) for one or
// more first m.Protocols() and also returns an int of how many protocols it can
// consume.
//
// matchPrefix returns an error if it can't find any or finds more than one
// MatchApplier
func (mr matchreg) matchPrefix(m ma.Multiaddr, side int) (match.MatchApplier, int, error) {
	ret := []match.MatchApplier{}

	for _, mch := range mr.reusable {
		if _, ok := mch.Match(m, side); ok {
			ret = append(ret, mch)
		}
	}

	if len(ret) == 1 {
		n, _ := ret[0].Match(m, side)
		return ret[0], n, nil
	} else if len(ret) > 1 {
		return nil, 0, fmt.Errorf("found more than one reusable for %s", m.String())
	}

	for _, mch := range mr.protocols {
		if _, ok := mch.Match(m, side); ok {
			ret = append(ret, mch)
		}
	}

	if len(ret) == 0 {
		return nil, 0, fmt.Errorf("no matchers found for %s", m.String())
	} else if len(ret) > 1 {
		return nil, 0, fmt.Errorf("found more than one matcher for %s", m.String())
	}

	n, _ := ret[0].Match(m, side)
	return ret[0], n, nil
}
