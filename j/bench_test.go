package main

import "testing"

// sinkSlice and sinkInt prevent the compiler from optimising away the
// baseline loops without adding interpreter overhead to the measurement.
var sinkSlice []int64
var sinkInt int64

// These benchmarks measure the cost of applying rank-0 verbs to large arrays,
// and compare the interpreter against the irreducible minimum (a typed loop).
//
// The fix — typed fast paths (monadInt, monadFloat, dyadInt, dyadFloat) on
// each verb that bypass cell extraction entirely — reduced allocation from
// millions of short-lived *Array values to a constant 3 per call:
// the input array, the output slice, and its *Array header.
//
// Results on Intel i5-6600K @ 3.50GHz:
//
//                    before           after          baseline
//   NegateMatrix   194ms  6M alloc   2.5ms  3 alloc   1.5ms  1 alloc
//   SumVector      316ms 12M alloc   1.8ms  3 alloc   0.5ms  0 alloc
//
// The remaining gap to baseline is function-call overhead and (for mixed
// int/float inputs) the toFloat64Slice conversion; it is constant, not
// proportional to array size.

// BenchmarkNegateMatrix applies monadic - to a 1000×1000 integer matrix.
// 3 allocs/op: input array + output []int64 + output *Array header.
func BenchmarkNegateMatrix(b *testing.B) {
	m := matrix([]int{1000, 1000}, benchIota(1_000_000))
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		applyMonad(verbMinus, m)
	}
}

// BenchmarkNegateBaseline shows the irreducible cost: one allocation for the
// output slice and a single tight loop with no dispatch overhead.
// 1 alloc/op: output []int64.
func BenchmarkNegateBaseline(b *testing.B) {
	src := benchIota(1_000_000)
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		out := make([]int64, len(src))
		for i, x := range src {
			out[i] = -x
		}
		sinkSlice = out
	}
}

// BenchmarkSumVector applies +/ to a 1 000 000-element vector.
// 3 allocs/op: input array + scalar result *Array + its []int64.
func BenchmarkSumVector(b *testing.B) {
	v := applyMonad(verbIota, scalar(1_000_000))
	fold := insertAdverb(verbPlus)
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		applyMonad(fold, v)
	}
}

// BenchmarkSumBaseline shows the irreducible cost: a single accumulator
// loop with no allocation.
// 0 allocs/op.
func BenchmarkSumBaseline(b *testing.B) {
	data := benchIota(1_000_000)
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		var acc int64
		for _, x := range data {
			acc += x
		}
		sinkInt = acc
	}
}

func benchIota(n int) []int64 {
	out := make([]int64, n)
	for i := range n {
		out[i] = int64(i)
	}
	return out
}
