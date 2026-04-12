# claudeIoJ

A minimal J interpreter in Go, following Roger Hui's
[An Implementation of J](https://www.jsoftware.com/ioj/ioj.htm).

Built as a learning exercise alongside
[bowenProlog](https://github.com/deosjr/bowenProlog) and
[claudeWAM](https://github.com/deosjr/claudeWAM).

## Usage

```
cd j
go run .
```

The REPL prompt is three spaces, matching standard J.

```
   1 2 3 + 4 5 6
5 7 9
   [ M=: 10 * i.3 4
 0 10 20  30
40 50 60  70
80 90 100 110
   $ M
3 4
   +/ i. 10
45
```

## What works

**Arrays** — integers, floats, booleans, boxed arrays; any rank; row-major flat storage.

**Primitive verbs**

| Spelling | Monad        | Dyad      |
|----------|--------------|-----------|
| `+`      | identity     | add       |
| `-`      | negate       | subtract  |
| `*`      | signum       | multiply  |
| `%`      | reciprocal   | divide    |
| `#`      | tally        | reshape   |
| `$`      | shape        | reshape   |
| `,`      | ravel        | append    |
| `<`      | box          | less-than |
| `>`      | unbox        | greater-than |
| `[`      | identity     | left      |
| `]`      | identity     | right     |
| `i.`     | iota / shape-fill | —    |

**Adverb** `/` — insert (fold): `+/ 1 2 3` → `6`

**Conjunction** `"` — rank override: `+/"1` inserts along rows

**Rank loop** — all verbs apply at their natural rank; the rank loop in
`verb.go` handles looping over frame cells and assembling results
automatically.

**Assignment** `=:` — global variable binding: `M=: i. 5`

**Parentheses** — standard grouping: `(1 + 2) * 3` → `9`

**Negative literals** use the J underscore convention: `_3` is −3.

## What is not implemented (yet)

- User-defined verbs (`=:` with an explicit definition `3 : '...'`)
- Adverbs beyond `/` (no `\`, `~`, `&`, `@`, ...)
- Trains (forks and hooks)
- Control flow
- Multiple locales / namespaces
- Extended/rational numeric types
- `i.` dyad (index of)

## File layout

```
j/
  array.go          Array type, constructors, cell extraction, assemble
  verb.go           Verb type, applyMonad/applyDyad rank loop, withRank
  primitives.go     All primitive verb definitions
  parse.go          Tokeniser, recursive-descent parser, evaluator, globals
  display.go        Array formatting for REPL output
  main.go           REPL
  array_test.go
  verb_test.go
  primitives_test.go
  parse_test.go
```

## Design notes

**Array representation** — `Array` holds a flat `[]int64`, `[]float64`,
`[]bool`, or `[]*Array` slice plus a shape. Scalars have a nil shape.
All indexing is row-major.

**Right-to-left evaluation** — `evalWords` finds the leftmost verb in the
word list and treats it as the principal connective. Everything to its right
is recursively evaluated first, giving J's right-to-left semantics without
an explicit stack machine.

**Rank loop** — `applyMonad` and `applyDyad` in `verb.go` implement the
general rank-application loop. Each verb carries a rank triple
`(monadRank, lRank, rRank)`; the loop slices the argument into cells of
that rank, applies the verb's core function to each cell, and reassembles
the results. `withRank` (the `"` conjunction) returns a copy of a verb with
the triple overridden.
