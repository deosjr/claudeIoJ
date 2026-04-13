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
	sent, _ := parseTokens(tokens)
	return sent
}

// parseTokens recursively consumes tokens from the front, building a Sentence.
// It returns the Sentence and any unconsumed tokens — non-empty only when
// a tRParen terminates a parenthesised sub-sentence mid-stream.
func parseTokens(tokens []token) (Sentence, []token) {
	if len(tokens) == 0 {
		return nil, nil
	}
	// tRParen closes the current group; return remaining tokens to the caller
	// that opened the matching tLParen.
	if tokens[0].kind == tRParen {
		return nil, tokens[1:]
	}
	word, rest, ok := consumeWord(tokens)
	sent, remaining := parseTokens(rest)
	if !ok {
		return sent, remaining
	}
	return append([]SentenceWord{word}, sent...), remaining
}

// consumeWord produces one SentenceWord from the front of tokens.
// It returns the word, the unconsumed remainder, and whether a word was produced.
// tRParen is never passed here; parseTokens handles it before calling consumeWord.
func consumeWord(tokens []token) (SentenceWord, []token, bool) {
	t := tokens[0]
	switch t.kind {
	case tLParen:
		// Recurse to collect the sub-sentence; parseTokens stops at tRParen.
		sub, rest := parseTokens(tokens[1:])
		return SentenceWord{Kind: SynGroup, Sub: sub}, rest, true
	case tNumber:
		return SentenceWord{Kind: SynNum, Text: t.value}, tokens[1:], true
	case tString:
		return SentenceWord{Kind: SynStr, Text: t.value}, tokens[1:], true
	case tVerb:
		// Single-character verb spellings are always primitives.
		return SentenceWord{Kind: SynPrim, Text: t.value}, tokens[1:], true
	case tName:
		// Peek ahead: if followed by =:, this is an assignment target.
		if len(tokens) > 1 && tokens[1].kind == tAssign {
			return SentenceWord{Kind: SynAssign, Text: t.value}, tokens[2:], true
		}
		// Classify by whether the name is a known primitive.
		if _, ok := primitives[t.value]; ok {
			return SentenceWord{Kind: SynPrim, Text: t.value}, tokens[1:], true
		}
		return SentenceWord{Kind: SynName, Text: t.value}, tokens[1:], true
	case tAdverb:
		return SentenceWord{Kind: SynAdverb, Text: t.value}, tokens[1:], true
	case tConj:
		return SentenceWord{Kind: SynConj, Text: t.value}, tokens[1:], true
	default:
		return SentenceWord{}, tokens[1:], false
	}
}

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
