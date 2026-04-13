package main

// SyntaxKind identifies the syntactic role of a word in a J sentence.
// These are purely lexical/grammatical categories — no J values are
// created during parsing, only during evaluation.
type SyntaxKind int

const (
	SynNum    SyntaxKind = iota // number literal;  Text = raw token  (e.g. "3", "_5")
	SynStr                      // string literal;   Text = content   (e.g. "hello")
	SynPrim                     // known primitive;  Text = spelling  (e.g. "+", "i.")
	SynName                     // unresolved name;  Text = name — resolved to verb or noun at eval time
	SynAdverb                   // adverb token;     Text = "/"
	SynConj                     // conjunction token; Text = `"`
	SynAssign                   // assignment target; Text = variable name (the =: has been consumed)
	SynGroup                    // parenthesized sub-sentence; Sub = inner Sentence
)

// SentenceWord is one word in a parsed J sentence.
// It carries only syntactic information: raw text and, for groups,
// the recursively parsed sub-sentence.  No *Array or *Verb is ever
// stored here — those are created by the evaluator.
type SentenceWord struct {
	Kind SyntaxKind
	Text string   // set for all kinds except SynGroup
	Sub  Sentence // set only for SynGroup
}

// Sentence is a flat list of syntactic words as produced by the parser.
//
// J's grammar is remarkably simple: evaluation is strictly right-to-left
// with no precedence.  Rather than building a tree, the parser produces
// this flat word list; the evaluator then applies two passes:
//   1. resolve — turn syntax words into live verbs/nouns and fold
//      adverb/conjunction patterns into derived verbs
//   2. evalWords — walk the resolved word list right-to-left
type Sentence = []SentenceWord
