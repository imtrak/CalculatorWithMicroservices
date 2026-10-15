// Package expression tokenizes, parses and evaluates arithmetic expressions.
//
// The grammar is an allowlist: finite decimal numbers, parentheses, spaces and
// tabs, the binary operators + - * /, and unary + and -. Input is never handed
// to a language runtime, shell or database.
//
//	expr    = term { ("+" | "-") term }
//	term    = unary { ("*" | "/") unary }
//	unary   = ("+" | "-") unary | primary
//	primary = number | "(" expr ")"
package expression

import (
	"fmt"
	"math"
	"strconv"
)

const (
	// MaxLength is the longest accepted expression, in bytes.
	MaxLength = 256
	// MaxDepth bounds parenthesis and unary-sign nesting.
	MaxDepth = 64
)

// Error codes reported to API clients.
const (
	CodeEmptyExpression      = "empty_expression"
	CodeExpressionTooLong    = "expression_too_long"
	CodeInvalidCharacters    = "invalid_characters"
	CodeInvalidNumber        = "invalid_number"
	CodeInvalidSyntax        = "invalid_syntax"
	CodeUnmatchedParenthesis = "unmatched_parenthesis"
	CodeNestingTooDeep       = "nesting_too_deep"
	CodeDivisionByZero       = "division_by_zero"
	CodeNonFiniteResult      = "non_finite_result"
)

// Error is a problem with the expression itself, as opposed to a server fault.
type Error struct {
	Code    string
	Message string
}

func (e *Error) Error() string { return e.Message }

func newError(code, format string, args ...any) *Error {
	return &Error{Code: code, Message: fmt.Sprintf(format, args...)}
}

// Evaluate computes the value of input. Every failure caused by the input is
// returned as *Error.
func Evaluate(input string) (float64, error) {
	if len(input) > MaxLength {
		return 0, newError(CodeExpressionTooLong, "Expression must be at most %d characters.", MaxLength)
	}

	tokens, err := tokenize(input)
	if err != nil {
		return 0, err
	}
	if len(tokens) == 0 {
		return 0, newError(CodeEmptyExpression, "Expression must not be empty.")
	}

	tree, err := parse(tokens)
	if err != nil {
		return 0, err
	}

	value, err := tree.eval()
	if err != nil {
		return 0, err
	}
	return tidy(value), nil
}

// tidy hides binary floating-point noise (0.1 + 0.2 is 0.30000000000000004) by
// rounding to 15 significant digits, which keeps integers up to 15 digits
// exact. It also turns negative zero into zero.
func tidy(v float64) float64 {
	if v == 0 {
		return 0
	}
	rounded, err := strconv.ParseFloat(strconv.FormatFloat(v, 'g', 15, 64), 64)
	if err != nil || math.IsInf(rounded, 0) {
		return v
	}
	return rounded
}
