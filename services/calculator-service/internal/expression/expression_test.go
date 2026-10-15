package expression

import (
	"errors"
	"math"
	"strings"
	"testing"
)

func TestEvaluateValid(t *testing.T) {
	tests := []struct {
		input string
		want  float64
	}{
		{"1+2", 3},
		{"(12 + 3) * 4", 60},
		{"2 + 3 * 4", 14},
		{"(2 + 3) * 4", 20},
		{"10 - 4 - 3", 3},   // left associative
		{"100 / 10 / 5", 2}, // left associative
		{"10 / 4", 2.5},
		{"10+50+60/80*40", 90}, // / and * share precedence, left to right
		{"60/(80*40)", 0.01875},
		{"10+50+60/(80*40)", 60.01875},
		{"-3 + 5", 2},
		{"+3", 3},
		{"--3", 3},
		{"2 * -3", -6},
		{"2 - -3", 5},
		{"-(1 + 2)", -3},
		{"0.1 + 0.2", 0.3},
		{".5 + .5", 1},
		{"5.", 5},
		{"1 - 1", 0},
		{"((((1))))", 1},
		{"\t7 *\t6 ", 42},
		{"123456789012345 + 0", 123456789012345},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := Evaluate(tt.input)
			if err != nil {
				t.Fatalf("Evaluate(%q) error: %v", tt.input, err)
			}
			if got != tt.want {
				t.Errorf("Evaluate(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestEvaluateNegativeZeroIsZero(t *testing.T) {
	got, err := Evaluate("-0")
	if err != nil {
		t.Fatal(err)
	}
	if got != 0 || math.Signbit(got) {
		t.Errorf("Evaluate(\"-0\") = %v (signbit %v), want +0", got, math.Signbit(got))
	}
}

func TestEvaluateErrors(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantCode string
	}{
		{"empty", "", CodeEmptyExpression},
		{"blank", "  \t ", CodeEmptyExpression},
		{"letters", "2 + abc", CodeInvalidCharacters},
		{"semicolon", "1;2", CodeInvalidCharacters},
		{"newline", "1+\n1", CodeInvalidCharacters},
		{"non-ascii", "1 × 2", CodeInvalidCharacters},
		{"trailing operator", "1 +", CodeInvalidSyntax},
		{"leading operator", "* 2", CodeInvalidSyntax},
		{"two numbers", "1 2", CodeInvalidSyntax},
		{"implicit multiplication", "2(3)", CodeInvalidSyntax},
		{"empty parentheses", "()", CodeInvalidSyntax},
		{"double operator", "1 + * 2", CodeInvalidSyntax},
		{"missing closing", "(1 + 2", CodeUnmatchedParenthesis},
		{"extra closing", "1 + 2)", CodeUnmatchedParenthesis},
		{"two dots", "1.2.3", CodeInvalidNumber},
		{"double dot", "1..2", CodeInvalidNumber},
		{"lone dot", ".", CodeInvalidNumber},
		{"division by zero", "1/0", CodeDivisionByZero},
		{"computed division by zero", "1/(2-2)", CodeDivisionByZero},
		{"syntax error wins over division by zero", "1/0 +", CodeInvalidSyntax},
		{"too long", strings.Repeat("1", MaxLength+1), CodeExpressionTooLong},
		{"too deeply nested parentheses", strings.Repeat("(", MaxDepth+1) + "1" + strings.Repeat(")", MaxDepth+1), CodeNestingTooDeep},
		{"too many unary signs", strings.Repeat("-", MaxDepth+1) + "1", CodeNestingTooDeep},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Evaluate(tt.input)

			var exprErr *Error
			if !errors.As(err, &exprErr) {
				t.Fatalf("Evaluate error = %v, want *Error", err)
			}
			if exprErr.Code != tt.wantCode {
				t.Errorf("code = %q (%s), want %q", exprErr.Code, exprErr.Message, tt.wantCode)
			}
			if exprErr.Message == "" {
				t.Error("error message is empty")
			}
		})
	}
}

func TestEvaluateAcceptsMaximumNesting(t *testing.T) {
	input := strings.Repeat("(", MaxDepth) + "1" + strings.Repeat(")", MaxDepth)
	if got, err := Evaluate(input); err != nil || got != 1 {
		t.Errorf("Evaluate(max nesting) = %v, %v; want 1, nil", got, err)
	}
}

// Overflow cannot be reached through MaxLength-limited input, so check the
// guard on the syntax tree directly.
func TestBinaryEvalRejectsNonFiniteResults(t *testing.T) {
	tree := binary{op: tokStar, left: number(1e200), right: number(1e200)}

	_, err := tree.eval()

	var exprErr *Error
	if !errors.As(err, &exprErr) || exprErr.Code != CodeNonFiniteResult {
		t.Errorf("eval error = %v, want code %q", err, CodeNonFiniteResult)
	}
}
