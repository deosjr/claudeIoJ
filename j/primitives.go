package main

import "math"

// primitives maps J spelling to Verb.
var primitives map[string]*Verb

func init() {
	primitives = map[string]*Verb{
		"+":  verbPlus,
		"-":  verbMinus,
		"*":  verbStar,
		"%":  verbPercent,
		"#":  verbHash,
		"$":  verbDollar,
		",":  verbComma,
		"<":  verbLt,
		">":  verbGt,
		"[":  verbLBracket,
		"]":  verbRBracket,
		"i.": verbIota,
	}
}

// verbIota is the built-in i. (integer iota / index generator).
// Monad: scalar n -> 0..n-1; vector shape -> array of that shape filled 0..product-1.
var verbIota = &Verb{
	name:      "i.",
	monadRank: MaxRank,
	lRank:     MaxRank,
	rRank:     MaxRank,
	monad: func(w *Array) *Array {
		if w.rank() == 0 {
			n := int(atomI(w))
			out := make([]int64, n)
			for i := range n {
				out[i] = int64(i)
			}
			return vec(out)
		}
		shape := toIntSlice(w)
		n := product(shape)
		out := make([]int64, n)
		for i := range n {
			out[i] = int64(i)
		}
		return &Array{shape: shape, data: out}
	},
	dyad: func(a, w *Array) *Array {
		panic("i. dyad not implemented")
	},
}

// --- [ left identity / ] right identity ---

var verbLBracket = &Verb{
	name:      "[",
	monadRank: MaxRank,
	lRank:     MaxRank,
	rRank:     MaxRank,
	monad: func(w *Array) *Array { return w },
	dyad:  func(a, w *Array) *Array { return a },
}

var verbRBracket = &Verb{
	name:      "]",
	monadRank: MaxRank,
	lRank:     MaxRank,
	rRank:     MaxRank,
	monad: func(w *Array) *Array { return w },
	dyad:  func(a, w *Array) *Array { return w },
}

// --- + identity / add ---

var verbPlus = &Verb{
	name:      "+",
	monadRank: 0,
	lRank:     0,
	rRank:     0,
	monad: func(w *Array) *Array {
		// identity
		return w
	},
	dyad: func(a, w *Array) *Array {
		if isFloat(a) || isFloat(w) {
			return scalarF(atomF(a) + atomF(w))
		}
		return scalar(atomI(a) + atomI(w))
	},
}

// --- - negate / subtract ---

var verbMinus = &Verb{
	name:      "-",
	monadRank: 0,
	lRank:     0,
	rRank:     0,
	monad: func(w *Array) *Array {
		if isFloat(w) {
			return scalarF(-atomFloat64(w))
		}
		return scalar(-atomInt64(w))
	},
	dyad: func(a, w *Array) *Array {
		if isFloat(a) || isFloat(w) {
			return scalarF(atomF(a) - atomF(w))
		}
		return scalar(atomI(a) - atomI(w))
	},
}

// --- * signum / multiply ---

var verbStar = &Verb{
	name:      "*",
	monadRank: 0,
	lRank:     0,
	rRank:     0,
	monad: func(w *Array) *Array {
		if isFloat(w) {
			v := atomFloat64(w)
			switch {
			case v > 0:
				return scalar(1)
			case v < 0:
				return scalar(-1)
			default:
				return scalar(0)
			}
		}
		v := atomInt64(w)
		switch {
		case v > 0:
			return scalar(1)
		case v < 0:
			return scalar(-1)
		default:
			return scalar(0)
		}
	},
	dyad: func(a, w *Array) *Array {
		if isFloat(a) || isFloat(w) {
			return scalarF(atomF(a) * atomF(w))
		}
		return scalar(atomI(a) * atomI(w))
	},
}

// --- % reciprocal / divide ---

var verbPercent = &Verb{
	name:      "%",
	monadRank: 0,
	lRank:     0,
	rRank:     0,
	monad: func(w *Array) *Array {
		return scalarF(1.0 / atomF(w))
	},
	dyad: func(a, w *Array) *Array {
		return scalarF(atomF(a) / atomF(w))
	},
}

// --- # tally / reshape ---

var verbHash = &Verb{
	name:      "#",
	monadRank: MaxRank,
	lRank:     MaxRank,
	rRank:     MaxRank,
	monad: func(w *Array) *Array {
		if w.rank() == 0 {
			return scalar(1)
		}
		return scalar(int64(w.shape[0]))
	},
	dyad: func(a, w *Array) *Array {
		// reshape: a gives new shape, w gives fill data
		newShape := toIntSlice(a)
		n := product(newShape)
		wn := w.n()
		if wn == 0 {
			panic("reshape: empty fill")
		}
		switch d := w.data.(type) {
		case []int64:
			flat := make([]int64, n)
			for i := range n {
				flat[i] = d[i%wn]
			}
			return &Array{shape: newShape, data: flat}
		case []float64:
			flat := make([]float64, n)
			for i := range n {
				flat[i] = d[i%wn]
			}
			return &Array{shape: newShape, data: flat}
		case []*Array:
			flat := make([]*Array, n)
			for i := range n {
				flat[i] = d[i%wn]
			}
			return &Array{shape: newShape, data: flat}
		}
		panic("reshape: unsupported type")
	},
}

// --- $ shape / reshape ---

var verbDollar = &Verb{
	name:      "$",
	monadRank: MaxRank,
	lRank:     MaxRank,
	rRank:     MaxRank,
	monad: func(w *Array) *Array {
		if w.rank() == 0 {
			return vec([]int64{})
		}
		out := make([]int64, len(w.shape))
		for i, v := range w.shape {
			out[i] = int64(v)
		}
		return vec(out)
	},
	dyad: func(a, w *Array) *Array {
		// same as # dyad: a gives shape
		return verbHash.dyad(a, w)
	},
}

// --- , ravel / append ---

var verbComma = &Verb{
	name:      ",",
	monadRank: MaxRank,
	lRank:     MaxRank,
	rRank:     MaxRank,
	monad: func(w *Array) *Array {
		n := w.n()
		switch d := w.data.(type) {
		case []int64:
			flat := make([]int64, n)
			copy(flat, d)
			return &Array{shape: []int{n}, data: flat}
		case []float64:
			flat := make([]float64, n)
			copy(flat, d)
			return &Array{shape: []int{n}, data: flat}
		case []bool:
			flat := make([]bool, n)
			copy(flat, d)
			return &Array{shape: []int{n}, data: flat}
		case []*Array:
			flat := make([]*Array, n)
			copy(flat, d)
			return &Array{shape: []int{n}, data: flat}
		}
		panic("ravel: unsupported type")
	},
	dyad: func(a, w *Array) *Array {
		// append: concatenate along first axis
		// promote to float if needed
		useFloat := isFloat(a) || isFloat(w)
		an, wn := a.n(), w.n()
		if useFloat {
			af := toFloat64Slice(a)
			wf := toFloat64Slice(w)
			flat := make([]float64, an+wn)
			copy(flat, af)
			copy(flat[an:], wf)
			newShape := appendShape(a.shape, w.shape)
			return &Array{shape: newShape, data: flat}
		}
		ai := toInt64Slice(a)
		wi := toInt64Slice(w)
		flat := make([]int64, an+wn)
		copy(flat, ai)
		copy(flat[an:], wi)
		newShape := appendShape(a.shape, w.shape)
		return &Array{shape: newShape, data: flat}
	},
}

// appendShape computes the result shape for , dyad.
func appendShape(as, ws []int) []int {
	switch {
	case len(as) == 0 && len(ws) == 0:
		return []int{2}
	case len(as) == 0:
		out := make([]int, len(ws))
		copy(out, ws)
		out[0]++
		return out
	case len(ws) == 0:
		out := make([]int, len(as))
		copy(out, as)
		out[0]++
		return out
	default:
		out := make([]int, len(as))
		copy(out, as)
		out[0] += ws[0]
		return out
	}
}

// --- < box / less-than ---

var verbLt = &Verb{
	name:      "<",
	monadRank: MaxRank,
	lRank:     0,
	rRank:     0,
	monad: func(w *Array) *Array {
		return scalarBox(w)
	},
	dyad: func(a, w *Array) *Array {
		if isFloat(a) || isFloat(w) {
			v := atomF(a) < atomF(w)
			return scalarB(v)
		}
		v := atomI(a) < atomI(w)
		return scalarB(v)
	},
}

// --- > unbox / greater-than ---

var verbGt = &Verb{
	name:      ">",
	monadRank: 0,
	lRank:     0,
	rRank:     0,
	monad: func(w *Array) *Array {
		boxes := w.data.([]*Array)
		return boxes[0]
	},
	dyad: func(a, w *Array) *Array {
		if isFloat(a) || isFloat(w) {
			v := atomF(a) > atomF(w)
			return scalarB(v)
		}
		v := atomI(a) > atomI(w)
		return scalarB(v)
	},
}

// --- insert adverb / (fold) ---

// insertAdverb wraps a dyadic verb into a monadic verb that inserts
// (folds) the verb between elements of a vector (v/ w).
func insertAdverb(v *Verb) *Verb {
	return &Verb{
		name:      v.name + "/",
		monadRank: MaxRank,
		monad: func(w *Array) *Array {
			if w.rank() == 0 {
				return w
			}
			n := w.shape[0]
			if n == 0 {
				panic("insert: empty array")
			}
			cellShape := w.cellShape(w.rank() - 1)
			acc := w.cell(n-1, cellShape)
			for i := n - 2; i >= 0; i-- {
				acc = applyDyad(v, w.cell(i, cellShape), acc)
			}
			return acc
		},
	}
}

// --- helpers ---

// atomI returns the scalar integer value of a rank-0 array.
func atomI(a *Array) int64 {
	switch d := a.data.(type) {
	case []int64:
		return d[0]
	case []float64:
		return int64(d[0])
	case []bool:
		if d[0] {
			return 1
		}
		return 0
	}
	panic("atomI: not a numeric scalar")
}

// atomF returns the scalar float value of a rank-0 array.
func atomF(a *Array) float64 {
	switch d := a.data.(type) {
	case []float64:
		return d[0]
	case []int64:
		return float64(d[0])
	case []bool:
		if d[0] {
			return 1
		}
		return 0
	}
	panic("atomF: not a numeric scalar")
}

// toIntSlice converts an integer array to []int (for use as a shape).
func toIntSlice(a *Array) []int {
	switch d := a.data.(type) {
	case []int64:
		out := make([]int, len(d))
		for i, v := range d {
			out[i] = int(v)
		}
		return out
	case []float64:
		out := make([]int, len(d))
		for i, v := range d {
			out[i] = int(math.Round(v))
		}
		return out
	}
	panic("toIntSlice: not integer data")
}
