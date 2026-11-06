package focusquery

import (
	"database/sql"
	"strings"
	"testing"
)

func TestParseAndCompile(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		wantClauseSub string
		wantArgCount  int
		wantTerms     []string
		wantErr       bool
	}{
		{
			name:          "single term",
			input:         "semiconductor",
			wantClauseSub: "items.title LIKE :ff_q_0",
			wantArgCount:  1,
			wantTerms:     []string{"semiconductor"},
			wantErr:       false,
		},
		{
			name:          "implicit AND with space",
			input:         "TSMC foundry",
			wantClauseSub: "AND",
			wantArgCount:  2,
			wantTerms:     []string{"TSMC", "foundry"},
			wantErr:       false,
		},
		{
			name:          "explicit OR with comma",
			input:         "Nvidia, AMD, Intel",
			wantClauseSub: "OR",
			wantArgCount:  3,
			wantTerms:     []string{"Nvidia", "AMD", "Intel"},
			wantErr:       false,
		},
		{
			name:          "explicit OR with keyword",
			input:         "Nvidia OR AMD",
			wantClauseSub: "OR",
			wantArgCount:  2,
			wantTerms:     []string{"Nvidia", "AMD"},
			wantErr:       false,
		},
		{
			name:          "negation with NOT",
			input:         "chips NOT gaming",
			wantClauseSub: "NOT",
			wantArgCount:  2,
			wantTerms:     []string{"chips"}, // "gaming" is negated, so excluded from positive terms
			wantErr:       false,
		},
		{
			name:          "negation with minus prefix",
			input:         "chips -gaming",
			wantClauseSub: "NOT",
			wantArgCount:  2,
			wantTerms:     []string{"chips"},
			wantErr:       false,
		},
		{
			name:          "parentheses grouping",
			input:         "(TSMC OR Intel) packaging",
			wantClauseSub: "OR",
			wantArgCount:  3,
			wantTerms:     []string{"TSMC", "Intel", "packaging"},
			wantErr:       false,
		},
		{
			name:          "exact phrase in quotes",
			input:         `"interest rate cut" PBOC`,
			wantClauseSub: "AND",
			wantArgCount:  2,
			wantTerms:     []string{"interest rate cut", "PBOC"},
			wantErr:       false,
		},
		{
			name:          "field scoping to title",
			input:         `title:"rate hike" PBOC`,
			wantClauseSub: "items.title LIKE :ff_q_0 OR items.translated_title LIKE :ff_q_0",
			wantArgCount:  2,
			wantTerms:     []string{"rate hike", "PBOC"},
			wantErr:       false,
		},
		{
			name:          "CJK / Multilingual terms",
			input:         `(日銀 OR 中国人民银行) AND 利上げ`,
			wantClauseSub: "AND",
			wantArgCount:  3,
			wantTerms:     []string{"日銀", "中国人民银行", "利上げ"},
			wantErr:       false,
		},
		{
			name:    "unmatched opening parenthesis",
			input:   "(semiconductor OR chip",
			wantErr: true,
		},
		{
			name:    "empty query",
			input:   "   ",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ast, err := Parse(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Parse() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr || ast == nil {
				return
			}

			terms := ExtractPositiveTerms(ast)
			if len(terms) != len(tt.wantTerms) {
				t.Errorf("ExtractPositiveTerms() got %v, want %v", terms, tt.wantTerms)
			} else {
				for i, want := range tt.wantTerms {
					if terms[i] != want {
						t.Errorf("ExtractPositiveTerms()[%d] = %q, want %q", i, terms[i], want)
					}
				}
			}

			clause, args, err := Compile(ast)
			if err != nil {
				t.Fatalf("Compile() error = %v", err)
			}

			if !strings.Contains(clause, tt.wantClauseSub) {
				t.Errorf("Compile() clause %q does not contain expected substring %q", clause, tt.wantClauseSub)
			}

			if len(args) != tt.wantArgCount {
				t.Errorf("Compile() args count = %d, want %d", len(args), tt.wantArgCount)
			}

			for _, arg := range args {
				named, ok := arg.(sql.NamedArg)
				if !ok {
					t.Errorf("arg is not sql.NamedArg: %v", arg)
				}
				if named.Name == "" || named.Value == "" {
					t.Errorf("named arg missing name or value: %v", named)
				}
			}
		})
	}
}
