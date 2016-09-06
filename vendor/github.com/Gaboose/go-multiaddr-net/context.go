package manet

import (
	"github.com/Gaboose/go-multiaddr-net/match"
	"reflect"
)

type context struct {
	m       map[string]interface{}
	misc    match.MiscContext
	special match.SpecialContext
}

func NewContext() *context {
	return &context{m: map[string]interface{}{}}
}

func (ctx *context) Map() map[string]interface{}    { return ctx.m }
func (ctx *context) Misc() *match.MiscContext       { return &ctx.misc }
func (ctx *context) Special() *match.SpecialContext { return &ctx.special }

func (ctx context) CopyTo(target match.Context) {
	// shallow copy
	*target.Misc() = ctx.misc
	*target.Special() = ctx.special

	trg := target.Map()

	for k, val := range ctx.Map() {
		// If we want to allow pointers to structs in the map, we have to copy
		// one level deeper, i.e. instead of copying the pointer, reflect and
		// copy what it points to.
		rval := reflect.ValueOf(val)
		if rval.Kind() == reflect.Ptr {

			rv := reflect.New(rval.Type().Elem())
			rv.Elem().Set(rval.Elem())
			trg[k] = rv.Interface()

		} else {
			trg[k] = val
		}
	}
}

func (ctx context) Reuse(mch match.Matcher) {
	// a snapshot of current context to be reused
	ctxcopy := NewContext()
	ctx.CopyTo(ctxcopy)

	// replace sctx.CloseFn with one that manages rc.usecount - the number of
	// Listener instances rc serves
	sctx := ctx.Special()
	rc := &reusableContext{mch, ctxcopy, sctx.CloseFn, 1}
	sctx.CloseFn = rc.Close

	matchers.reusable = append(matchers.reusable, rc)
}
