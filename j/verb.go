package main

import "math"

// MaxRank is the sentinel meaning "consume the entire argument".
const MaxRank = math.MaxInt

// Verb holds a J primitive (or derived) verb with its rank triple and
// monadic/dyadic implementations.
//
// monadInt and monadFloat are optional rank-0 fast paths. When set,
// applyMonad bypasses cell extraction and operates directly on the flat
// slice, eliminating the two *Array allocations per element that the
// general loop requires.
//
// dyadInt and dyadFloat play the same role for applyDyad.
// When dyadInt is nil but dyadFloat is set, int inputs are promoted.
type Verb struct {
	name      string
	monadRank int
	lRank     int
	rRank     int
	monad     func(w *Array) *Array
	dyad      func(a, w *Array) *Array
	// rank-0 fast paths; nil means fall back to monad/dyad above
	monadInt   func(int64) int64
	monadFloat func(float64) float64
	dyadInt    func(int64, int64) int64
	dyadFloat  func(float64, float64) float64
}

// applyMonad applies v monadically to w, looping over frame cells as
// determined by v.monadRank.
func applyMonad(v *Verb, w *Array) *Array {
	vrank := v.monadRank
	if vrank >= w.rank() {
		return v.monad(w)
	}
	// rank-0 fast path: iterate directly over the flat data, no cell allocation
	if vrank == 0 {
		return applyMonadFlat(v, w)
	}
	cellShape := w.cellShape(vrank)
	frameShape := w.frameShape(vrank)
	frameSize := product(frameShape)

	results := make([]*Array, frameSize)
	for i := range frameSize {
		results[i] = v.monad(w.cell(i, cellShape))
	}
	return assemble(results, frameShape)
}

// applyMonadFlat is the rank-0 fast path for applyMonad.
// It requires vrank == 0 and w.rank() > 0.
func applyMonadFlat(v *Verb, w *Array) *Array {
	switch d := w.data.(type) {
	case []int64:
		if v.monadInt != nil {
			out := make([]int64, len(d))
			for i, x := range d {
				out[i] = v.monadInt(x)
			}
			return &Array{shape: w.shape, data: out}
		}
		if v.monadFloat != nil {
			out := make([]float64, len(d))
			for i, x := range d {
				out[i] = v.monadFloat(float64(x))
			}
			return &Array{shape: w.shape, data: out}
		}
	case []float64:
		if v.monadFloat != nil {
			out := make([]float64, len(d))
			for i, x := range d {
				out[i] = v.monadFloat(x)
			}
			return &Array{shape: w.shape, data: out}
		}
	}
	// fall back to general cell loop
	n := w.n()
	results := make([]*Array, n)
	for i := range n {
		results[i] = v.monad(w.cell(i, nil))
	}
	return assemble(results, w.shape)
}

// applyDyad applies v dyadically to a (left) and w (right), looping over
// frame cells as determined by v.lRank and v.rRank.
func applyDyad(v *Verb, a, w *Array) *Array {
	lr, rr := v.lRank, v.rRank

	// rank-0 fast path: operate directly on flat data, no cell allocation
	if lr == 0 && rr == 0 {
		return applyDyadFlat(v, a, w)
	}

	lFrame := frameOf(a, lr)
	rFrame := frameOf(w, rr)
	lCell := cellOf(a, lr)
	rCell := cellOf(w, rr)

	lSize := product(lFrame)
	rSize := product(rFrame)

	// frames must agree or one must be empty (scalar extension)
	var frameShape []int
	switch {
	case len(lFrame) == 0 && len(rFrame) == 0:
		return v.dyad(a.cell(0, lCell), w.cell(0, rCell))
	case len(lFrame) == 0:
		frameShape = rFrame
		results := make([]*Array, rSize)
		for i := range rSize {
			results[i] = v.dyad(a, w.cell(i, rCell))
		}
		return assemble(results, frameShape)
	case len(rFrame) == 0:
		frameShape = lFrame
		results := make([]*Array, lSize)
		for i := range lSize {
			results[i] = v.dyad(a.cell(i, lCell), w)
		}
		return assemble(results, frameShape)
	default:
		if !shapeEqual(lFrame, rFrame) {
			panic("applyDyad: frames do not agree")
		}
		frameShape = lFrame
		n := product(frameShape)
		results := make([]*Array, n)
		for i := range n {
			results[i] = v.dyad(a.cell(i, lCell), w.cell(i, rCell))
		}
		return assemble(results, frameShape)
	}
}

// applyDyadFlat is the rank-0 fast path for applyDyad.
// It requires lRank == 0 and rRank == 0.
// Handles scalar extension: if either argument is rank-0 it is broadcast.
func applyDyadFlat(v *Verb, a, w *Array) *Array {
	aScalar := a.rank() == 0
	wScalar := w.rank() == 0

	var outShape []int
	switch {
	case aScalar && wScalar:
		outShape = nil
	case aScalar:
		outShape = w.shape
	case wScalar:
		outShape = a.shape
	default:
		if !shapeEqual(a.shape, w.shape) {
			panic("applyDyad: frames do not agree")
		}
		outShape = a.shape
	}

	n := 1
	if outShape != nil {
		n = product(outShape)
	}

	useFloat := isFloat(a) || isFloat(w)

	if !useFloat && v.dyadInt != nil {
		ad := a.data.([]int64)
		wd := w.data.([]int64)
		out := make([]int64, n)
		switch {
		case aScalar && wScalar:
			out[0] = v.dyadInt(ad[0], wd[0])
		case aScalar:
			av := ad[0]
			for i, x := range wd {
				out[i] = v.dyadInt(av, x)
			}
		case wScalar:
			wv := wd[0]
			for i, x := range ad {
				out[i] = v.dyadInt(x, wv)
			}
		default:
			for i := range n {
				out[i] = v.dyadInt(ad[i], wd[i])
			}
		}
		if outShape == nil {
			return scalar(out[0])
		}
		return &Array{shape: outShape, data: out}
	}

	if v.dyadFloat != nil {
		af := toFloat64Slice(a)
		wf := toFloat64Slice(w)
		out := make([]float64, n)
		switch {
		case aScalar && wScalar:
			out[0] = v.dyadFloat(af[0], wf[0])
		case aScalar:
			av := af[0]
			for i, x := range wf {
				out[i] = v.dyadFloat(av, x)
			}
		case wScalar:
			wv := wf[0]
			for i, x := range af {
				out[i] = v.dyadFloat(x, wv)
			}
		default:
			for i := range n {
				out[i] = v.dyadFloat(af[i], wf[i])
			}
		}
		if outShape == nil {
			return scalarF(out[0])
		}
		return &Array{shape: outShape, data: out}
	}

	// fall back to general cell loop (verb has no typed fast path)
	switch {
	case aScalar && wScalar:
		return v.dyad(a, w)
	case aScalar:
		results := make([]*Array, n)
		for i := range n {
			results[i] = v.dyad(a, w.cell(i, nil))
		}
		return assemble(results, outShape)
	case wScalar:
		results := make([]*Array, n)
		for i := range n {
			results[i] = v.dyad(a.cell(i, nil), w)
		}
		return assemble(results, outShape)
	default:
		results := make([]*Array, n)
		for i := range n {
			results[i] = v.dyad(a.cell(i, nil), w.cell(i, nil))
		}
		return assemble(results, outShape)
	}
}

func frameOf(a *Array, r int) []int {
	if r >= MaxRank {
		return nil
	}
	return a.frameShape(r)
}

func cellOf(a *Array, r int) []int {
	if r >= MaxRank {
		return a.shape
	}
	return a.cellShape(r)
}

// --- trains: hooks, forks, capped forks ---

// constVerb returns a verb that ignores its argument and always returns n.
// Used as the left tine of a capped fork: (n g h) y = n g (h y).
func constVerb(n *Array) *Verb {
	return &Verb{
		name:      "const",
		monadRank: MaxRank,
		lRank:     MaxRank,
		rRank:     MaxRank,
		monad:     func(_ *Array) *Array { return n },
		dyad:      func(_, _ *Array) *Array { return n },
	}
}

// hookVerb forms a two-verb train (hook).
//
//	monad: (f g) y   = y f (g y)
//	dyad:  x (f g) y = x f (g y)
func hookVerb(f, g *Verb) *Verb {
	return &Verb{
		name:      "(" + f.name + " " + g.name + ")",
		monadRank: MaxRank,
		lRank:     MaxRank,
		rRank:     MaxRank,
		monad: func(w *Array) *Array {
			return applyDyad(f, w, applyMonad(g, w))
		},
		dyad: func(a, w *Array) *Array {
			return applyDyad(f, a, applyMonad(g, w))
		},
	}
}

// forkVerb forms a three-verb train (fork).
//
//	monad: (f g h) y   = (f y) g (h y)
//	dyad:  x (f g h) y = (x f y) g (x h y)
func forkVerb(f, g, h *Verb) *Verb {
	return &Verb{
		name:      "(" + f.name + " " + g.name + " " + h.name + ")",
		monadRank: MaxRank,
		lRank:     MaxRank,
		rRank:     MaxRank,
		monad: func(w *Array) *Array {
			return applyDyad(g, applyMonad(f, w), applyMonad(h, w))
		},
		dyad: func(a, w *Array) *Array {
			return applyDyad(g, applyDyad(f, a, w), applyDyad(h, a, w))
		},
	}
}

// foldVerbRun reduces a run of two or more verbs into a single derived verb
// by grouping from the right: odd-length runs form a fork at the left,
// even-length runs form a hook at the left.
//
//	2 verbs: hook(f, g)
//	3 verbs: fork(f, g, h)
//	4 verbs: hook(f, fork(g, h, k))
//	5 verbs: fork(f, g, fork(h, j, k))
func foldVerbRun(verbs []*Verb) *Verb {
	switch len(verbs) {
	case 1:
		return verbs[0]
	case 2:
		return hookVerb(verbs[0], verbs[1])
	case 3:
		return forkVerb(verbs[0], verbs[1], verbs[2])
	default:
		if len(verbs)%2 == 0 {
			return hookVerb(verbs[0], foldVerbRun(verbs[1:]))
		}
		return forkVerb(verbs[0], verbs[1], foldVerbRun(verbs[2:]))
	}
}

// withRank returns a copy of v with the rank triple overridden.
// This is the implementation of the J rank conjunction ".
func withRank(v *Verb, m, l, r int) *Verb {
	return &Verb{
		name:       v.name + `"`,
		monadRank:  m,
		lRank:      l,
		rRank:      r,
		monad:      v.monad,
		dyad:       v.dyad,
		monadInt:   v.monadInt,
		monadFloat: v.monadFloat,
		dyadInt:    v.dyadInt,
		dyadFloat:  v.dyadFloat,
	}
}
