package cmdparse

import (
	"fmt"
	"io"
	"strings"
)

// TODO: instead of matchedThreads, make a slice of match structs. Each has the bindings,
// and also how far into the input they matched. For initial prefix matches that will allow the user
// to print an error message indicating where more input is needed.

type vm struct {
	prog  prog
	input []string
	// currentThreads are the threads to run this iteration
	currentThreads *threadList
	// nextThreads are the threads to run next iteration
	nextThreads *threadList
	// List of threads that matched in the last iteration
	matches []match
	// gen is the current generation, used to tell if we already added a thread to one of the
	// thread lists
	gen int

	// thread is the currently executing thread
	thread *thread

	wordIndex int

	traceWriter io.Writer
}

type threadList []*thread

type thread struct {
	pc int
	// items are the sequence of matched keywords or variables
	items []binding

	meta interface{}
}

func (t thread) clone() *thread {
	var t2 thread
	t2.pc = t.pc
	t2.meta = t.meta
	t2.items = make([]binding, len(t.items))
	copy(t2.items, t.items)
	return &t2
}

func (t *thread) setPc(pc int) *thread {
	t.pc = pc
	return t
}

func (t *thread) bind(instr *instr, val *string) {
	if t.items == nil {
		t.items = make([]binding, 1, 10)
		t.items[0].instr = instr
		t.items[0].val = val
	} else {
		t.items = append(t.items, binding{instr, val})
	}
}

type match struct {
	items []interface{}
	meta  interface{}
}

type VarValue struct {
	Name  string
	Type  string
	Value string
}

type keywordValue struct {
	Name  string
	Value string
}

// binding is a binding of a keyword to the value the user entered for it,
// or a variable name and type to the value the user entered.
// The pointer to an instruction defines the keyword or name and type of the variable,
// and the string pointer points to the input word that represents the value
type binding struct {
	instr *instr
	val   *string
}

// input are the space-separated words of the command the user entered, split on spaces.
func (v *vm) execute(prog prog, input []string) {
	v.prog = prog
	v.input = input

	v.makeThreadLists()
	v.matches = make([]match, 0, 10)

	v.gen = 1

	v.addThread(v.currentThreads, &thread{pc: 0})
	for v.wordIndex = range input {
		v.processWord(&input[v.wordIndex])
	}
	v.processWord(nil)
	v.finishThreads()

}

func (v *vm) makeThreadLists() {
	l := make(threadList, 0, len(v.prog))
	v.currentThreads = &l
	l2 := make(threadList, 0, len(v.prog))
	v.nextThreads = &l2
}

func (v *vm) processWord(word *string) {

	v.gen++
	// New threads may get appended to the currentThreads while we are iterating it
	// Thus we use an index-based iteration.
	for i := 0; i < len(*v.currentThreads); i++ {

		v.thread = (*v.currentThreads)[i]
		v.continu(word)
	}

	v.swap(v.currentThreads, v.nextThreads)
	v.clear(v.nextThreads)
}

func (v *vm) finishThreads() {
	// We need to continue the threads one last time since on the final word of the input
	// the thread completes executing the final opCmp or opSave instruction, but is then
	// added to the nextThreads list and so doesn't execute the final opMatch instruction.
	// This final continue, with no word, lets it execute that last instruction.
	v.processWord(nil)
}

func (v *vm) continu(word *string) {
	// In this function some instructions append the thread to the currentThreads list
	// which means they will continue executing instructions on this current iteration (this
	// current input word). Some are instead added to the nextThreads list because they
	// must match the next word only.

	instr := v.currentinstr()
	v.trace()
	switch instr.opcode {
	case opNop:
		return
	case opJmp:
		v.doJmp(instr)
	case opMatch:
		v.addMatch(v.thread)
	case opSplit:
		v.doSplit(instr)
	case opCmp:
		v.doCmp(instr, word)
	case opSave:
		v.doSave(instr, word)
	case opMeta:
		v.doMeta(instr)
	default:
		panic(fmt.Sprintf("Unknown instruction %v", instr))
	}
}

func (v *vm) doJmp(instr *instr) {
	v.thread.pc = instr.ints[0]
	v.addThread(v.currentThreads, v.thread)
}

func (v *vm) doSplit(instr *instr) {
	t2 := v.thread.clone().setPc(instr.ints[1])
	v.thread.pc = instr.ints[0]
	v.addThread(v.currentThreads, v.thread)
	v.addThread(v.currentThreads, t2)
}

func (v *vm) doCmp(instr *instr, word *string) {
	if word != nil && strings.HasPrefix(instr.strs[0], *word) {
		v.thread.bind(instr, word)
		v.traceBind()
		v.thread.pc++
		v.addThread(v.nextThreads, v.thread)
	}
}

func (v *vm) doSave(instr *instr, word *string) {
	if word != nil {
		v.thread.bind(instr, word)
		v.traceBind()
		v.thread.pc++
		v.addThread(v.nextThreads, v.thread)
	}
}

func (v *vm) doMeta(instr *instr) {
	v.thread.meta = instr.intf
	v.thread.pc++
	v.addThread(v.currentThreads, v.thread)
}

func (v *vm) trace() {
	if v.traceWriter == nil {
		return
	}

	word := v.input[v.wordIndex]
	fmt.Fprintf(v.traceWriter, "trace: thread pc=%d %v on word '%s'\n",
		v.thread.pc, v.currentinstr(), word)
}

func (v *vm) traceBind() {
	if v.traceWriter == nil {
		return
	}

	word := v.input[v.wordIndex]
	fmt.Fprintf(v.traceWriter, "trace:     binding %s (%d items)\n",
		word, len(v.thread.items))
}

func (v *vm) addMatch(t *thread) {
	var m match
	for _, b := range t.items {
		var item interface{}
		switch b.instr.opcode {
		case opCmp:
			item = keywordValue{Name: b.instr.strs[0], Value: *b.val}
		case opSave:
			item = VarValue{Name: b.instr.strs[0],
				Type:  b.instr.strs[1],
				Value: *b.val,
			}
		default:
			panic("Unsupported opcode in thread bindings")
		}

		m.items = append(m.items, item)
	}
	m.meta = t.meta
	v.matches = append(v.matches, m)
}

func (v *vm) currentinstr() *instr {
	return &v.prog[v.thread.pc]
}

func (v *vm) currentWord() *string {
	return &v.input[v.wordIndex]
}

func (v *vm) swap(a, b *threadList) {
	*a, *b = *b, *a
}

func (v *vm) clear(l *threadList) {
	*l = (*l)[:0]
}

func (v *vm) addThread(l *threadList, t *thread) {
	if len(v.prog) == 0 {
		return
	}
	instr := &v.prog[t.pc]
	if instr.gen == v.gen {
		return // already on the list
	}
	*l = append(*l, t)
}

func (v *vm) longestMatches() []match {
	count := 0
	mlen := 0
	for _, m := range v.matches {
		if len(m.items) > mlen {
			count = 1
			mlen = len(m.items)
		} else if len(m.items) == mlen {
			count++
		}
	}

	matches := make([]match, count)
	i := 0
	for _, m := range v.matches {
		if len(m.items) == mlen {
			matches[i] = m
			i++
		}
	}

	return matches
}

func (v *vm) maximalMatches() []match {
	m := v.longestMatches()
	if len(m) > 0 && len(m[0].items) != len(v.input) {
		m = []match{}
	}
	return m
}
