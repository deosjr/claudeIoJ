package main

import (
	"reflect"
	"testing"
)

func TestTokenise(t *testing.T) {
	for i, tt := range []struct {
		input string
		want  []token
	}{
		{"3", []token{{tNumber, "3"}}},
		{"3.14", []token{{tNumber, "3.14"}}},
		{"_3", []token{{tNumber, "_3"}}},
		{"1 2 3", []token{{tNumber, "1"}, {tNumber, "2"}, {tNumber, "3"}}},
		{"1 + 2", []token{{tNumber, "1"}, {tVerb, "+"}, {tNumber, "2"}}},
		{"# 1 2 3", []token{{tVerb, "#"}, {tNumber, "1"}, {tNumber, "2"}, {tNumber, "3"}}},
		{"+/", []token{{tVerb, "+"}, {tAdverb, "/"}}},
	} {
		got := tokenise(tt.input)
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("%d: got %v want %v", i, got, tt.want)
		}
	}
}

func TestEvalScalar(t *testing.T) {
	result := eval(parse(tokenise("3")))
	if result == nil || result.rank() != 0 || atomInt64(result) != 3 {
		t.Errorf("eval '3': got %v", result)
	}
}

func TestEvalVector(t *testing.T) {
	result := eval(parse(tokenise("1 2 3")))
	if result == nil {
		t.Fatal("eval '1 2 3': nil result")
	}
	if !reflect.DeepEqual(result.shape, []int{3}) {
		t.Errorf("shape: got %v want [3]", result.shape)
	}
	if !reflect.DeepEqual(result.data.([]int64), []int64{1, 2, 3}) {
		t.Errorf("data: got %v want [1 2 3]", result.data)
	}
}

func TestEvalMonad(t *testing.T) {
	result := eval(parse(tokenise("- 3")))
	if atomInt64(result) != -3 {
		t.Errorf("- 3: got %d want -3", atomInt64(result))
	}
}

func TestEvalDyad(t *testing.T) {
	result := eval(parse(tokenise("1 + 2")))
	if atomInt64(result) != 3 {
		t.Errorf("1 + 2: got %d want 3", atomInt64(result))
	}
}

func TestEvalVectorDyad(t *testing.T) {
	result := eval(parse(tokenise("1 2 3 + 4 5 6")))
	if !reflect.DeepEqual(result.data.([]int64), []int64{5, 7, 9}) {
		t.Errorf("1 2 3 + 4 5 6: got %v want [5 7 9]", result.data)
	}
}

func TestEvalMonadPlus(t *testing.T) {
	// monadic + is identity
	result := eval(parse(tokenise("+ 1 2 3")))
	if !reflect.DeepEqual(result.data.([]int64), []int64{1, 2, 3}) {
		t.Errorf("+ 1 2 3: got %v want [1 2 3]", result.data)
	}
}

func TestEvalTally(t *testing.T) {
	result := eval(parse(tokenise("# 1 2 3")))
	if atomInt64(result) != 3 {
		t.Errorf("# 1 2 3: got %d want 3", atomInt64(result))
	}
}

func TestEvalInsert(t *testing.T) {
	// +/ 1 2 3 = 6
	result := eval(parse(tokenise("+/ 1 2 3")))
	if atomInt64(result) != 6 {
		t.Errorf("+/ 1 2 3: got %d want 6", atomInt64(result))
	}
}

func TestEvalParens(t *testing.T) {
	// (1 + 2) = 3 as scalar
	result := eval(parse(tokenise("(1 + 2)")))
	if atomInt64(result) != 3 {
		t.Errorf("(1 + 2): got %d want 3", atomInt64(result))
	}
}

func TestEvalIota(t *testing.T) {
	// i. 5 = 0 1 2 3 4
	result := eval(parse(tokenise("i. 5")))
	if !reflect.DeepEqual(result.data.([]int64), []int64{0, 1, 2, 3, 4}) {
		t.Errorf("i. 5: got %v", result.data)
	}
}

func TestEvalReshapeWithIota(t *testing.T) {
	// 2 3 $ i. 6 -> 2x3 matrix
	result := eval(parse(tokenise("2 3 $ i. 6")))
	if !reflect.DeepEqual(result.shape, []int{2, 3}) {
		t.Errorf("2 3 $ i. 6 shape: got %v want [2 3]", result.shape)
	}
	if !reflect.DeepEqual(result.data.([]int64), []int64{0, 1, 2, 3, 4, 5}) {
		t.Errorf("2 3 $ i. 6 data: got %v want [0..5]", result.data)
	}
}

func TestEvalNegativeNumber(t *testing.T) {
	result := eval(parse(tokenise("_3")))
	if atomInt64(result) != -3 {
		t.Errorf("_3: got %d want -3", atomInt64(result))
	}
}
