package main

import "testing"

// The top performance issue: applyMonad allocates two *Array values per
// element when applying a rank-0 verb to a large array.
//
// For each of the frameSize iterations:
//
//   w.cell(i, nil)  →  &Array{data: []int64{x}}   -- one *Array + one []int64
//   v.monad(cell)   →  &Array{data: []int64{-x}}  -- one *Array + one []int64
//
// BenchmarkNegateMatrix and BenchmarkNegateBaseline show the gap.
// BenchmarkSumVector and BenchmarkSumBaseline show the same problem in
// the insert adverb's accumulation loop.

// BenchmarkNegateMatrix applies monadic - to a 1000×1000 integer matrix.
// Expected: ~4 000 000 allocs/op (two per element: cell + result).
func BenchmarkNegateMatrix(b *testing.B) {
	m := matrix([]int{1000, 1000}, benchIota(1_000_000))
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		applyMonad(verbMinus, m)
	}
}

// BenchmarkNegateBaseline shows what negating a million elements actually
// needs: one output allocation and a tight loop.
// Expected: 1–2 allocs/op.
func BenchmarkNegateBaseline(b *testing.B) {
	src := benchIota(1_000_000)
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		out := make([]int64, len(src))
		for i, x := range src {
			out[i] = -x
		}
		_ = &Array{shape: []int{1000, 1000}, data: out}
	}
}

// BenchmarkSumVector applies +/ to a 1 000 000-element vector.
// insertAdverb's fold loop calls applyDyad once per element; each call
// allocates a new *Array for the accumulator.
// Expected: ~1 000 000 allocs/op.
func BenchmarkSumVector(b *testing.B) {
	v := applyMonad(verbIota, scalar(1_000_000))
	fold := insertAdverb(verbPlus)
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		applyMonad(fold, v)
	}
}

// BenchmarkSumBaseline shows what summing a million int64s actually needs.
// Expected: 1 alloc/op.
func BenchmarkSumBaseline(b *testing.B) {
	data := benchIota(1_000_000)
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		var acc int64
		for _, x := range data {
			acc += x
		}
		_ = scalar(acc)
	}
}

func benchIota(n int) []int64 {
	out := make([]int64, n)
	for i := range n {
		out[i] = int64(i)
	}
	return out
}
