package main

import (
	"reflect"
	"testing"
)

func TestNegate(t *testing.T) {
	result := applyMonad(verbMinus, scalar(3))
	if atomInt64(result) != -3 {
		t.Errorf("negate 3: got %d want -3", atomInt64(result))
	}
}

func TestAdd(t *testing.T) {
	result := applyDyad(verbPlus, scalar(2), scalar(3))
	if atomInt64(result) != 5 {
		t.Errorf("2+3: got %d want 5", atomInt64(result))
	}
}

func TestTally(t *testing.T) {
	result := applyMonad(verbHash, vec([]int64{1, 2, 3}))
	if atomInt64(result) != 3 {
		t.Errorf("# 1 2 3: got %d want 3", atomInt64(result))
	}
}

func TestTallyScalar(t *testing.T) {
	result := applyMonad(verbHash, scalar(42))
	if atomInt64(result) != 1 {
		t.Errorf("# scalar: got %d want 1", atomInt64(result))
	}
}

func TestShape(t *testing.T) {
	m := matrix([]int{2, 3}, []int64{1, 2, 3, 4, 5, 6})
	result := applyMonad(verbDollar, m)
	if !reflect.DeepEqual(result.shape, []int{2}) {
		t.Errorf("$ shape: got %v want [2]", result.shape)
	}
	if !reflect.DeepEqual(result.data.([]int64), []int64{2, 3}) {
		t.Errorf("$ data: got %v want [2 3]", result.data)
	}
}

func TestRavel(t *testing.T) {
	m := matrix([]int{2, 3}, []int64{1, 2, 3, 4, 5, 6})
	result := applyMonad(verbComma, m)
	if !reflect.DeepEqual(result.shape, []int{6}) {
		t.Errorf(", shape: got %v want [6]", result.shape)
	}
	if !reflect.DeepEqual(result.data.([]int64), []int64{1, 2, 3, 4, 5, 6}) {
		t.Errorf(", data: got %v want [1..6]", result.data)
	}
}

func TestBox(t *testing.T) {
	boxed := applyMonad(verbLt, scalar(42))
	if boxed.rank() != 0 {
		t.Errorf("< rank: got %d want 0", boxed.rank())
	}
	if !isBox(boxed) {
		t.Errorf("< result not boxed")
	}
}

func TestUnbox(t *testing.T) {
	boxed := applyMonad(verbLt, scalar(42))
	result := applyMonad(verbGt, boxed)
	if result.rank() != 0 {
		t.Errorf("> rank: got %d want 0", result.rank())
	}
	if atomInt64(result) != 42 {
		t.Errorf("> value: got %d want 42", atomInt64(result))
	}
}

func TestReshape(t *testing.T) {
	shape := vec([]int64{2, 3})
	data := vec([]int64{1, 2, 3, 4, 5, 6})
	result := applyDyad(verbHash, shape, data)
	if !reflect.DeepEqual(result.shape, []int{2, 3}) {
		t.Errorf("reshape shape: got %v want [2 3]", result.shape)
	}
}

func TestInsert(t *testing.T) {
	// +/ 1 2 3 = 6
	plusInsert := insertAdverb(verbPlus)
	result := applyMonad(plusInsert, vec([]int64{1, 2, 3}))
	if atomInt64(result) != 6 {
		t.Errorf("+/ 1 2 3: got %d want 6", atomInt64(result))
	}
}
