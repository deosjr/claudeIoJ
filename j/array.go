package main

// Array is the fundamental J data type: a ranked, typed, flat array.
//
// shape == nil or len(shape)==0  =>  scalar (rank 0), exactly 1 atom
// data is always a flat slice in row-major order:
//   []int64 | []float64 | []bool | []*Array  (boxed)
type Array struct {
	shape []int
	data  any
}

func (a *Array) rank() int { return len(a.shape) }

// n returns the total number of atoms.
func (a *Array) n() int {
	if len(a.shape) == 0 {
		return 1
	}
	return product(a.shape)
}

// cellShape returns the last r dimensions (the shape of one cell at rank r).
func (a *Array) cellShape(r int) []int {
	if r <= 0 {
		return nil
	}
	start := len(a.shape) - r
	if start < 0 {
		start = 0
	}
	return a.shape[start:]
}

// frameShape returns the leading dimensions not consumed by rank r.
func (a *Array) frameShape(r int) []int {
	if r <= 0 {
		return a.shape
	}
	end := len(a.shape) - r
	if end <= 0 {
		return nil
	}
	return a.shape[:end]
}

// cell extracts the i-th frame cell (a sub-array of shape cellShape).
func (a *Array) cell(i int, cs []int) *Array {
	if len(cs) == 0 {
		// scalar cell
		switch d := a.data.(type) {
		case []int64:
			return scalar(d[i])
		case []float64:
			return scalarF(d[i])
		case []bool:
			return scalarB(d[i])
		case []*Array:
			return scalarBox(d[i])
		}
	}
	size := product(cs)
	switch d := a.data.(type) {
	case []int64:
		return &Array{shape: cs, data: d[i*size : (i+1)*size]}
	case []float64:
		return &Array{shape: cs, data: d[i*size : (i+1)*size]}
	case []bool:
		return &Array{shape: cs, data: d[i*size : (i+1)*size]}
	case []*Array:
		return &Array{shape: cs, data: d[i*size : (i+1)*size]}
	}
	panic("cell: unknown data type")
}

// assemble collects same-shaped results into one array.
// If shapes differ, each result is boxed.
func assemble(results []*Array, frameShape []int) *Array {
	if len(results) == 0 {
		return &Array{shape: frameShape, data: []int64{}}
	}
	// check all shapes equal
	first := results[0]
	allSame := true
	for _, r := range results[1:] {
		if !shapeEqual(r.shape, first.shape) {
			allSame = false
			break
		}
	}
	if !allSame {
		boxed := make([]*Array, len(results))
		copy(boxed, results)
		outShape := append(frameShape, 1) // each cell is a scalar box
		_ = outShape
		return &Array{shape: frameShape, data: boxed}
	}
	outShape := make([]int, 0, len(frameShape)+len(first.shape))
	outShape = append(outShape, frameShape...)
	outShape = append(outShape, first.shape...)

	switch first.data.(type) {
	case []float64:
		flat := make([]float64, 0, len(results)*first.n())
		for _, r := range results {
			flat = append(flat, toFloat64Slice(r)...)
		}
		return &Array{shape: outShape, data: flat}
	case []int64:
		flat := make([]int64, 0, len(results)*first.n())
		for _, r := range results {
			flat = append(flat, toInt64Slice(r)...)
		}
		return &Array{shape: outShape, data: flat}
	case []bool:
		flat := make([]bool, 0, len(results)*first.n())
		for _, r := range results {
			flat = append(flat, toBoolSlice(r)...)
		}
		return &Array{shape: outShape, data: flat}
	case []*Array:
		flat := make([]*Array, 0, len(results)*first.n())
		for _, r := range results {
			flat = append(flat, toBoxSlice(r)...)
		}
		return &Array{shape: outShape, data: flat}
	}
	// scalar int64 assembled
	flat := make([]int64, 0, len(results))
	for _, r := range results {
		flat = append(flat, toInt64Slice(r)...)
	}
	return &Array{shape: outShape, data: flat}
}

// --- constructors ---

func scalar(v int64) *Array {
	return &Array{shape: nil, data: []int64{v}}
}

func scalarF(v float64) *Array {
	return &Array{shape: nil, data: []float64{v}}
}

func scalarB(v bool) *Array {
	return &Array{shape: nil, data: []bool{v}}
}

// scalarBox wraps an array in a rank-0 box.
func scalarBox(a *Array) *Array {
	return &Array{shape: nil, data: []*Array{a}}
}

func vec(vals []int64) *Array {
	cp := make([]int64, len(vals))
	copy(cp, vals)
	return &Array{shape: []int{len(vals)}, data: cp}
}

func vecF(vals []float64) *Array {
	cp := make([]float64, len(vals))
	copy(cp, vals)
	return &Array{shape: []int{len(vals)}, data: cp}
}

func matrix(shape []int, vals []int64) *Array {
	cp := make([]int64, len(vals))
	copy(cp, vals)
	s := make([]int, len(shape))
	copy(s, shape)
	return &Array{shape: s, data: cp}
}

// --- helpers ---

func product(dims []int) int {
	p := 1
	for _, d := range dims {
		p *= d
	}
	return p
}

func shapeEqual(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func toInt64Slice(a *Array) []int64 {
	switch d := a.data.(type) {
	case []int64:
		return d
	case []float64:
		out := make([]int64, len(d))
		for i, v := range d {
			out[i] = int64(v)
		}
		return out
	case []bool:
		out := make([]int64, len(d))
		for i, v := range d {
			if v {
				out[i] = 1
			}
		}
		return out
	}
	panic("toInt64Slice: unsupported type")
}

func toFloat64Slice(a *Array) []float64 {
	switch d := a.data.(type) {
	case []float64:
		return d
	case []int64:
		out := make([]float64, len(d))
		for i, v := range d {
			out[i] = float64(v)
		}
		return out
	case []bool:
		out := make([]float64, len(d))
		for i, v := range d {
			if v {
				out[i] = 1
			}
		}
		return out
	}
	panic("toFloat64Slice: unsupported type")
}

func toBoolSlice(a *Array) []bool {
	switch d := a.data.(type) {
	case []bool:
		return d
	case []int64:
		out := make([]bool, len(d))
		for i, v := range d {
			out[i] = v != 0
		}
		return out
	}
	panic("toBoolSlice: unsupported type")
}

func toBoxSlice(a *Array) []*Array {
	switch d := a.data.(type) {
	case []*Array:
		return d
	}
	panic("toBoxSlice: not a boxed array")
}

// isFloat reports whether the array holds float data.
func isFloat(a *Array) bool {
	_, ok := a.data.([]float64)
	return ok
}

// isBox reports whether the array holds boxed data.
func isBox(a *Array) bool {
	_, ok := a.data.([]*Array)
	return ok
}

// atomInt64 returns the single integer value of a scalar int64 array.
func atomInt64(a *Array) int64 {
	return a.data.([]int64)[0]
}

// atomFloat64 returns the single float value of a scalar float64 array.
func atomFloat64(a *Array) float64 {
	return a.data.([]float64)[0]
}
