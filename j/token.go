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
