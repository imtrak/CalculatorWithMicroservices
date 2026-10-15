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

		// Power: right associative, stronger than * and /, weaker than %.
		{"2^3", 8},
		{"2^3^2", 512},
		{"(2^3)^2", 64},
		{"2^-1", 0.5},
		{"2*3^2", 18},
		{"2^2*3", 12},
		{"10/2^2", 2.5},
		{"-2^2", -4}, // unary minus is weaker than ^
		{"(-2)^2", 4},
		{"(-2)^3", -8},
		{"2^0", 1},
		{"0^0", 1},
		{"0^2", 0},
		{"2^0.5", 1.4142135623731},

		// Square root.
		{"sqrt(16)", 4},
		{"sqrt(0)", 0},
		{"sqrt(2+2)", 2},
		{"sqrt(9) + sqrt(16)", 7},
		{"sqrt(16)*2", 8},
		{"sqrt(sqrt(16))", 2},
		{"2*sqrt(9)^2", 18},
		{"sqrt(2)", 1.4142135623731},
		{"-sqrt(4)", -2},
		{" sqrt( 25 ) ", 5},

		// Percentage: postfix, divides by 100.
		{"50%", 0.5},
		{"200*50%", 100},
		{"100 + 10%", 110}, // after + or -, % is a percent of the left operand
		{"100 - 10%", 90},
		{"100 * 10%", 10},
		{"100 / 10%", 1000},
		{"200 + 50% + 10%", 330}, // left operand is the running total
		{"100 + (10%)", 100.1},   // parentheses make it a plain number
		{"100 + 5*10%", 100.5},   // only a directly following % counts
		{"100 + 10%%", 100.1},
		{"10%%", 0.001},
		{"(25+25)%", 0.5},
		{"-50%", -0.5},
		{"4^50%", 2}, // % binds tighter than ^
		{"sqrt(1600%)", 4},
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
		{"uppercase function", "SQRT(4)", CodeInvalidCharacters},
		{"unknown function", "cos(4)", CodeInvalidCharacters},
		{"function glued to letters", "sqrtx(4)", CodeInvalidCharacters},
		{"exponent notation", "1e5", CodeInvalidCharacters},
		{"sqrt without parentheses", "sqrt 4", CodeInvalidSyntax},
		{"sqrt alone", "sqrt", CodeInvalidSyntax},
		{"sqrt empty", "sqrt()", CodeInvalidSyntax},
		{"sqrt unclosed", "sqrt(4", CodeUnmatchedParenthesis},
		{"implicit multiplication with sqrt", "2sqrt(4)", CodeInvalidSyntax},
		{"missing exponent", "2^", CodeInvalidSyntax},
		{"missing base", "^2", CodeInvalidSyntax},
		{"lone percent", "%", CodeInvalidSyntax},
		{"no modulo operator", "5%2", CodeInvalidSyntax},
		{"negative square root", "sqrt(-1)", CodeNegativeSquareRoot},
		{"computed negative square root", "sqrt(1-4)", CodeNegativeSquareRoot},
		{"fractional power of a negative", "(-8)^0.5", CodeInvalidPower},
		{"cube root of a negative", "(-8)^(1/3)", CodeInvalidPower},
		{"zero to a negative power", "0^-1", CodeDivisionByZero},
		{"power overflow", "10^400", CodeNonFiniteResult},
		{"tower overflow", "99^99^99", CodeNonFiniteResult},
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
