package main

import (
	"reflect"
	"testing"
)

func TestScalar(t *testing.T) {
	a := scalar(42)
	if a.rank() != 0 {
		t.Errorf("scalar rank: got %d want 0", a.rank())
	}
	if a.n() != 1 {
		t.Errorf("scalar n: got %d want 1", a.n())
	}
}

func TestVec(t *testing.T) {
	a := vec([]int64{1, 2, 3})
	if a.rank() != 1 {
		t.Errorf("vec rank: got %d want 1", a.rank())
	}
	if !reflect.DeepEqual(a.shape, []int{3}) {
		t.Errorf("vec shape: got %v want [3]", a.shape)
	}
	if a.n() != 3 {
		t.Errorf("vec n: got %d want 3", a.n())
	}
}

func TestMatrix(t *testing.T) {
	a := matrix([]int{2, 3}, []int64{1, 2, 3, 4, 5, 6})
	if a.rank() != 2 {
		t.Errorf("matrix rank: got %d want 2", a.rank())
	}
	if a.n() != 6 {
		t.Errorf("matrix n: got %d want 6", a.n())
	}
}

func TestCellShape(t *testing.T) {
	a := matrix([]int{2, 3}, []int64{1, 2, 3, 4, 5, 6})
	cs := a.cellShape(1)
	if !reflect.DeepEqual(cs, []int{3}) {
		t.Errorf("cellShape(1): got %v want [3]", cs)
	}
	fs := a.frameShape(1)
	if !reflect.DeepEqual(fs, []int{2}) {
		t.Errorf("frameShape(1): got %v want [2]", fs)
	}
}

func TestCellExtraction(t *testing.T) {
	a := matrix([]int{2, 3}, []int64{1, 2, 3, 4, 5, 6})
	row := a.cell(0, []int{3})
	if !reflect.DeepEqual(row.shape, []int{3}) {
		t.Errorf("cell shape: got %v want [3]", row.shape)
	}
	if !reflect.DeepEqual(row.data.([]int64), []int64{1, 2, 3}) {
		t.Errorf("cell data: got %v want [1 2 3]", row.data)
	}
	row1 := a.cell(1, []int{3})
	if !reflect.DeepEqual(row1.data.([]int64), []int64{4, 5, 6}) {
		t.Errorf("cell(1) data: got %v want [4 5 6]", row1.data)
	}
}

func TestAssemble(t *testing.T) {
	// three scalar results assembled into a vector
	r := assemble([]*Array{scalar(1), scalar(2), scalar(3)}, []int{3})
	if !reflect.DeepEqual(r.shape, []int{3}) {
		t.Errorf("assemble shape: got %v want [3]", r.shape)
	}
	if !reflect.DeepEqual(r.data.([]int64), []int64{1, 2, 3}) {
		t.Errorf("assemble data: got %v want [1 2 3]", r.data)
	}
}

func TestAssembleRows(t *testing.T) {
	// two row vectors assembled into a matrix
	row0 := vec([]int64{1, 2, 3})
	row1 := vec([]int64{4, 5, 6})
	r := assemble([]*Array{row0, row1}, []int{2})
	if !reflect.DeepEqual(r.shape, []int{2, 3}) {
		t.Errorf("assemble rows shape: got %v want [2 3]", r.shape)
	}
}
