package main

import (
	"strings"
	"unicode"
)

// TokenKind identifies the lexical category of a J token.
type TokenKind int

const (
	tNumber  TokenKind = iota
	tString            // 'hello'
	tName              // alphabetic name, e.g. i.
	tVerb              // single-character primitive verb
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

// verbChars lists all single-character primitive verb spellings.
const verbChars = "+-*%#$,<>[]"

// tokenise breaks a J sentence into tokens.
func tokenise(s string) []token {
	return tokeniseRunes([]rune(s))
}

// tokeniseRunes recursively consumes tokens from the front of rs.
// Each call to nextToken consumes exactly the runes needed for one token
// and returns the unconsumed remainder, so there is no position index to manage.
func tokeniseRunes(rs []rune) []token {
	if len(rs) == 0 {
		return nil
	}
	tok, rest, ok := nextToken(rs)
	if !ok {
		return tokeniseRunes(rest)
	}
	return append([]token{tok}, tokeniseRunes(rest)...)
}

// nextToken consumes the next token from the front of rs.
// It returns the token, the remaining runes, and whether a token was produced.
// Whitespace and unrecognised characters are consumed silently (ok = false).
func nextToken(rs []rune) (token, []rune, bool) {
	c := rs[0]
	switch {
	case c == ' ' || c == '\t':
		return token{}, rs[1:], false
	case c == '(':
		return token{tLParen, "("}, rs[1:], true
	case c == ')':
		return token{tRParen, ")"}, rs[1:], true
	case c == '/':
		return token{tAdverb, "/"}, rs[1:], true
	case c == '"':
		return token{tConj, `"`}, rs[1:], true
	case c == '\'':
		j := 1
		for j < len(rs) && rs[j] != '\'' {
			j++
		}
		rest := rs[j+1:]
		if j >= len(rs) {
			rest = nil // unterminated string: consume to end
		}
		return token{tString, string(rs[1:j])}, rest, true
	case c == '_' || unicode.IsDigit(c):
		j := 0
		if rs[j] == '_' {
			j++
		}
		for j < len(rs) && (unicode.IsDigit(rs[j]) || rs[j] == '.') {
			j++
		}
		if j < len(rs) && (rs[j] == 'e' || rs[j] == 'E') {
			j++
			if j < len(rs) && (rs[j] == '+' || rs[j] == '-') {
				j++
			}
			for j < len(rs) && unicode.IsDigit(rs[j]) {
				j++
			}
		}
		return token{tNumber, string(rs[:j])}, rs[j:], true
	case unicode.IsLetter(c):
		j := 0
		for j < len(rs) && (unicode.IsLetter(rs[j]) || rs[j] == '_') {
			j++
		}
		// consume trailing dot if it's part of a name (e.g. i.)
		if j < len(rs) && rs[j] == '.' {
			j++
		}
		return token{tName, string(rs[:j])}, rs[j:], true
	case c == '=':
		if len(rs) > 1 && rs[1] == ':' {
			return token{tAssign, "=:"}, rs[2:], true
		}
		return token{}, rs[1:], false // bare = is not J syntax
	case strings.ContainsRune(verbChars, c):
		return token{tVerb, string(c)}, rs[1:], true
	default:
		return token{}, rs[1:], false
	}
}
