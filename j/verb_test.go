package main

import (
	"reflect"
	"testing"
)

func TestApplyMonadRank0OnVector(t *testing.T) {
	// rank-0 monad applied to a vector should apply element-wise
	result := applyMonad(verbMinus, vec([]int64{1, 2, 3}))
	if !reflect.DeepEqual(result.shape, []int{3}) {
		t.Errorf("shape: got %v want [3]", result.shape)
	}
	if !reflect.DeepEqual(result.data.([]int64), []int64{-1, -2, -3}) {
		t.Errorf("data: got %v want [-1 -2 -3]", result.data)
	}
}

func TestApplyMonadMaxRankOnMatrix(t *testing.T) {
	// MaxRank monad (tally) on a 2x3 matrix: applies to whole matrix
	m := matrix([]int{2, 3}, []int64{1, 2, 3, 4, 5, 6})
	result := applyMonad(verbHash, m)
	if result.rank() != 0 {
		t.Errorf("rank: got %d want 0", result.rank())
	}
	if atomInt64(result) != 2 {
		t.Errorf("tally of 2x3 matrix: got %d want 2", atomInt64(result))
	}
}

func TestApplyDyadRank0Vectors(t *testing.T) {
	// rank-0 dyad applied to two vectors: element-wise
	a := vec([]int64{1, 2, 3})
	w := vec([]int64{4, 5, 6})
	result := applyDyad(verbPlus, a, w)
	if !reflect.DeepEqual(result.shape, []int{3}) {
		t.Errorf("shape: got %v want [3]", result.shape)
	}
	if !reflect.DeepEqual(result.data.([]int64), []int64{5, 7, 9}) {
		t.Errorf("data: got %v want [5 7 9]", result.data)
	}
}

func TestApplyDyadScalarExtension(t *testing.T) {
	// scalar + vector
	a := scalar(10)
	w := vec([]int64{1, 2, 3})
	result := applyDyad(verbPlus, a, w)
	if !reflect.DeepEqual(result.data.([]int64), []int64{11, 12, 13}) {
		t.Errorf("scalar extension: got %v want [11 12 13]", result.data)
	}
}

func TestWithRank(t *testing.T) {
	// withRank overrides rank triple
	v := withRank(verbPlus, 1, 1, 1)
	if v.monadRank != 1 || v.lRank != 1 || v.rRank != 1 {
		t.Errorf("withRank: got %d %d %d want 1 1 1", v.monadRank, v.lRank, v.rRank)
	}
}
