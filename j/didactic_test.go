package main

import "testing"

// didactic_test.go exercises complete J sentences end-to-end.
// The tests are ordered didactically: reading top to bottom gives a
// short tour of J's core ideas.

// run evaluates a J sentence and returns its display string.
func run(s string) string {
	return display(eval(parse(tokenise(s))))
}

// TestLiterals — how J writes numbers and arrays.
//
// The simplest J sentence is a noun: a number or sequence of numbers.
// Consecutive numbers on one line form a vector (a rank-1 array).
// J uses the underscore _ as a negative sign in literals; the minus
// sign - is reserved for the subtraction / negation verb.
func TestLiterals(t *testing.T) {
	for i, tt := range []struct {
		j, want string
	}{
		{"3", "3"},
		{"_3", "-3"}, // underscore prefix = negative
		{"1.5", "1.5"},
		{"1 2 3", "1 2 3"}, // three scalars become a vector
		{"1 2 3 4 5", "1 2 3 4 5"},
		{"1.5 2.5 3.5", "1.5 2.5 3.5"},
		{"0 _1 _2 _3", "0 -1 -2 -3"}, // negative elements in a vector
	} {
		if got := run(tt.j); got != tt.want {
			t.Errorf("%d: J %q\n got  %q\nwant %q", i, tt.j, got, tt.want)
		}
	}
}

// TestMonadicVerbs — a verb to the left of a noun is applied monadically.
//
// Monadic means one argument (the noun to the right).
// Most arithmetic verbs have a different monadic and dyadic meaning.
func TestMonadicVerbs(t *testing.T) {
	for i, tt := range []struct {
		j, want string
	}{
		{"- 3", "-3"},            // negate
		{"- _3", "3"},            // negate a negative
		{"* 5", "1"},             // signum: 1 for positive
		{"* _5", "-1"},           // signum: -1 for negative
		{"* 0", "0"},             // signum: 0 for zero
		{"% 4", "0.25"},          // reciprocal: 1 divided by the argument
		{"+ 42", "42"},           // identity: returns the argument unchanged
		{"# 1 2 3", "3"},         // tally: length of the leading dimension
		{"# 42", "1"},            // tally of a scalar is 1
		{"$ 1 2 3", "3"},         // shape: returns shape as a vector; a vector has one dimension
		{", 1 2 3", "1 2 3"},     // ravel: flatten to a vector (no-op for a vector)
		{"< 42", "+-\n|42|\n+-"}, // box: wrap any array in a container
	} {
		if got := run(tt.j); got != tt.want {
			t.Errorf("%d: J %q\n got  %q\nwant %q", i, tt.j, got, tt.want)
		}
	}
}

// TestDyadicVerbs — a verb between two nouns is applied dyadically.
//
// Dyadic means two arguments: the noun to the left and the noun to the right.
func TestDyadicVerbs(t *testing.T) {
	for i, tt := range []struct {
		j, want string
	}{
		{"1 + 2", "3"},
		{"5 - 3", "2"},
		{"3 * 4", "12"},
		{"7 % 2", "3.5"}, // division always produces a float
		{"3 < 5", "1"},   // less-than returns 1 (true) or 0 (false)
		{"5 < 3", "0"},
		{"5 > 3", "1"},
	} {
		if got := run(tt.j); got != tt.want {
			t.Errorf("%d: J %q\n got  %q\nwant %q", i, tt.j, got, tt.want)
		}
	}
}

// TestRightToLeft — J has no precedence rules; all verbs are equal.
//
// Evaluation proceeds strictly right-to-left: the rightmost verb
// always binds first. Parentheses override this order.
func TestRightToLeft(t *testing.T) {
	for i, tt := range []struct {
		j, want string
	}{
		// 2 * 3 + 4  =  2 * (3 + 4)  =  2 * 7  =  14
		// not: (2 * 3) + 4 = 10
		{"2 * 3 + 4", "14"},

		// 24 % 2 * 3  =  24 % (2 * 3)  =  24 % 6  =  4
		{"24 % 2 * 3", "4"},

		// parentheses force left-first evaluation
		{"(1 + 2) * 3", "9"}, // (1+2)*3 = 9, without parens: 1+(2*3) = 7
		{"2 * (i. 4)", "0 2 4 6"},
	} {
		if got := run(tt.j); got != tt.want {
			t.Errorf("%d: J %q\n got  %q\nwant %q", i, tt.j, got, tt.want)
		}
	}
}

// TestRank — verbs apply at their natural rank; J loops over the rest.
//
// Every verb has a rank: how many dimensions it "sees" per call.
// Rank-0 verbs (+ - * %) work on individual scalars.
// When a rank-0 verb meets a higher-rank array, J applies it to every
// element automatically. This is not a special case — it is the general
// rank-application loop, the same mechanism used by all verbs.
func TestRank(t *testing.T) {
	for i, tt := range []struct {
		j, want string
	}{
		// rank-0 verbs apply element-wise to vectors
		{"- 1 2 3", "-1 -2 -3"},
		{"* _2 0 3", "-1 0 1"},
		{"1 2 3 + 4 5 6", "5 7 9"},
		{"1 2 3 - 4 5 6", "-3 -3 -3"},

		// scalar extension: J broadcasts a scalar across a vector
		{"10 * 1 2 3", "10 20 30"},
		{"1 2 3 + 10", "11 12 13"},

		// rank-0 verbs apply to each element of a matrix
		{"- 2 3 $ i. 6", "0 -1 -2\n-3 -4 -5"},
		{"10 * 2 3 $ i. 6", "0 10 20\n30 40 50"},
	} {
		if got := run(tt.j); got != tt.want {
			t.Errorf("%d: J %q\n got  %q\nwant %q", i, tt.j, got, tt.want)
		}
	}
}

// TestShapeAndIota — building and inspecting arrays.
//
// i. generates a sequence of integers.
// $ (shape-of) and # (tally) inspect the structure of an array.
// Dyadic $ reshapes data into a new shape, cycling the fill if needed.
func TestShapeAndIota(t *testing.T) {
	for i, tt := range []struct {
		j, want string
	}{
		// i. n generates 0 1 2 ... n-1
		{"i. 5", "0 1 2 3 4"},
		{"i. 1", "0"},

		// i. with a vector argument generates an array of that shape
		{"i. 2 3", "0 1 2\n3 4 5"},
		{"$ i. 3 4", "3 4"}, // shape of i.(3 4) is 3 4

		// $ (monadic) returns the shape of any array
		{"$ 1 2 3", "3"},        // a vector has one dimension
		{"$ 2 3 $ i. 6", "2 3"}, // a 2x3 matrix has shape 2 3
		{"$ 42", ""},            // a scalar has an empty shape

		// # (monadic) returns the length of the leading dimension
		{"# 1 2 3", "3"},
		{"# 2 3 $ i. 6", "2"}, // leading dimension of a 2x3 matrix is 2
		{"# 42", "1"},         // a scalar has tally 1

		// dyadic $ reshapes: left is the new shape, right is the fill data
		{"2 3 $ i. 6", "0 1 2\n3 4 5"},
		{"3 3 $ i. 9", "0 1 2\n3 4 5\n6 7 8"},

		// if the fill is shorter than needed, it cycles
		{"2 3 $ 1 2", "1 2 1\n2 1 2"},
		{"3 $ 0", "0 0 0"},
	} {
		if got := run(tt.j); got != tt.want {
			t.Errorf("%d: J %q\n got  %q\nwant %q", i, tt.j, got, tt.want)
		}
	}
}

// TestRavelAndAppend — , flattens or joins arrays.
//
// Monadic , (ravel) collapses any array into a vector.
// Dyadic , (append) joins two arrays along the first axis.
func TestRavelAndAppend(t *testing.T) {
	for i, tt := range []struct {
		j, want string
	}{
		{", 2 3 $ i. 6", "0 1 2 3 4 5"}, // ravel a 2x3 matrix into 6 elements
		{", 42", "42"},                  // ravel a scalar gives a 1-element vector
		{"1 2 , 3 4", "1 2 3 4"},
		{"0 , i. 4", "0 0 1 2 3"},
	} {
		if got := run(tt.j); got != tt.want {
			t.Errorf("%d: J %q\n got  %q\nwant %q", i, tt.j, got, tt.want)
		}
	}
}

// TestInsert — the adverb / turns a dyadic verb into a fold.
//
// f/ inserts f between elements, reducing the array to a single value.
// Evaluation is right-to-left: +/ 1 2 3  =  1 + (2 + 3)  =  6.
func TestInsertFold(t *testing.T) {
	for i, tt := range []struct {
		j, want string
	}{
		{"+/ 1 2 3", "6"},
		{"+/ i. 5", "10"}, // 0+1+2+3+4
		{"*/ 1 2 3 4", "24"},
		{"+/ 1 2 3 4 5", "15"},
		// on a matrix, +/ reduces along the leading axis (column sums)
		{"+/ 2 3 $ i. 6", "3 5 7"}, // [0,1,2]+[3,4,5] = [3,5,7]
	} {
		if got := run(tt.j); got != tt.want {
			t.Errorf("%d: J %q\n got  %q\nwant %q", i, tt.j, got, tt.want)
		}
	}
}

// TestAssignment — =: binds a name to a value in the global vocabulary.
//
// The [ verb (left identity) is often used to display an assignment result:
// without it, the REPL would still compute the value but suppress output.
func TestAssignment(t *testing.T) {
	globals = map[string]*Array{} // reset for a clean slate
	verbGlobals = map[string]*Verb{}
	for i, tt := range []struct {
		j, want string
	}{
		{"[ x=: 42", "42"},
		{"x", "42"},     // recall the stored value
		{"x + 1", "43"}, // use in an expression
		{"[ v=: 1 2 3", "1 2 3"},
		{"# v", "3"},
		{"10 * v", "10 20 30"},
		{"[ M=: 2 3 $ i. 6", "0 1 2\n3 4 5"},
		{"$ M", "2 3"},
		{"# M", "2"},
		{"+/ M", "3 5 7"}, // column sums (insert along leading axis)
	} {
		if got := run(tt.j); got != tt.want {
			t.Errorf("%d: J %q\n got  %q\nwant %q", i, tt.j, got, tt.want)
		}
	}
}

// TestBoxes — < wraps any array into a scalar container; > unwraps it.
//
// Boxes let you store arrays of different shapes in a single array.
// A boxed array has rank 0 regardless of the contents.
func TestBoxes(t *testing.T) {
	for i, tt := range []struct {
		j, want string
	}{
		{"< 42", "+-\n|42|\n+-"},
		{"> < 42", "42"}, // box then unbox returns the original
		{"> < 1 2 3", "1 2 3"},
		{"# < 1 2 3", "1"}, // a box is rank 0, so tally is 1
		{"$ < 1 2 3", ""},  // a box is a scalar: empty shape
	} {
		if got := run(tt.j); got != tt.want {
			t.Errorf("%d: J %q\n got  %q\nwant %q", i, tt.j, got, tt.want)
		}
	}
}

// TestRankConjunction — " lets you override the rank at which a verb applies.
//
// f"n applies f treating each rank-n cell as one argument.
// This makes rank the primary tool for expressing iteration in J:
// instead of writing a loop, you choose the rank.
func TestRankConjunction(t *testing.T) {
	for i, tt := range []struct {
		j, want string
	}{
		// +/ on a matrix reduces along the leading axis: column sums
		{"+/ 2 3 $ i. 6", "3 5 7"}, // [0,1,2]+[3,4,5] = [3,5,7]

		// +/"1 applies +/ to each rank-1 cell (each row)
		// i. 3 4 =  0  1  2  3
		//           4  5  6  7
		//           8  9 10 11
		// row sums:  6   22   38
		{`+/"1 i. 3 4`, "6 22 38"},

		// -"0 applies - to each scalar — same as plain - on an array
		{`-"0 i. 4`, "0 -1 -2 -3"},

		// *"0 applies signum element-wise; same as plain * here
		{`*"0 i. 5`, "0 1 1 1 1"},
	} {
		if got := run(tt.j); got != tt.want {
			t.Errorf("%d: J %q\n got  %q\nwant %q", i, tt.j, got, tt.want)
		}
	}
}
