package cmdparse

import (
	"testing"
)

func TestScanner(t *testing.T) {
	ensureToksEqual := func(exp, act token, index int) {
		if exp.tokenType() != act.tokenType() {
			t.Fatalf("at index %d in tokens: expected token %v but got %v", index, exp, act)
		}
		if exp.value != act.value {
			t.Fatalf("at index %d in tokens: expected token %v but got %v", index, exp, act)
		}
	}

	ensureTokListsEqual := func(exp, act []token) {
		if len(exp) != len(act) {
			t.Fatalf("expected %d tokens but got %d. Actual tokens: %v", len(exp), len(act), act)
		}

		for i := 0; i < len(exp); i++ {
			ensureToksEqual(exp[i], act[i], i)
		}
	}

	ensureErrListEqual := func(errs []error, errMsgs []string) {
		if len(errs) != len(errMsgs) {
			t.Fatalf("expected %d errors but got %d. Actual errors: %v", len(errMsgs), len(errs), errs)
		}

		for i := 0; i < len(errs); i++ {
			if errs[i].Error() != errMsgs[i] {
				t.Fatalf("expected error %s but got %s. Actual errors: %v", errMsgs[i], errs[i].Error(), errs)
			}
		}
	}

	tests := []struct {
		name     string
		input    string
		expected []token
		ok       bool
		errors   []string
	}{
		{
			name:     "empty",
			input:    "",
			expected: []token{},
			ok:       true,
			errors:   []string{},
		},
		{
			name:     "word",
			input:    "word",
			expected: []token{{typ: wordTok, value: "word"}},
			ok:       true,
			errors:   []string{},
		},
		{
			name:     "spaced word",
			input:    "   word \t",
			expected: []token{{typ: wordTok, value: "word"}},
			ok:       true,
			errors:   []string{},
		},
		{
			name:     "thing?",
			input:    "thing?",
			expected: []token{{typ: wordTok, value: "thing"}, {typ: questionTok}},
			ok:       true,
			errors:   []string{},
		},
		{
			name:     "<:?",
			input:    "<:?",
			expected: []token{{typ: lessThanTok}, {typ: colonTok}, {typ: questionTok}},
			ok:       true,
			errors:   []string{},
		},
		{
			name:     "  <  :    \t?",
			input:    "  <  :    \t?",
			expected: []token{{typ: lessThanTok}, {typ: colonTok}, {typ: questionTok}},
			ok:       true,
			errors:   []string{},
		},
		{
			name:     "word:word2",
			input:    "word:word2",
			expected: []token{{typ: wordTok, value: "word"}, {typ: colonTok}, {typ: wordTok, value: "word2"}},
			ok:       true,
			errors:   []string{},
		},
		{
			name:     "alts with quotes",
			input:    "set \"<a>\"",
			expected: nil,
			ok:       false,
			errors:   []string{"Invalid character '\"' encountered", "Invalid character '\"' encountered"},
		},
	}

	for _, tc := range tests {

		t.Run(tc.name, func(t *testing.T) {

			var s scanner
			toks, ok := s.Scan(tc.input)
			if ok != tc.ok {
				t.Fatalf("Scan returned ok=%v but expected %v", ok, tc.ok)
			}
			if ok {
				ensureTokListsEqual(tc.expected, toks)
			}
			ensureErrListEqual(s.errs, tc.errors)
		})
	}

}
