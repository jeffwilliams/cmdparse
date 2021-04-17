package cmdparse

import "testing"

func TestCmdScanner(t *testing.T) {

	ensureTokListsEqual := func(exp, act []string) {
		if len(exp) != len(act) {
			t.Fatalf("expected %d tokens but got %d. Actual tokens: %v", len(exp), len(act), act)
		}

		for i := 0; i < len(exp); i++ {
			if exp[i] != act[i] {
				t.Fatalf("at index %d in tokens: expected token %v but got %v", i, exp, act)
			}
		}
	}

	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "empty",
			input:    "",
			expected: []string{},
		},
		{
			name:     "word",
			input:    "word",
			expected: []string{"word"},
		},
		{
			name:     "spaced word",
			input:    "   word \t",
			expected: []string{"word"},
		},
		{
			name:     "this is a test",
			input:    "this is a test",
			expected: []string{"this", "is", "a", "test"},
		},
		{
			name:     " this    is \t\n a   test  ",
			input:    " this    is \t\n a   test  ",
			expected: []string{"this", "is", "a", "test"},
		},
		{
			name:     `what "is this thing"`,
			input:    `what "is this thing"`,
			expected: []string{"what", "is this thing"},
		},
		{
			name:     `"is this thing" this "thing"`,
			input:    `"is this thing" this "thing"`,
			expected: []string{"is this thing", "this", "thing"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var s cmdScanner
			toks := s.Scan(tc.input)
			ensureTokListsEqual(tc.expected, toks)
		})
	}

}

func TestCmdParse(t *testing.T) {
	type tcmd struct {
		syntax string
		cback  Callback
	}

	var cbackCalled bool
	var cbackId string

	tests := []struct {
		name            string
		cmds            []tcmd
		input           string
		ok              bool
		expectedCbackId string
	}{
		{
			name:  "empty",
			cmds:  []tcmd{},
			input: "blah",
			ok:    false,
		},
		{
			name: "one",
			cmds: []tcmd{
				{"get <what>",
					func(match Match, ctx interface{}) {
						cbackCalled = true
						vals := match.Var("what")
						if len(vals) == 0 || len(vals) > 1 {
							t.Fatalf("Command match was %d when it shouldn't be", len(vals))
						}
						if vals[0].Value != "leaf" {
							t.Fatalf("Command match was %s when it should be %s",
								vals[0].Value, "leaf")
						}
					}},
			},
			input: "get leaf",
			ok:    true,
		},
		{
			name: "two.1",
			cmds: []tcmd{
				{"info things?",
					func(match Match, ctx interface{}) {
						cbackCalled = true
						if !match.KeywordPresent("things") {
							t.Fatalf("The ‘things’ keyword was not present when it should be")
						}
						cbackId = "info"
					},
				},
				{"drop",
					func(match Match, ctx interface{}) {
						cbackCalled = true
						cbackId = "drop"
					},
				},
			},
			input:           "in t",
			ok:              true,
			expectedCbackId: "info",
		},
		{
			name: "two.2",
			cmds: []tcmd{
				{"info things?",
					func(match Match, ctx interface{}) {
						cbackCalled = true
						if !match.KeywordPresent("things") {
							t.Fatalf("The ‘things’ keyword was not present when it should be")
						}
						cbackId = "info"
					},
				},
				{"drop",
					func(match Match, ctx interface{}) {
						cbackCalled = true
						cbackId = "drop"
					},
				},
			},
			input:           "dr",
			ok:              true,
			expectedCbackId: "drop",
		},
		{
			name: "two.3",
			cmds: []tcmd{
				{"info things?",
					func(match Match, ctx interface{}) {
						cbackCalled = true
						if !match.KeywordPresent("things") {
							t.Fatalf("The ‘things’ keyword was not present when it should be")
						}
						cbackId = "info"
					},
				},
				{"drop",
					func(match Match, ctx interface{}) {
						cbackCalled = true
						cbackId = "drop"
					},
				},
			},
			input: "bloop",
			ok:    false,
		},
		{
			name: "checkCtx",
			cmds: []tcmd{
				{"doit",
					func(match Match, ctx interface{}) {
						if ctx.(int) != 5 {
							t.Fatalf("Context is not passed properly to callback")
						}
						cbackCalled = true
						cbackId = "doit"
					},
				},
			},
			input:           "doit",
			ok:              true,
			expectedCbackId: "doit",
		},
		{
			name: "show_results",
			cmds: []tcmd{
				{"show results (source (scheduled | unscheduled | all))? detail?",
					func(match Match, ctx interface{}) {
						cbackCalled = true
						cbackId = "1"
					},
				},
			},
			input:           "sh res so sch",
			ok:              true,
			expectedCbackId: "1",
		},
	}

	for _, tc := range tests {

		t.Run(tc.name, func(t *testing.T) {

			var cmds Cmds

			for _, c := range tc.cmds {
				cmds.Add(c.syntax, c.cback)
			}

			cmds.Compile()

			cbackCalled = false
			ok := cmds.Parse(tc.input, 5)
			if ok != tc.ok {
				t.Fatalf("Parse returned %v when it should have returned %v", ok, tc.ok)
			}

			if tc.ok {
				if !cbackCalled {
					t.Fatalf("Parse returned with success, but the callback was not called")
				}
				if cbackId != tc.expectedCbackId {
					t.Fatalf("Expected callback ‘%s’ to be called but instead ‘%s’ was", tc.expectedCbackId, cbackId)
				}
			}
		})
	}

}
