package cmdparse

import (
	"bytes"
	"fmt"
	"testing"
)

type Comparer struct {
	exp, act prog
	t        *testing.T
}

func (c *Comparer) setT(t *testing.T) {
	c.t = t
}

func (c *Comparer) setCode(exp, act prog) {
	c.exp = exp
	c.act = act
}

func (c Comparer) ensureCodeEqual() {
	if len(c.exp) != len(c.act) {
		c.fail("Expected %d instructions but got %d", len(c.exp), len(c.act))
	}

	for i := 0; i < len(c.exp); i++ {
		c.ensureInstrEqual(c.exp[i], c.act[i])
	}
}

func (c Comparer) fail(f string, args ...interface{}) {
	m := fmt.Sprintf(f, args...)
	p := c.progsToStr()
	c.t.Fatalf("%s\n%s", m, p)
}

func (c Comparer) progsToStr() string {
	var buf bytes.Buffer
	buf.WriteString("programs:\n")
	buf.WriteString("expected:\n")
	c.exp.Print(&buf)
	buf.WriteString("actual:\n")
	c.act.Print(&buf)
	return buf.String()
}

func (c Comparer) ensureInstrEqual(exp, act instr) {
	if exp.opcode != act.opcode {
		c.fail("Expected opcode %s but got opcode %s",
			exp.opcode, act.opcode)
	}
}

func TestCompiler(t *testing.T) {

	tests := []struct {
		name     string
		input    interface{}
		expected []instr
	}{
		{
			name:  "show",
			input: word("show"),
			expected: prog{
				instr{opcode: opCmp, strs: [2]string{"show"}},
				instr{opcode: opMatch},
			},
		},
		{
			name: "this | that",
			input: alts{
				word("this"),
				word("that"),
			},
			expected: prog{
				instr{opcode: opSplit, ints: [2]int{1, 3}},
				instr{opcode: opCmp, strs: [2]string{"this"}},
				instr{opcode: opJmp, ints: [2]int{4}},
				instr{opcode: opCmp, strs: [2]string{"that"}},
				instr{opcode: opMatch},
			},
		},
		{
			name: "this | that | other",
			input: alts{
				word("this"),
				alts{
					word("that"),
					word("other"),
				},
			},
			expected: prog{
				instr{opcode: opSplit, ints: [2]int{1, 3}},
				instr{opcode: opCmp, strs: [2]string{"this"}},
				instr{opcode: opJmp, ints: [2]int{7}},

				instr{opcode: opSplit, ints: [2]int{4, 6}},
				instr{opcode: opCmp, strs: [2]string{"that"}},
				instr{opcode: opJmp, ints: [2]int{7}},
				instr{opcode: opCmp, strs: [2]string{"other"}},

				instr{opcode: opMatch},
			},
		},
		{
			name: "this*",
			input: rep{
				Term: word("this"),
				Op:   repeatZeroOrMore,
			},
			expected: prog{
				instr{opcode: opSplit, ints: [2]int{1, 3}},
				instr{opcode: opCmp, strs: [2]string{"this"}},
				instr{opcode: opJmp, ints: [2]int{0}},

				instr{opcode: opMatch},
			},
		},
		{
			name: "(this | that)?",
			input: rep{
				Op: repeatZeroOrOne,
				Term: alts{
					word("this"),
					word("that"),
				},
			},
			expected: prog{
				instr{opcode: opSplit, ints: [2]int{1, 5}},

				instr{opcode: opSplit, ints: [2]int{2, 4}},
				instr{opcode: opCmp, strs: [2]string{"this"}},
				instr{opcode: opJmp, ints: [2]int{5}},
				instr{opcode: opCmp, strs: [2]string{"that"}},

				instr{opcode: opMatch},
			},
		},
		{
			name: "a+",
			input: rep{
				Op:   repeatOneOrMore,
				Term: word("a"),
			},
			expected: prog{
				instr{opcode: opCmp, strs: [2]string{"a"}},
				instr{opcode: opSplit, ints: [2]int{0, 2}},

				instr{opcode: opMatch},
			},
		},
		{
			name: "get hat",
			input: terms{
				Left:  word("get"),
				Right: word("hat"),
			},
			expected: prog{
				instr{opcode: opCmp, strs: [2]string{"get"}},
				instr{opcode: opCmp, strs: [2]string{"hat"}},

				instr{opcode: opMatch},
			},
		},
		{
			name: "get <var>",
			input: terms{
				Left:  word("get"),
				Right: variable{"var", "string"},
			},
			expected: prog{
				instr{opcode: opCmp, strs: [2]string{"get"}},
				instr{opcode: opSave, strs: [2]string{"var", "string"}},

				instr{opcode: opMatch},
			},
		},
	}

	for _, tc := range tests {

		t.Run(tc.name, func(t *testing.T) {

			var c compiler

			c.compile(tc.input)
			prog := c.prog()
			
			// Uncomment below to see a dump of the program instructions
			/*
			fmt.Printf("Program is\n")
			prog.Print(os.Stdout)
			*/

			var cmp Comparer
			cmp.setT(t)
			cmp.setCode(tc.expected, prog)
			cmp.ensureCodeEqual()

		})
	}

}
