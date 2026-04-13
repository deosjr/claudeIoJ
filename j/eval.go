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

// allVerbWords reports whether every word in the slice is a verb.
func allVerbWords(words []word) bool {
	for _, w := range words {
		if w.pos != posVerb {
			return false
		}
	}
	return true
}

// --- trains ---

// foldTrains scans a word list for runs of two or more consecutive verb words
// and folds each run into a single derived verb (hook or fork).
// It does not absorb a preceding noun; capped forks are handled by the
// SynGroup case in resolve, where parentheses make the context unambiguous.
func foldTrains(words []word) []word {
	var result []word
	for i := 0; i < len(words); {
		if words[i].pos != posVerb {
			result = append(result, words[i])
			i++
			continue
		}
		// collect the run of consecutive verb words
		j := i
		for j < len(words) && words[j].pos == posVerb {
			j++
		}
		run := words[i:j]
		// Don't fold a verb run that is immediately followed by a noun:
		// those verbs are applied right-to-left to that noun, not a train.
		// Trains only form when the verb run is at the end (no noun to the right).
		if j < len(words) && words[j].pos == posNoun {
			result = append(result, run...)
			i = j
			continue
		}
		if len(run) < 2 {
			result = append(result, run[0])
			i = j
			continue
		}
		verbs := make([]*Verb, len(run))
		for k, w := range run {
			verbs[k] = w.verb
		}
		result = append(result, word{pos: posVerb, verb: foldVerbRun(verbs)})
		i = j
	}
	return result
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
	for i := 0; i < len(sent); i++ {
		sw := sent[i]
		switch sw.Kind {
		case SynNum:
			words = append(words, word{pos: posNoun, noun: parseNumber(sw.Text)})
		case SynStr:
			runes := make([]int64, 0, len(sw.Text))
			for _, r := range sw.Text {
				runes = append(runes, int64(r))
			}
			words = append(words, word{pos: posNoun, noun: vec(runes)})
		case SynGroup:
			// Parenthesised sub-sentence.
			// Three cases:
			//   1. Capped fork: (n f g …) — leading noun, all rest are verbs.
			//      The noun becomes a constant verb as the leftmost tine.
			//   2. Pure-verb train: all words are verbs — return as a single verb.
			//   3. Everything else: evaluate and return as a noun.
			inner := resolve(sw.Sub)
			if len(inner) >= 3 && inner[0].pos == posNoun && allVerbWords(inner[1:]) {
				verbs := make([]*Verb, len(inner))
				verbs[0] = constVerb(inner[0].noun)
				for k, w := range inner[1:] {
					verbs[k+1] = w.verb
				}
				words = append(words, word{pos: posVerb, verb: foldVerbRun(verbs)})
			} else {
				inner = foldTrains(inner)
				if len(inner) == 1 && inner[0].pos == posVerb {
					words = append(words, inner[0])
				} else {
					words = append(words, word{pos: posNoun, noun: evalWords(inner)})
				}
			}
		case SynPrim:
			words = append(words, word{pos: posVerb, verb: primitives[sw.Text]})
		case SynName:
			// User-defined names: check verb namespace first, then noun namespace.
			if v, ok := verbGlobals[sw.Text]; ok {
				words = append(words, word{pos: posVerb, verb: v})
			} else if n, ok := globals[sw.Text]; ok {
				words = append(words, word{pos: posNoun, noun: n})
			} else {
				panic("unknown name: " + sw.Text)
			}
		case SynAssign:
			words = append(words, word{pos: posAssign, name: sw.Text})
		case SynAdverb:
			// "/" must immediately follow a verb.
			if len(words) == 0 || words[len(words)-1].pos != posVerb {
				panic("/ without preceding verb")
			}
			v := insertAdverb(words[len(words)-1].verb)
			words[len(words)-1] = word{pos: posVerb, verb: v}
		case SynConj:
			// `"` forms a rank conjunction: verb " rankArg → derived verb.
			// The rank argument is the next element; i++ here plus the loop's
			// own increment means we advance past both `"` and its argument.
			if len(words) == 0 || words[len(words)-1].pos != posVerb {
				panic(`" without preceding verb`)
			}
			i++
			var rankNoun *Array
			if i < len(sent) {
				switch sent[i].Kind {
				case SynNum:
					rankNoun = parseNumber(sent[i].Text)
				case SynGroup:
					rankNoun = evalWords(resolve(sent[i].Sub))
				}
			}
			if rankNoun == nil {
				panic(`" with no rank argument`)
			}
			ranks := parseRankArg(rankNoun)
			v := words[len(words)-1].verb
			words[len(words)-1] = word{pos: posVerb, verb: withRank(v, ranks[0], ranks[1], ranks[2])}
		}
	}
	return words
}

// --- evalWords: the J evaluation algorithm ---

// evalWords evaluates a flat resolved word list using J's right-to-left rule.
// All verbs have equal precedence; the leftmost verb is the principal verb
// because everything to its right is evaluated first (recursively).
func evalWords(words []word) *Array {
	words = foldTrains(words)
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
	allScalar, hasFloat := true, false
	for _, w := range words {
		if w.noun.rank() != 0 {
			allScalar = false
			break
		}
		if isFloat(w.noun) {
			hasFloat = true
		}
	}
	if allScalar {
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
