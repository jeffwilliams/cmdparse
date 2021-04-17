package cmdparse

import (
	"bytes"
	"io"
	"unicode"
)

// Cmds is used to register callbacks for command definitions and to parse input 
// to match a registered command.
//
// To use Cmds, first call Cmds.Add multiple times to register command definitions and their 
// callbacks, then call Cmds.Compile to compile the parsing VM. Now it's ready to parse 
// user-entered commands. Next call Parse at will to parse a command, and call LongestMatches
// after a failing Parse if needed.
//
// Command definitions use a simple grammar to define the syntax (keywords and variables)
// in the command. The grammar for the command definitions is:
//
//    command → alternatives EOF
//    alternatives → terms ( '|' alternatives )?
//    terms → repetition ( terms )?
//    repetition → group (  '*' |  '+' |  '?' )?
//    group → '(' alternatives ')' | term
//    term → var | WORD
//    var → '<' WORD (':' WORD)? '>'
//
// For example the following syntax defines a command that would match ‘load’, ‘load file.txt’, and ‘load file.txt other.txt’:
//
//    load <file>*
// 
// If a command is matched the command handler is called with the match from which it can extract
// the matched variables. 
type Cmds struct {
	parseTree interface{}
	prog      prog
	trace     io.Writer
}


// Add registers the command definition ‘cmd’. When this command is matched, the 
// callback ‘cback’ is called.
func (c *Cmds) Add(cmd string, cback Callback) error {
	// Each command that Add is passed is added as a branch in an alternative (alt)
	// at the top level of a parse tree. After all the commands are added we have a
	// parse tree that represents that any of the commands can cause a match:
	//      command1 | command2 | ...
	// Just below each alternative we add a metadata node that contains the callback
	// to call if that command is matched. That metadata node when compiled updates
	// the metadata register stored in the thread.

	t, err := c.scanAndParse(cmd)
	if err != nil {
		return err
	}

	c.addParseTree(t, cback)

	return nil
}

func (c *Cmds) scanAndParse(cmd string) (tree interface{}, err error) {
	var s scanner
	tokens, ok := s.Scan(cmd)
	if !ok {
		err = ScanError(s.errs)
		return
	}

	var p parser
	tree, err = p.Parse(tokens)
	return
}

func (c *Cmds) addParseTree(tree interface{}, cback Callback) {
	var m meta
	m.ch = tree
	m.data = cback
	if c.parseTree == nil {
		c.parseTree = m
	} else {
		c.parseTree = alts{Left: m, Right: c.parseTree}
	}
}

// Callback is a function that gets called when Cmds.Parse succeeds. It is called with 
// a Match representing the parsed command.
type Callback func(match Match, ctx interface{})

// Match is used to find out what keywords and variables were matched on the command when 
// Cmds.Parse was called.
type Match interface {
	// Var returns the type and value of all variables with the name ‘name’ that was filled in 
	// from the command. If no variables were found that match the name an empty slice
	// is returned.
	Var(name string) (value []*VarValue)
	// KeywordPresent retuurns true if the keyword ‘name’ was entered in the input.
	KeywordPresent(name string) bool
}



// meta is used as a node in the parse tree that applies metadata to it's child
type meta struct {
	data interface{}
	ch   interface{}
}

type ScanError []error

func (s ScanError) Error() string {
	var buf bytes.Buffer
	for _, e := range s {
		buf.WriteString(e.Error())
		buf.WriteRune('\n')
	}
	return buf.String()
}

// Compile the registered commands into a VM.
func (c *Cmds) Compile() {
	var cmp compiler
	cmp.compile(c.parseTree)
	c.prog = cmp.prog()
	return
}

// TraceExecutionTo sets the Writer to which execution logs are printed
// when Parse is called. 
func (c *Cmds) TraceExecutionTo(w io.Writer) {
	c.trace = w
}

// Parse attempts to parse the user-entered text ‘cmd’. If the input matches one of 
// the commands registered by Add it returns true.
func (c *Cmds) Parse(cmd string, ctx interface{}) (ok bool) {
	var s cmdScanner
	toks := s.Scan(cmd)

	var v vm
	v.traceWriter = c.trace
	v.execute(c.prog, toks)

	if len(v.maximalMatches()) != 1 {
		return false
	}

	mm := v.maximalMatches()[0]
	cback := mm.meta.(Callback)
	cback(cmdMatch(mm), ctx)

	return true
}

type cmdMatch match

func (c cmdMatch) Var(name string) (value []*VarValue) {
	value = make([]*VarValue, 0)
	for _, w := range c.items {
		if v, b := w.(VarValue); b {
			if v.Name == name {
				value = append(value, &v)
			}
		}	
	}
	return
}

func (c cmdMatch) KeywordPresent(name string) bool {
	for _, w := range c.items {
		if v, b := w.(keywordValue); b {
			if v.Name == name {
				return true
			}
		}	
	}
	return false
}
	
// Return the longest — not necessarily maximal —matches after Parse was called. This method is useful in case Parse couldn't find a single longest match (it returned false) so that the caller can look at all matches to attempt to print a helpful error message.
func (c *Cmds) LongestMatches() {
}

type cmdScanner struct {
	runes []rune
	word  bytes.Buffer
	words []string
}

func (t *cmdScanner) Scan(command string) []string {
	t.runes = []rune(command)
	t.innerTokenize()
	return t.words
}

func (t *cmdScanner) innerTokenize() {
	const (
		Default = iota
		InWord
		WaitingForTerminator
	)

	var state = Default
	var terminator rune
	for _, r := range t.runes {
		switch state {
		case Default:
			if !unicode.IsSpace(r) {
				if r == '"' {
					state = WaitingForTerminator
					terminator = '"'
					continue
				}

				t.addRuneToWord(r)
				state = InWord
			}
		case InWord:
			if unicode.IsSpace(r) {
				t.addWord()
				state = Default
				continue
			}
			t.addRuneToWord(r)
		case WaitingForTerminator:
			if r == terminator {
				t.addWord()
				state = Default
				continue
			}
			t.addRuneToWord(r)
		}
	}

	if !t.wordIsEmpty() {
		t.addWord()
	}
}

func (t *cmdScanner) addRuneToWord(r rune) {
	t.word.WriteRune(r)
}

func (t *cmdScanner) wordIsEmpty() bool {
	return t.word.Len() == 0
}

func (t *cmdScanner) addWord() {
	t.words = append(t.words, t.word.String())
	t.word.Reset()
}
