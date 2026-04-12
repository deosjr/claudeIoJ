package main

import (
	"strconv"
	"strings"
	"unicode"
)

// TokenKind identifies the category of a J token.
type TokenKind int

const (
	tNumber  TokenKind = iota
	tString            // 'hello'
	tName              // alphabetic name, e.g. i.
	tVerb              // primitive verb spelling
	tAdverb            // /
	tConj              // "
	tAssign            // =:
	tLParen
	tRParen
)

type token struct {
	kind  TokenKind
	value string
}

// verbSpellings lists all primitive verb characters we recognise.
const verbChars = "+-*%#$,<>[]"

// tokenise breaks a J sentence into tokens.
func tokenise(s string) []token {
	var tokens []token
	i := 0
	for i < len(s) {
		c := rune(s[i])
		switch {
		case c == ' ' || c == '\t':
			i++
		case c == '(':
			tokens = append(tokens, token{tLParen, "("})
			i++
		case c == ')':
			tokens = append(tokens, token{tRParen, ")"})
			i++
		case c == '\'':
			j := i + 1
			for j < len(s) && s[j] != '\'' {
				j++
			}
			tokens = append(tokens, token{tString, s[i+1 : j]})
			if j < len(s) {
				j++
			}
			i = j
		case c == '_' || unicode.IsDigit(c):
			j := i
			if s[j] == '_' {
				j++
			}
			for j < len(s) && (unicode.IsDigit(rune(s[j])) || s[j] == '.') {
				j++
			}
			if j < len(s) && (s[j] == 'e' || s[j] == 'E') {
				j++
				if j < len(s) && (s[j] == '+' || s[j] == '-') {
					j++
				}
				for j < len(s) && unicode.IsDigit(rune(s[j])) {
					j++
				}
			}
			tokens = append(tokens, token{tNumber, s[i:j]})
			i = j
		case unicode.IsLetter(c):
			j := i
			for j < len(s) && (unicode.IsLetter(rune(s[j])) || s[j] == '_') {
				j++
			}
			// consume trailing dot if it's part of a name (e.g. i.)
			if j < len(s) && s[j] == '.' {
				j++
			}
			tokens = append(tokens, token{tName, s[i:j]})
			i = j
		case c == '=':
			if i+1 < len(s) && s[i+1] == ':' {
				tokens = append(tokens, token{tAssign, "=:"})
				i += 2
			} else {
				i++
			}
		case c == '/':
			tokens = append(tokens, token{tAdverb, "/"})
			i++
		case c == '"':
			tokens = append(tokens, token{tConj, `"`})
			i++
		case strings.ContainsRune(verbChars, c):
			tokens = append(tokens, token{tVerb, string(c)})
			i++
		default:
			i++
		}
	}
	return tokens
}

// --- part-of-speech types ---

type posKind int

const (
	posNoun   posKind = iota
	posVerb
	posAssign // assignment target; name field holds the variable name
	posMark
)

type word struct {
	pos  posKind
	noun *Array
	verb *Verb
	name string // for posAssign
}

// globals is the module-level symbol table for noun =: assignments.
// verbGlobals holds verb assignments (e.g. mul =: *).
var globals = map[string]*Array{}
var verbGlobals = map[string]*Verb{}

// --- eval: entry point ---

// eval tokenises and evaluates a J sentence.
func eval(tokens []token) *Array {
	words, _ := parseWords(tokens, 0)
	return evalWords(words)
}

// parseWords builds a word list from tokens starting at index start.
// It returns the words and the index of the first unconsumed token.
// It stops at tRParen, returning the index after the ')'.
func parseWords(tokens []token, start int) ([]word, int) {
	var words []word
	i := start
	for i < len(tokens) {
		t := tokens[i]
		switch t.kind {
		case tLParen:
			inner, j := parseWords(tokens, i+1)
			noun := evalWords(inner)
			words = append(words, word{pos: posNoun, noun: noun})
			i = j
		case tRParen:
			return words, i + 1
		case tNumber:
			words = append(words, word{pos: posNoun, noun: parseNumber(t.value)})
			i++
		case tString:
			runes := make([]int64, 0, len(t.value))
			for _, r := range t.value {
				runes = append(runes, int64(r))
			}
			words = append(words, word{pos: posNoun, noun: vec(runes)})
			i++
		case tVerb:
			v, ok := primitives[t.value]
			if !ok {
				panic("unknown verb: " + t.value)
			}
			words = append(words, word{pos: posVerb, verb: v})
			i++
		case tName:
			// Check for assignment: Name =: ...
			if i+1 < len(tokens) && tokens[i+1].kind == tAssign {
				words = append(words, word{pos: posAssign, name: t.value})
				i += 2
				continue
			}
			if v, ok := primitives[t.value]; ok {
				words = append(words, word{pos: posVerb, verb: v})
			} else if v, ok := verbGlobals[t.value]; ok {
				words = append(words, word{pos: posVerb, verb: v})
			} else if n, ok := globals[t.value]; ok {
				words = append(words, word{pos: posNoun, noun: n})
			} else {
				panic("unknown name: " + t.value)
			}
			i++
		case tAdverb:
			if t.value == "/" {
				if len(words) > 0 && words[len(words)-1].pos == posVerb {
					v := insertAdverb(words[len(words)-1].verb)
					words[len(words)-1] = word{pos: posVerb, verb: v}
				} else {
					panic("/ without preceding verb")
				}
			}
			i++
		case tConj:
			if t.value == `"` {
				// rank conjunction: the next token(s) are the rank argument
				if len(words) == 0 || words[len(words)-1].pos != posVerb {
					panic(`" without preceding verb`)
				}
				i++ // consume "
				var rankNoun *Array
				if i < len(tokens) {
					switch tokens[i].kind {
					case tNumber:
						rankNoun = parseNumber(tokens[i].value)
						i++
					case tLParen:
						inner, j := parseWords(tokens, i+1)
						rankNoun = evalWords(inner)
						i = j
					}
				}
				if rankNoun == nil {
					panic(`" with no rank argument`)
				}
				ranks := parseRankArg(rankNoun)
				v := words[len(words)-1].verb
				words[len(words)-1] = word{pos: posVerb, verb: withRank(v, ranks[0], ranks[1], ranks[2])}
			}
		default:
			i++
		}
	}
	return words, i
}

// evalWords evaluates a flat word list using right-to-left semantics.
// In J all verbs have equal precedence; the leftmost verb is the "main
// connective" because everything to its right is evaluated first.
func evalWords(words []word) *Array {
	if len(words) == 0 {
		return nil
	}
	if len(words) == 1 {
		if words[0].pos == posNoun {
			return words[0].noun
		}
		return nil
	}

	// Handle assignment: Name =: rhs  (may be preceded by verbs like [)
	for i, w := range words {
		if w.pos == posAssign {
			rhs := words[i+1:]
			// Verb assignment: Name =: someVerb
			if len(rhs) == 1 && rhs[0].pos == posVerb {
				verbGlobals[w.name] = rhs[0].verb
				// Nothing to substitute back: discard the assign word and
				// evaluate whatever was to its left (e.g. a leading [).
				return evalWords(words[:i])
			}
			// Noun assignment
			result := evalWords(rhs)
			globals[w.name] = result
			newWords := make([]word, i+1)
			copy(newWords, words[:i])
			newWords[i] = word{pos: posNoun, noun: result}
			return evalWords(newWords)
		}
	}

	// Find the leftmost verb — this is the principal verb of the sentence.
	verbIdx := -1
	for i, w := range words {
		if w.pos == posVerb {
			verbIdx = i
			break
		}
	}

	if verbIdx == -1 {
		// No verb: all nouns — assemble into a vector.
		return assembleNouns(words)
	}

	v := words[verbIdx].verb
	right := evalWords(words[verbIdx+1:])
	if right == nil {
		panic("verb " + v.name + " has no right argument")
	}

	if verbIdx == 0 {
		// Monad
		return applyMonad(v, right)
	}

	// Dyad: left is everything before the verb.
	left := evalWords(words[:verbIdx])
	if left == nil {
		panic("dyad " + v.name + " has no left argument")
	}
	return applyDyad(v, left, right)
}

// assembleNouns combines a sequence of noun words into a single array.
// Scalars are combined into a flat vector; higher-rank arrays are stacked.
func assembleNouns(words []word) *Array {
	if len(words) == 1 {
		return words[0].noun
	}
	// Check if all are rank-0 scalars.
	allScalar := true
	for _, w := range words {
		if w.noun.rank() != 0 {
			allScalar = false
			break
		}
	}
	if allScalar {
		hasFloat := false
		for _, w := range words {
			if isFloat(w.noun) {
				hasFloat = true
				break
			}
		}
		if hasFloat {
			vals := make([]float64, len(words))
			for i, w := range words {
				vals[i] = atomF(w.noun)
			}
			return vecF(vals)
		}
		vals := make([]int64, len(words))
		for i, w := range words {
			vals[i] = atomI(w.noun)
		}
		return vec(vals)
	}
	// Higher-rank: assemble as rows.
	nouns := make([]*Array, len(words))
	for i, w := range words {
		nouns[i] = w.noun
	}
	return assemble(nouns, []int{len(nouns)})
}

// parseNumber converts a J number token (may use _ for negative) to an Array.
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
// 1 number -> [n,n,n]; 2 numbers -> [v1,v0,v1]; 3 numbers -> [v0,v1,v2]
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

