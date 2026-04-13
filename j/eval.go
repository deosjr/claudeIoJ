package main

// --- symbol tables ---

// globals holds noun (array) assignments: name =: someArray
var globals = map[string]*Array{}

// verbGlobals holds verb assignments: name =: someVerb
var verbGlobals = map[string]*Verb{}

// --- internal resolved word list ---

// posKind is the part-of-speech of a fully resolved word.
type posKind int

const (
	posNoun posKind = iota
	posVerb
	posAssign // assignment target; name field holds the variable name
	posMark
)

// word is a fully resolved word: a live *Array or *Verb, ready for evaluation.
// It is the output of resolve and the input to evalWords.
type word struct {
	pos  posKind
	noun *Array
	verb *Verb
	name string // for posAssign
}

// --- eval: the public entry point ---

// eval resolves a parsed Sentence and evaluates it to a value.
// The three-phase pipeline is:
//
//	tokenise → parse → eval
//	         ↑ syntax  ↑ semantics
func eval(sent Sentence) *Array {
	return evalWords(resolve(sent))
}

// --- resolve: syntax → live objects ---

// resolve walks a Sentence and produces a flat []word list with all
// syntactic work done:
//   - number/string literals are parsed into *Array values
//   - SynGroup sub-sentences are recursively evaluated into noun words
//   - SynPrim names are looked up in the primitives map
//   - SynName names are resolved against verbGlobals then globals
//   - SynAdverb ("/") folds into the preceding verb via insertAdverb
//   - SynConj (`"`) consumes the following rank argument and folds into
//     the preceding verb via withRank
//
// After resolve, evalWords sees only posNoun, posVerb, and posAssign words.
func resolve(sent Sentence) []word {
	var words []word
	for i := 0; i < len(sent); {
		sw := sent[i]
		switch sw.Kind {
		case SynNum:
			words = append(words, word{pos: posNoun, noun: parseNumber(sw.Text)})
			i++
		case SynStr:
			runes := make([]int64, 0, len(sw.Text))
			for _, r := range sw.Text {
				runes = append(runes, int64(r))
			}
			words = append(words, word{pos: posNoun, noun: vec(runes)})
			i++
		case SynGroup:
			// Parenthesised sub-sentence: evaluate it now, treat result as a noun.
			noun := evalWords(resolve(sw.Sub))
			words = append(words, word{pos: posNoun, noun: noun})
			i++
		case SynPrim:
			words = append(words, word{pos: posVerb, verb: primitives[sw.Text]})
			i++
		case SynName:
			// User-defined names: check verb namespace first, then noun namespace.
			if v, ok := verbGlobals[sw.Text]; ok {
				words = append(words, word{pos: posVerb, verb: v})
			} else if n, ok := globals[sw.Text]; ok {
				words = append(words, word{pos: posNoun, noun: n})
			} else {
				panic("unknown name: " + sw.Text)
			}
			i++
		case SynAssign:
			words = append(words, word{pos: posAssign, name: sw.Text})
			i++
		case SynAdverb:
			// "/" must immediately follow a verb.
			if sw.Text == "/" {
				if len(words) == 0 || words[len(words)-1].pos != posVerb {
					panic("/ without preceding verb")
				}
				v := insertAdverb(words[len(words)-1].verb)
				words[len(words)-1] = word{pos: posVerb, verb: v}
			}
			i++
		case SynConj:
			// `"` forms a rank conjunction: verb " rankArg → derived verb.
			// The rank argument is the next word in the sentence (number or group).
			if sw.Text == `"` {
				if len(words) == 0 || words[len(words)-1].pos != posVerb {
					panic(`" without preceding verb`)
				}
				i++ // consume the `"` word
				var rankNoun *Array
				if i < len(sent) {
					switch sent[i].Kind {
					case SynNum:
						rankNoun = parseNumber(sent[i].Text)
						i++
					case SynGroup:
						rankNoun = evalWords(resolve(sent[i].Sub))
						i++
					}
				}
				if rankNoun == nil {
					panic(`" with no rank argument`)
				}
				ranks := parseRankArg(rankNoun)
				v := words[len(words)-1].verb
				words[len(words)-1] = word{pos: posVerb, verb: withRank(v, ranks[0], ranks[1], ranks[2])}
			} else {
				i++
			}
		default:
			i++
		}
	}
	return words
}

// --- evalWords: the J evaluation algorithm ---

// evalWords evaluates a flat resolved word list using J's right-to-left rule.
// All verbs have equal precedence; the leftmost verb is the principal verb
// because everything to its right is evaluated first (recursively).
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
		if w.pos != posAssign {
			continue
		}
		rhs := words[i+1:]
		// Verb assignment: Name =: someVerb
		if len(rhs) == 1 && rhs[0].pos == posVerb {
			verbGlobals[w.name] = rhs[0].verb
			return evalWords(words[:i])
		}
		// Noun assignment: evaluate rhs, store, then substitute back.
		result := evalWords(rhs)
		globals[w.name] = result
		newWords := make([]word, i+1)
		copy(newWords, words[:i])
		newWords[i] = word{pos: posNoun, noun: result}
		return evalWords(newWords)
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
		// Monad: verb with one right argument.
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
	nouns := make([]*Array, len(words))
	for i, w := range words {
		nouns[i] = w.noun
	}
	return assemble(nouns, []int{len(nouns)})
}
