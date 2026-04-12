package main

// MaxRank is the sentinel meaning "consume the entire argument".
const MaxRank = 9999

// Verb holds a J primitive (or derived) verb with its rank triple and
// monadic/dyadic implementations.
type Verb struct {
	name      string
	monadRank int
	lRank     int
	rRank     int
	monad     func(w *Array) *Array
	dyad      func(a, w *Array) *Array
}

// applyMonad applies v monadically to w, looping over frame cells as
// determined by v.monadRank.
func applyMonad(v *Verb, w *Array) *Array {
	vrank := v.monadRank
	if vrank >= w.rank() {
		return v.monad(w)
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

// applyDyad applies v dyadically to a (left) and w (right), looping over
// frame cells as determined by v.lRank and v.rRank.
func applyDyad(v *Verb, a, w *Array) *Array {
	lr, rr := v.lRank, v.rRank

	lFrame := frameOf(a, lr)
	rFrame := frameOf(w, rr)
	lCell := cellOf(a, lr)
	rCell := cellOf(w, rr)

	lSize := product(lFrame)
	rSize := product(rFrame)

	// frames must agree or one must be length 1 (scalar extension)
	var frameShape []int
	switch {
	case len(lFrame) == 0 && len(rFrame) == 0:
		// both scalars at frame level
		return v.dyad(a.cell(0, lCell), w.cell(0, rCell))
	case len(lFrame) == 0:
		// extend left across right frame
		frameShape = rFrame
		results := make([]*Array, rSize)
		for i := range rSize {
			results[i] = v.dyad(a, w.cell(i, rCell))
		}
		return assemble(results, frameShape)
	case len(rFrame) == 0:
		// extend right across left frame
		frameShape = lFrame
		results := make([]*Array, lSize)
		for i := range lSize {
			results[i] = v.dyad(a.cell(i, lCell), w)
		}
		return assemble(results, frameShape)
	default:
		// frames must agree
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

// withRank returns a copy of v with the rank triple overridden.
// This is the implementation of the J rank conjunction ".
func withRank(v *Verb, m, l, r int) *Verb {
	return &Verb{
		name:      v.name + `"`,
		monadRank: m,
		lRank:     l,
		rRank:     r,
		monad:     v.monad,
		dyad:      v.dyad,
	}
}
