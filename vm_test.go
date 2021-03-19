package cmdparse

import (
	"bytes"
	"fmt"
	"testing"
)

func TestVmSwap(t *testing.T) {
	var v vm

	t1 := thread{pc: 1}
	t2 := thread{pc: 2}
	t3 := thread{pc: 3}
	t4 := thread{pc: 4}

	l1 := threadList([]*thread{&t1, &t2})
	l2 := threadList([]*thread{&t3, &t4})

	v.currentThreads = &l1
	v.nextThreads = &l2

	v.swap(v.currentThreads, v.nextThreads)

	pc := (*v.currentThreads)[0].pc
	if pc != 3 {
		t.Fatalf("Swap failed: expected element 0 to be %d but was %d ", 3, pc)
	}

	pc = (*v.nextThreads)[0].pc
	if pc != 1 {
		t.Fatalf("Swap failed: expected element 0 to be %d but was %d ", 1, pc)
	}
}

func TestVmClear(t *testing.T) {
	var v vm

	t1 := thread{pc: 1}
	t2 := thread{pc: 2}

	l1 := threadList([]*thread{&t1, &t2})

	v.currentThreads = &l1

	v.clear(v.currentThreads)

	if len(*v.currentThreads) != 0 {
		t.Fatalf("Clear failed")
	}
}

func TestVmAddThread(t *testing.T) {
	var v vm
	v.makeThreadLists()
	v.prog = make([]instr, 10)
	v.gen = 1
	v.addThread(v.currentThreads, &thread{pc: 0})

	if (*v.currentThreads)[0] == nil {
		t.Fatalf("Added a thread to list, but it's nil")
	}
}

func TestVm(t *testing.T) {

	tests := []struct {
		name     string
		syntax   string
		input    []string
		valid    bool
		expected []match
	}{
		{
			name:   "show",
			syntax: "show",
			input:  []string{"show"},
			valid:  true,
			expected: []match{
				{items: []interface{}{keywordValue{"show", "show"}}},
			},
		},
		{
			name:   "show something",
			syntax: "show",
			input:  []string{"show", "something"},
			valid:  false,
			expected: []match{
				{items: []interface{}{keywordValue{"show", "show"}}},
			},
		},
		{
			name:   "show | tell",
			syntax: "show | tell",
			input:  []string{"te"},
			valid:  true,
			expected: []match{
				{items: []interface{}{keywordValue{"tell", "te"}}},
			},
		},
		{
			name:   "get hat",
			syntax: "get hat",
			input:  []string{"get", "hat"},
			valid:  true,
			expected: []match{
				{items: []interface{}{keywordValue{"get", "get"},
					keywordValue{"hat", "hat"}}},
			},
		},
		{
			name:   "get <file> verbose?",
			syntax: "get <file> verbose?",
			input:  []string{"get", "a.html"},
			valid:  true,
			expected: []match{
				{items: []interface{}{keywordValue{"get", "get"},
					VarValue{"file", "str", "a.html"}}},
			},
		},
		{
			name:   "get <file> verbose?",
			syntax: "get <file> verbose?",
			input:  []string{"get", "a.html", "v"},
			valid:  true,
			expected: []match{
				{items: []interface{}{keywordValue{"get", "get"},
					VarValue{"file", "str", "a.html"},
					keywordValue{"verbose", "v"}}},
			},
		},
		{
			// Ambiguous matches: the v can match the file variable and also the verbose? keyword
			// We detect both so that the user can report an ambiguety and the user can use something other than
			// v as the name
			name:   "get <file>* verbose? with get v",
			syntax: "get <file>* verbose?",
			input:  []string{"get", "v"},
			valid:  true,
			expected: []match{
				{items: []interface{}{keywordValue{"get", "get"},
					keywordValue{"verbose", "v"}}},
				{items: []interface{}{keywordValue{"get", "get"},
					VarValue{"file", "str", "v"}}},
			},
		},
		{
			// In possible matches, prefer the leftmost match first.
			name:   "many commands 1",
			syntax: "(do (thing|<v>)) | (add <n>*) | (clear logs?)",
			input:  []string{"do", "thing"},
			valid:  true,
			expected: []match{
				{items: []interface{}{keywordValue{"do", "do"},
					VarValue{"v", "str", "thing"}}},
				{items: []interface{}{keywordValue{"do", "do"},
					keywordValue{"thing", "thing"}}},
			},
		},
		{
			// In possible matches, prefer the leftmost match first.
			name:   "many commands 2",
			syntax: "(do (thing|<v>)) | (add <n:int>*) | (clear logs?)",
			input:  []string{"a", "1", "2", "3"},
			valid:  true,
			expected: []match{
				{items: []interface{}{keywordValue{"add", "a"},
					VarValue{"n", "int", "1"},
					VarValue{"n", "int", "2"},
					VarValue{"n", "int", "3"}},
				},
			},
		},
	}

	for _, tc := range tests {

		t.Run(tc.name, func(t *testing.T) {

			var s scanner
			tokens, ok := s.Scan(tc.syntax)
			if !ok {
				t.Fatalf("Scanning failed: %v", s.errs)
			}

			var p parser
			ptree, err := p.Parse(tokens)
			if err != nil {
				t.Fatalf("Parsing failed: %v", err)
			}

			var c compiler
			c.compile(ptree)
			prog := c.prog()

			// Uncomment below to see a dump of the program instructions
			/*
			fmt.Printf("Program is\n")
			prog.Print(os.Stdout)
			*/

			var v vm
			// Uncomment below to see an execution trace of the tests
			//v.traceWriter = os.Stdout
			v.execute(prog, tc.input)

			/*
			fmt.Printf("all matches: %v\n", v.matches)
			fmt.Printf("full matches: %v\n", v.maximalMatches())
			fmt.Printf("longest matches: %v\n", v.longestMatches())
			*/

			var comp MatchComparer
			comp.setT(t)
			comp.setProg(prog)
			if tc.valid {
				matches := v.maximalMatches()
				comp.setMatches(tc.expected, matches)
			} else {
				matches := v.longestMatches()
				comp.setMatches(tc.expected, matches)
			}
			comp.ensureMatchesEqual()
		})
	}

}

type MatchComparer struct {
	exp, act []match
	t        *testing.T
	prog     prog
}

func (c *MatchComparer) setT(t *testing.T) {
	c.t = t
}

func (c *MatchComparer) setMatches(exp, act []match) {
	c.exp = exp
	c.act = act
}

func (c *MatchComparer) setProg(p prog) {
	c.prog = p
}

func (c MatchComparer) ensureMatchesEqual() {
	if len(c.exp) != len(c.act) {
		c.fail("Expected match %v but actual match is %v", c.exp, c.act)
	}

	for i := 0; i < len(c.exp); i++ {
		c.ensureMatchEqual(c.exp[i], c.act[i])
	}
}

func (c MatchComparer) ensureMatchEqual(exp, act match) {
	if len(exp.items) != len(act.items) {
		c.fail("Expected match %v but actual match is %v", c.exp, c.act)
	}

	for i := range exp.items {
		c.ensureItemEqual(exp.items[i], act.items[i])
	}
}

func (c MatchComparer) ensureItemEqual(exp, act interface{}) {
	switch expReal := exp.(type) {
	case VarValue:
		actReal, ok := act.(VarValue)
		if !ok {
			c.fail("Expected match %v but actual match is %v", c.exp, c.act)
		}
		if expReal != actReal {
			c.fail("Expected match %v but actual match is %v", c.exp, c.act)
		}
	}
}

func (c MatchComparer) fail(f string, args ...interface{}) {
	m := fmt.Sprintf(f, args...)
	p := c.progToStr()
	c.t.Fatalf("%s\n%s", m, p)
}

func (c MatchComparer) progToStr() string {
	var buf bytes.Buffer
	buf.WriteString("program:\n")
	c.prog.Print(&buf)
	return buf.String()
}
