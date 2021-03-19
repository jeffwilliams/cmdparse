package cmdparse

import (
	"fmt"
	"os"
	"runtime/debug"
)

/*

command → alternatives EOF
alternatives → terms ( '|' alternatives )?
terms → repetition ( terms )?
repetition → group (  '*' |  '+' |  '?' )?
group → '(' alternatives ')' | term
term → var | WORD
var → '<' WORD (':' WORD)? '>'

Notes:
	• If unspecified, a variable's type is str

*/

// Recursive Descent parser
// https://craftinginterpreters.com/parsing-expressions.html

type parser struct {
	tokens  []token
	errors  Errors
	current int

	// For debugging
	matchLimit int
	matchCalls int
}

func (p *parser) Parse(tokens []token) (tree interface{}, err error) {
	p.tokens = tokens
	p.errors = newErrors()
	p.current = 0
	return p.parse()
}

func (p *parser) parse() (tree interface{}, err error) {
	tree = p.Command()

	if !p.atEnd() {
		p.addErrorAtPosition("extra tokens after end of command")
	}

	err = p.errors.nilIfEmpty()
	return
}

func (p *parser) Command() interface{} {
	return p.Alternatives()
}

func (p *parser) Alternatives() interface{} {
	l := p.Terms()
	var r interface{}

	if p.match(pipeTok) {
		r = p.Alternatives()
		if r == nil {
			p.addErrorAtPosition("expected more tokens after the |")
		}
	}

	if r == nil {
		return l
	}

	return alts{Left: l, Right: r}
}

func (p *parser) Terms() interface{} {
	l := p.Repetition()
	if l == nil {
		return nil
	}
	var r interface{}

	if !p.atEnd() {
		r = p.Terms()
	}

	if r == nil {
		return l
	}

	return terms{Left: l, Right: r}
}

func (p *parser) Repetition() interface{} {
	t := p.Group()
	if t == nil {
		return nil
	}

	r := rep{Term: t}

	if p.match(starTok, plusTok, questionTok) {
		switch p.previous().tokenType() {
		case starTok:
			r.Op = repeatZeroOrMore
		case plusTok:
			r.Op = repeatOneOrMore
		case questionTok:
			r.Op = repeatZeroOrOne
		}
	} else {
		return r.Term
	}

	return r
}

func (p *parser) Group() interface{} {
	if p.match(leftParenTok) {
		res := p.Alternatives()

		if !p.match(rightParenTok) {
			p.addErrorAtPosition("expected ) to close the group")
		}

		return res
	}

	return p.Term()
}

func (p *parser) Term() interface{} {
	r := p.Var()
	if r == nil {
		r = p.Word()
	}
	return r
}

func (p *parser) Var() interface{} {
	if !p.match(lessThanTok) {
		return nil
	}

	name := p.Word()
	if name == nil {
		p.addErrorAtPosition("expected variable name after <")
		return nil
	}

	var typ string
	hasColon := true
	if !p.match(colonTok) {
		typ = "str"
		hasColon = false
	} else {
		w := p.Word()

		if w == nil {
			p.addErrorAtPosition("expected variable type after :")
			return nil
		}
		typ = string(w.(word))
	}

	if !p.match(greaterThanTok) {
		if hasColon {
			p.addErrorAtPosition("expected > to complete variable definition")
		} else {
			p.addErrorAtPosition("expected either : to specify variable type, or > to complete variable definition")
		}
		return nil
	}

	return variable{string(name.(word)), typ}
}

func (p *parser) Word() interface{} {
	if !p.match(wordTok) {
		return nil
	}
	return word(p.previous().value)
}

func (p *parser) match(types ...tokenType) bool {

	if p.matchLimit > 0 {
		p.matchCalls++
		if p.matchCalls > p.matchLimit {
			p.abortAndPrintState()
		}
	}

	for _, t := range types {
		if p.check(t) {
			p.advance()
			return true
		}
	}
	return false
}

func (p *parser) check(typ tokenType) bool {
	if p.atEnd() {
		return false
	}
	return p.peek().tokenType() == typ
}

func (p *parser) advance() token {
	if !p.atEnd() {
		p.current++
	}
	return p.previous()
}

func (p *parser) peek() token {
	return p.tokens[p.current]
}

func (p *parser) previous() token {
	return p.tokens[p.current-1]
}

func (p *parser) atEnd() bool {
	return p.current >= len(p.tokens)
}

func (p *parser) position() int {
	return p.current
}

func (p *parser) runePosition() int {
	if p.current == 0 {
		return 1
	}

	return p.previous().pos + p.previous().len()
}

func (p *parser) addError(e error) {
	p.errors.add(e)
}

func (p *parser) addErrorAtPosition(msg string) {
	p.addError(fmt.Errorf("At character %d: %s", p.runePosition()+1, msg))
}

func (p *parser) abortAndPrintState() {
	fmt.Fprintf(os.Stderr, "Aborting\n")
	fmt.Fprintf(os.Stderr, "Tokens: %v\n", p.tokens)
	tok := "<at end>"
	if !p.atEnd() {
		tok = fmt.Sprintf("%s", p.tokens[p.current])
	}
	fmt.Fprintf(os.Stderr, "trying to match: %d (%s)\n", p.current, tok)
	debug.PrintStack()
	panic("Abort")
}

type alts struct {
	Left, Right interface{}
}

func (a alts) String() string {
	return "alternatives"
}

func (a alts) Children() []interface{} {
	return []interface{}{a.Left, a.Right}
}

type terms struct {
	Left, Right interface{}
}

func (a terms) String() string {
	return "terms"
}

func (a terms) Children() []interface{} {
	return []interface{}{a.Left, a.Right}
}

type rep struct {
	Op   repOp
	Term interface{}
}

func (a rep) String() string {
	return "repetition"
}

func (a rep) Children() []interface{} {
	return []interface{}{a.Term}
}

type repOp int

const (
	repeatUnset repOp = iota
	repeatZeroOrMore
	repeatOneOrMore
	repeatZeroOrOne
)

func (r repOp) String() string {
	switch r {
	case repeatUnset:
		return "<unset>"
	case repeatZeroOrMore:
		return "*"
	case repeatOneOrMore:
		return "+"
	case repeatZeroOrOne:
		return "?"
	default:
		return "<unknown>"
	}
}

type word string

func (w word) String() string {
	return `"` + string(w) + `"`
}

func (w word) Children() []interface{} {
	return nil
}

type variable struct {
	Name string
	Type string
}

func (v variable) String() string {
	return v.Name + ":" + v.Type
}

func (v variable) Children() []interface{} {
	return nil
}

type childrener interface {
	Children() []interface{}
}

func printTree(tree interface{}) {
	printTreeInner(tree, 0)
}

func printTreeInner(tree interface{}, depth int) {
	var s string
	v, ok := tree.(fmt.Stringer)
	if ok {
		s = v.String()
	} else {
		s = fmt.Sprintf("non-stringer type %T", v)
	}

	for i := 0; i < depth*2; i++ {
		fmt.Printf(" ")
	}

	fmt.Printf("%s\n", s)

	c, ok := v.(childrener)
	if ok {
		for _, ch := range c.Children() {
			printTreeInner(ch, depth+1)
		}
	}
}
