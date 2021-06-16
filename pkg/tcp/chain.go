package tcp

import (
	"fmt"
)

// Constructor A constructor for a piece of TCP middleware.
// Some TCP middleware use this constructor out of the box,
// so in most cases you can just pass somepackage.New.
type Constructor func(Handler) (Handler, error)

// Chain is a chain for TCP handlers.
// Chain acts as a list of tcp.Handler constructors.
// Chain is effectively immutable:
// once created, it will always hold
// the same set of constructors in the same order.
type Chain struct {
	constructors []Constructor
}

// NewChain creates a new TCP chain,
// memorizing the given list of TCP middleware constructors.
// New serves no other function,
// constructors are only called upon a call to Then().
func NewChain(constructors ...Constructor) Chain {
	return Chain{constructors: constructors}
}

// Then adds an handler at the end of the chain.
func (c Chain) Then(h Handler) (Handler, error) {
	if h == nil {
		return nil, fmt.Errorf("cannot add a nil handler to the chain")
	}

	for i := range c.constructors {
		handler, err := c.constructors[len(c.constructors)-1-i](h)
		if err != nil {
			return nil, err
		}
		h = handler
	}

	return h, nil
}

// Append extends a chain, adding the specified constructors
// as the last ones in the request flow.
//
// Append returns a new chain, leaving the original one untouched.
//
//     stdChain := tcp.NewChain(m1, m2)
//     extChain := stdChain.Append(m3, m4)
//     // requests in stdChain go m1 -> m2
//     // requests in extChain go m1 -> m2 -> m3 -> m4
func (c Chain) Append(constructors ...Constructor) Chain {
	newCons := make([]Constructor, 0, len(c.constructors)+len(constructors))
	newCons = append(newCons, c.constructors...)
	newCons = append(newCons, constructors...)

	return Chain{newCons}
}

// Extend extends a chain by adding the specified chain
// as the last one in the request flow.
//
// Extend returns a new chain, leaving the original one untouched.
//
//     stdChain := tcp.NewChain(m1, m2)
//     ext1Chain := tcp.NewChain(m3, m4)
//     ext2Chain := stdChain.Extend(ext1Chain)
//     // requests in stdChain go  m1 -> m2
//     // requests in ext1Chain go m3 -> m4
//     // requests in ext2Chain go m1 -> m2 -> m3 -> m4
//
// Another example:
//  aHtmlAfterNosurf := tcp.NewChain(m2)
// 	aHtml := tcp.NewChain(m1, func(h tcp.Handler) tcp.Handler {
// 		csrf := nosurf.New(h)
// 		csrf.SetFailureHandler(aHtmlAfterNosurf.ThenFunc(csrfFail))
// 		return csrf
// 	}).Extend(aHtmlAfterNosurf)
//		// requests to aHtml hitting nosurfs success handler go m1 -> nosurf -> m2 -> target-handler
//		// requests to aHtml hitting nosurfs failure handler go m1 -> nosurf -> m2 -> csrfFail
func (c Chain) Extend(chain Chain) Chain {
	return c.Append(chain.constructors...)
}
