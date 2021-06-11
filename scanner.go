package cmdparse

import (
	"bytes"
	"fmt"
	"io"
	"unicode"
)

type scanner struct {
	pos    int
	input  []rune
	tokens []token
	errs   []error
}

type token struct {
	typ   tokenType
	value string
	// pos is the index of the rune in the input
	// where the token started
	pos int
}

func (t token) tokenType() tokenType {
	return t.typ
}

func (t token) len() int {
	if t.typ == wordTok {
		return len(t.value)
	} else {
		return 1
	}
}

func (t token) String() string {
	if t.value != "" {
		return fmt.Sprintf("(%s)", t.typ)
	} else {
		return fmt.Sprintf("(%s, %s)", t.typ, t.value)
	}
}

var nilToken = token{}

func (s *scanner) Scan(cmd string) (tokens []token, ok bool) {
	s.input = []rune(cmd)
	// TODO: to generate less garbage, re-use the existing arrays.
	s.tokens = make([]token, 0, 10)
	s.errs = make([]error, 0, 10)

	for {
		t, err := s.next()
		if err != nil {
			if err == io.EOF {
				break
			}

			s.errs = append(s.errs, err)
		}
		s.addToken(t)
	}
	return s.tokens, len(s.errs) == 0
}

func (s *scanner) next() (tok token, err error) {
	var r rune
	for {
		if s.atEnd() {
			return nilToken, io.EOF
		}

		r = s.input[s.pos]
		if !unicode.IsSpace(r) {
			break
		}
		s.pos++
	}

	tok.pos = s.pos
	switch r {
	case '<':
		s.pos++
		tok.typ = lessThanTok
	case '>':
		s.pos++
		tok.typ = greaterThanTok
	case '|':
		s.pos++
		tok.typ = pipeTok
	case '*':
		s.pos++
		tok.typ = starTok
	case '+':
		s.pos++
		tok.typ = plusTok
	case '?':
		s.pos++
		tok.typ = questionTok
	case '(':
		s.pos++
		tok.typ = leftParenTok
	case ')':
		s.pos++
		tok.typ = rightParenTok
	case ':':
		s.pos++
		tok.typ = colonTok
	default:
		p := s.pos
		tok, err = s.word()
		if err != nil {
			return
		}
		tok.pos = p
	}

	return tok, nil
}

func (s *scanner) atEnd() bool {
	return s.pos >= len(s.input)
}

func (s *scanner) word() (token, error) {
	var buf bytes.Buffer
	r := s.input[s.pos]

	if !s.isValidWordRune(r) {
		s.pos++ // Consume this bad character
		return nilToken, fmt.Errorf("Invalid character '%c' encountered", r)
	}

	for s.isValidWordRune(r) {
		buf.WriteRune(r)
		s.pos++

		if s.atEnd() {
			return token{typ: wordTok, value: buf.String()}, nil
		}
		r = s.input[s.pos]
	}

	return token{typ: wordTok, value: buf.String()}, nil
}

func (s *scanner) isValidWordRune(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '-'
}

func (s *scanner) addToken(t token) {
	s.tokens = append(s.tokens, t)
}

func (s *scanner) addError(e error) {
	s.errs = append(s.errs, e)
}

type tokenType int

const (
	nilTok tokenType = iota
	lessThanTok
	greaterThanTok
	pipeTok
	starTok
	plusTok
	questionTok
	leftParenTok
	rightParenTok
	colonTok

	wordTok
)

func (t tokenType) String() string {
	switch t {
	case nilTok:
		return "nilTok"
	case lessThanTok:
		return "lessThanTok"
	case greaterThanTok:
		return "greaterThanTok"
	case pipeTok:
		return "pipeTok"
	case starTok:
		return "starTok"
	case plusTok:
		return "plusTok"
	case questionTok:
		return "questionTok"
	case leftParenTok:
		return "leftParenTok"
	case rightParenTok:
		return "rightParenTok"
	case colonTok:
		return "colonTok"
	case wordTok:
		return "wordTok"
	}
	return "<unknown token>"
}
