package match

import (
	ma "github.com/jbenet/go-multiaddr"
	"net"
)

type Matcher interface {
	// Given a prefix-truncated multiaddr and side (S_Client or S_Server),
	// Match returns how many protocols (as in m.Protocols()) it would handle.
	// Or returns ok == false, if it can't.
	Match(m ma.Multiaddr, side int) (n int, ok bool)
}

// Protocol handlers like IP, TCP, WS implement this interface.
type MatchApplier interface {
	Matcher

	// Apply advances the connection by mutating the context.
	// It should only be called if Match returned ok with the same m and side.
	Apply(m ma.Multiaddr, side int, ctx Context) error
}

// Passed as an arg "side" to MatchAppliers.
// Specifies whether we're dialing or listening.
const (
	S_Client = iota
	S_Server
)

// Context is applied to MatchAppliers in a sequence from left to right.
type Context interface {

	// Map is not used by implementations in this package, but is meant to
	// help you experiment with new protocol implementations and whatever they
	// need to pass between each other without the need (hopefully) to modify
	// this library, at least until it's ready to merge.
	Map() map[string]interface{}

	// Holds useful info and objects. See struct definitions below.
	Misc() *MiscContext
	Special() *SpecialContext

	// CopyTo shallow-copies contents to another context. Called by Reuse.
	CopyTo(Context)

	// A MatchApplier can offer its current context to be reused by another
	// Listen() call later by invoking this function.
	// E.g. /http does this to share a single ServeMux with several /ws listeners
	//
	// Specifically, Reuse copies and stores ctx, then applies it to
	// multiaddresses that are matched by the given mch.
	Reuse(mch Matcher)
}

// MiscContext holds things produced by some MatchAppliers and required by others
type MiscContext struct {
	IPs     []net.IP
	Host    string
	HTTPMux *ServeMux
}

// SpecialContext holds values that are used or written outside MatchApplier
// objects by the library in between and after Apply() method calls.
type SpecialContext struct {

	// Dial() embedds NetConn into its returned Conn
	NetConn net.Conn

	// Listen() embedds NetListener into its returned Listener
	NetListener net.Listener

	// chain[i] MatchApplier will find PreAddr to hold the left part of the full
	// Multiaddr, which has already been executed by the chain[:i] MatchAppliers
	//
	// E.g. if the full Multiaddr is /ip4/127.0.0.1/tcp/80/http/ws,
	// during "http" Apply() execution PreAddr will be /ip4/127.0.0.1/tcp/80
	PreAddr ma.Multiaddr

	// CloseFn overrides the Close function of embedded NetConn and NetListener.
	//
	// If a MatchApplier needs to override this function, it should take the
	// responsibility of calling the one written by a previous MatchApplier
	// in the chain.
	CloseFn func() error
}
