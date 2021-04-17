package cmdparse

import (
	"fmt"
	"io"
)

/*
instructions:

Alts: compile a split instruction: continue at all addresses
Terms: concatenate all the instructions of the subterms
Rep: split
Word: match: take the current input word w and see if it is a prefix of the Word token
Var: collect the field into a list for that varname

*/

type compiler struct {
	instr prog
	pc    int
}

type prog []instr

func (c *compiler) compile(ptree interface{}) {
	if ptree == nil {
		return
	}
	c.instr = make([]instr, c.countinstrForProgram(ptree))
	c.emit(ptree)
	c.emitMatch()
}

func (c *compiler) prog() prog {
	return c.instr
}

func (c compiler) countinstrForProgram(ptree interface{}) int {
	// Add 1 for the final opMatch
	return c.countinstr(ptree) + 1
}

func (c compiler) countinstr(ptree interface{}) int {
	switch node := ptree.(type) {
	case alts:
		return 2 + c.countinstr(node.Left) + c.countinstr(node.Right)
	case word:
		return 1
	case variable:
		return 1
	case rep:
		switch node.Op {
		case repeatZeroOrMore:
			return 2 + c.countinstr(node.Term)
		case repeatOneOrMore:
			return c.countinstr(node.Term) + 1
		case repeatZeroOrOne:
			return 1 + c.countinstr(node.Term)
		}
	case terms:
		return c.countinstr(node.Left) + c.countinstr(node.Right)
	case meta:
		return 1 + c.countinstr(node.ch)
	default:
		panic(fmt.Sprintf("Compiler.countinstr: unknown node type %T in parse tree", node))
	}
	return 0
}

func (c *compiler) emit(ptree interface{}) {
	switch node := ptree.(type) {
	case alts:
		c.emitAlts(node)
	case word:
		c.emitWord(node)
	case variable:
		c.emitVar(node)
	case terms:
		c.emitTerms(node)
	case rep:
		c.emitRep(node)
	case meta:
		c.emitMeta(node)
	default:
		panic(fmt.Sprintf("Compiler.emit: unknown node type %T in parse tree", node))
	}
}

func (c *compiler) emitAlts(a alts) {
	split := &c.instr[c.pc]
	split.opcode = opSplit
	split.ints[0] = c.pc + 1
	c.pc++
	c.emit(a.Left)

	jmp := &c.instr[c.pc]
	jmp.opcode = opJmp
	c.pc++

	split.ints[1] = c.pc
	c.emit(a.Right)
	jmp.ints[0] = c.pc
}

func (c *compiler) emitWord(w word) {
	c.instr[c.pc].opcode = opCmp
	c.instr[c.pc].strs[0] = string(w)
	c.pc++
}

func (c *compiler) emitVar(v variable) {
	c.instr[c.pc].opcode = opSave
	c.instr[c.pc].strs[0] = v.Name
	c.instr[c.pc].strs[1] = v.Type
	c.pc++
}

func (c *compiler) emitTerms(t terms) {
	c.emit(t.Left)
	c.emit(t.Right)
}

func (c *compiler) emitRep(r rep) {
	switch r.Op {
	case repeatZeroOrMore:
		c.emitZeroOrMore(r)
	case repeatOneOrMore:
		c.emitOneOrMore(r)
	case repeatZeroOrOne:
		c.emitZeroOrOne(r)
	}
}

func (c *compiler) emitZeroOrMore(r rep) {
	splitNdx := c.pc
	split := &c.instr[c.pc]
	split.opcode = opSplit
	split.ints[0] = c.pc + 1
	c.pc++

	c.emit(r.Term)

	jmp := &c.instr[c.pc]
	jmp.opcode = opJmp
	jmp.ints[0] = splitNdx
	c.pc++

	split.ints[1] = c.pc
}

func (c *compiler) emitOneOrMore(r rep) {
	splitDst1 := c.pc

	c.emit(r.Term)

	split := &c.instr[c.pc]
	split.opcode = opSplit
	split.ints[0] = splitDst1
	split.ints[1] = c.pc + 1
	c.pc++
}

func (c *compiler) emitZeroOrOne(r rep) {
	split := &c.instr[c.pc]
	split.opcode = opSplit
	split.ints[0] = c.pc + 1
	c.pc++

	c.emit(r.Term)

	split.ints[1] = c.pc
}

func (c *compiler) emitMatch() {
	c.instr[c.pc].opcode = opMatch
	c.pc++
}

func (c *compiler) emitMeta(m meta) {
	c.instr[c.pc].opcode = opMeta
	c.instr[c.pc].intf = m.data
	c.pc++

	c.emit(m.ch)
}

func (c compiler) printinstr(w io.Writer) {
	c.instr.Print(w)
}

type opcode int

const (
	opNop opcode = iota
	opSplit
	opJmp
	opCmp   // Compare current token against a keyword
	opSave  // Save the value of the current token as a variable. NOTE: this is different from Russ Cox' code!
	opMeta  // Set the metadata for the current thread
	opMatch // All done, we matched the command
)

func (o opcode) String() string {
	switch o {
	case opNop:
		return "nop"
	case opSplit:
		return "split"
	case opJmp:
		return "jmp"
	case opCmp:
		return "cmp"
	case opMatch:
		return "match"
	case opSave:
		return "save"
	case opMeta:
		return "meta"
	}
	return "unknown"
}

func (o opcode) NumArgs() int {
	switch o {
	case opSplit, opSave:
		return 2
	case opJmp, opCmp:
		return 1
	case opMeta:
		return 1
	default:
		return 0
	}
}

func (o opcode) Arg(n *instr, i int) interface{} {

	switch o {
	case opNop, opMatch:
		return nil
	case opSplit, opJmp:
		return n.ints[i]
	case opCmp, opSave:
		return "'" + n.strs[i] + "'"
	case opMeta:
		return n.intf
	}
	return nil
}

type instr struct {
	opcode opcode
	ints   [2]int
	strs   [2]string
	intf   interface{}
	// gen is the generation of the instruction (set and used by the VM when executing)
	gen int
}

func (i instr) String() string {
	s := fmt.Sprintf("%s ", i.opcode)
	for j := 0; j < i.opcode.NumArgs(); j++ {
		if j > 0 {
			s += ", "
		}
		s += fmt.Sprintf("%3v", i.opcode.Arg(&i, j))
	}

	return s
}

func (p prog) Print(w io.Writer) {
	for i, instr := range p {
		fmt.Fprintf(w, "%3d: %s\n", i, instr)
	}
}
