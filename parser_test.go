package cmdparse

import (
	"reflect"
	"testing"
)

func ensureTreesEqual(t *testing.T, exp, act interface{}) {
	if reflect.TypeOf(exp) != reflect.TypeOf(act) {
		t.Fatalf("In parse tree: expected type %T but found %T", exp, act)
	}

	switch e := exp.(type) {
	case alts:
		a := act.(alts)
		ensureSliceEqual(t, e.Children(), a.Children())
	case terms:
		a := act.(terms)
		ensureSliceEqual(t, e.Children(), a.Children())
	case rep:
		a := act.(rep)
		if e.Op != a.Op {
			t.Fatalf("In parse tree: expected Rep op to be %d but found %d", e.Op, a.Op)
		}
		ensureTreesEqual(t, e.Term, a.Term)
	case variable:
		a := act.(variable)
		if e.Name != a.Name || e.Type != e.Type {
			t.Fatalf("In parse tree: expected Var to be %s but found %s", e, a)
		}
	case word:
		a := act.(word)
		if string(e) != string(a) {
			t.Fatalf("In parse tree: expected Word to be %s but found %s", string(e), string(a))
		}
	case nil:
		if act != nil {
			t.Fatalf("In parse tree: expected nil but found %T", act)
		}
	default:
		t.Fatalf("In parse tree: unknown parse tree node type %T", e)
	}
}

func ensureSliceEqual(t *testing.T, exp, act []interface{}) {
	if len(exp) != len(act) {
		t.Fatalf("In parse tree: expected node to have %d children but found %d", len(exp), len(act))
	}
	for i := 0; i < len(exp); i++ {
		ensureTreesEqual(t, exp[i], act[i])
	}
}

func TestParser(t *testing.T) {

	tests := []struct {
		name     string
		input    string
		expected interface{}
		ok       bool
		error    string
	}{
		{
			name:     "empty",
			input:    "",
			expected: nil,
			ok:       true,
			error:    "",
		},
		{
			name:     "show",
			input:    "show",
			expected: word("show"),
			ok:       true,
			error:    "",
		},
		{
			name:  "show  this",
			input: "show  this",
			expected: terms{
				word("show"), word("this"),
			},
			ok:    true,
			error: "",
		},
		{
			name:  "show  this",
			input: "show  this",
			expected: terms{
				word("show"), word("this"),
			},
			ok:    true,
			error: "",
		},
		{
			name:  "do this*",
			input: "do this*",
			expected: terms{
				word("do"), rep{
					Op:   repeatZeroOrMore,
					Term: word("this")},
			},
			ok:    true,
			error: "",
		},
		{
			name:  "this | that",
			input: "this | that",
			expected: alts{
				word("this"),
				word("that"),
			},
			ok:    true,
			error: "",
		},
		{
			name:  "this | that | other",
			input: "this | that | other",
			expected: alts{
				word("this"),
				alts{
					word("that"),
					word("other"),
				},
			},
			ok:    true,
			error: "",
		},
		{
			// Here we test the precedence. The | operator should be lowest.
			name:  "this a | that b",
			input: "this a | that b",
			expected: alts{
				terms{
					word("this"),
					word("a"),
				},
				terms{
					word("that"),
					word("b"),
				},
			},
			ok:    true,
			error: "",
		},
		{
			name:  "this? that* <myValue:int>",
			input: "this? that* <myValue:int>",
			expected: terms{
				rep{Op: repeatZeroOrOne,
					Term: word("this")},
				terms{
					rep{Op: repeatZeroOrMore,
						Term: word("that")},
					variable{Name: "myValue",
						Type: "int"},
				},
			},
			ok:    true,
			error: "",
		},
		{
			name:  "(this that)",
			input: "(this that)",
			expected: terms{
				word("this"), word("that"),
			},

			ok:    true,
			error: "",
		},
		{
			name:  "(this that)*",
			input: "(this that)*",
			expected: rep{Op: repeatZeroOrMore,
				Term: terms{
					word("this"), word("that"),
				},
			},
			ok:    true,
			error: "",
		},
		{
			name:     "((this ))",
			input:    "((this ))",
			expected: word("this"),
			ok:       true,
			error:    "",
		},
		{
			name:  "<var1> <var2:int>",
			input: "<var1> <var2:int>",
			expected: terms{
				variable{Name: "var1",
					Type: "str"},
				variable{Name: "var2",
					Type: "int"},
			},
			ok:    true,
			error: "",
		},
		{
			name:  "word+word2",
			input: "word+word2",
			expected: terms{
				rep{Op: repeatOneOrMore,
					Term: word("word")},

				word("word2"),
			},
			ok:    true,
			error: "",
		},
		{
			name:  "get (<v>|all)",
			input: "get (<v>|all)",
			expected: terms{
				word("get"),
				alts{
					variable{Name: "v", Type: "str"},
					word("all"),
				},
			},
			ok:    true,
			error: "",
		},
		// Failures
		{
			name:     "this** extra repeat",
			input:    "this**",
			expected: nil,
			ok:       false,
			error:    "At character 6: extra tokens after end of command",
		},
		{
			name:     "this| ends with pipe",
			input:    "this|",
			expected: nil,
			ok:       false,
			error:    "At character 6: expected more tokens after the |",
		},
		{
			name:     "<   ",
			input:    "<   ",
			expected: nil,
			ok:       false,
			error:    "At character 2: expected variable name after <",
		},
		{
			name:     "<var",
			input:    "<var",
			expected: nil,
			ok:       false,
			error:    "At character 5: expected either : to specify variable type, or > to complete variable definition",
		},
		{
			name:     "<var :",
			input:    "<var :",
			expected: nil,
			ok:       false,
			error:    "At character 7: expected variable type after :",
		},
		{
			name:     "( word*",
			input:    "( word*",
			expected: nil,
			ok:       false,
			error:    "At character 8: expected ) to close the group",
		},
		{
			name:     "( word   *",
			input:    "( word   *",
			expected: nil,
			ok:       false,
			error:    "At character 11: expected ) to close the group",
		},
	}

	for _, tc := range tests {

		t.Run(tc.name, func(t *testing.T) {

			var s scanner
			toks, ok := s.Scan(tc.input)
			if !ok {
				t.Fatalf("Scan failed")
			}

			var p parser
			p.matchLimit = 100
			tree, err := p.Parse(toks)
			// Uncomment below to print the parse tree 
			/*
			fmt.Printf("test '%s': Parse tree returned:\n", tc.name)
			printTree(tree)
			*/

			if err != nil {
				if tc.ok {
					t.Fatalf("Parse failed when it should succeed. Error: %s", err)
				}

				if err.Error() != tc.error {
					t.Fatalf("Parse failed as expected, but with wrong error. Expected '%s' but got '%s'", tc.error, err.Error())
				}
				return
			}

			if err == nil && !tc.ok {
				t.Fatalf("Parse succeeded when it should have failed")
			}

			ensureTreesEqual(t, tc.expected, tree)
		})
	}

}
