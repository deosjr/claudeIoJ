package main

import (
	"strconv"
	"strings"
)

// parse converts a token stream into a Sentence: a flat list of
// SentenceWords containing only syntactic information.
//
// The parser makes exactly one decision that requires semantic knowledge:
// it checks whether a tName token matches a known primitive, and if so
// classifies it as SynPrim rather than SynName.  This lets the evaluator
// skip the globals lookup for spellings like "i." that will never be
// reassigned.  All other name resolution — user-defined verbs and nouns —
// is deferred to the evaluator.
//
// Parenthesised sub-expressions become SynGroup nodes; they are NOT
// evaluated here.  Adverbs and conjunctions are left as separate
// SynAdverb / SynConj words; the evaluator folds them into derived verbs.
func parse(tokens []token) Sentence {
	sent, _ := parseFrom(tokens, 0)
	return sent
}

// parseFrom builds a Sentence starting at index start.
// It returns the Sentence and the index of the first unconsumed token.
// It stops and returns when it sees tRParen, consuming the ')'.
func parseFrom(tokens []token, start int) (Sentence, int) {
	var sent Sentence
	i := start
	for i < len(tokens) {
		t := tokens[i]
		switch t.kind {
		case tLParen:
			sub, j := parseFrom(tokens, i+1)
			sent = append(sent, SentenceWord{Kind: SynGroup, Sub: sub})
			i = j
		case tRParen:
			return sent, i + 1
		case tNumber:
			sent = append(sent, SentenceWord{Kind: SynNum, Text: t.value})
			i++
		case tString:
			sent = append(sent, SentenceWord{Kind: SynStr, Text: t.value})
			i++
		case tVerb:
			// Single-character verb spellings are always primitives.
			sent = append(sent, SentenceWord{Kind: SynPrim, Text: t.value})
			i++
		case tName:
			// Peek ahead: if followed by =:, this is an assignment target.
			if i+1 < len(tokens) && tokens[i+1].kind == tAssign {
				sent = append(sent, SentenceWord{Kind: SynAssign, Text: t.value})
				i += 2 // consume name and =:
				continue
			}
			// Classify by whether the name is a known primitive.
			if _, ok := primitives[t.value]; ok {
				sent = append(sent, SentenceWord{Kind: SynPrim, Text: t.value})
			} else {
				sent = append(sent, SentenceWord{Kind: SynName, Text: t.value})
			}
			i++
		case tAdverb:
			sent = append(sent, SentenceWord{Kind: SynAdverb, Text: t.value})
			i++
		case tConj:
			sent = append(sent, SentenceWord{Kind: SynConj, Text: t.value})
			i++
		default:
			i++
		}
	}
	return sent, i
}

// --- helpers ---

// parseNumber converts a J number token (uses _ for negative) to a scalar Array.
func parseNumber(s string) *Array {
	norm := strings.Replace(s, "_", "-", 1)
	if i, err := strconv.ParseInt(norm, 10, 64); err == nil {
		return scalar(i)
	}
	if f, err := strconv.ParseFloat(norm, 64); err == nil {
		return scalarF(f)
	}
	panic("parseNumber: cannot parse " + s)
}

// parseRankArg extracts a rank triple from a rank-argument noun.
// 1 number  → [n, n, n]     (same rank for monad and both dyad sides)
// 2 numbers → [v1, v0, v1]  (monad rank, then dyad left/right)
// 3 numbers → [v0, v1, v2]
func parseRankArg(a *Array) [3]int {
	ns := toInt64Slice(a)
	switch len(ns) {
	case 1:
		v := int(ns[0])
		return [3]int{v, v, v}
	case 2:
		return [3]int{int(ns[1]), int(ns[0]), int(ns[1])}
	case 3:
		return [3]int{int(ns[0]), int(ns[1]), int(ns[2])}
	}
	panic("parseRankArg: bad rank argument")
}
