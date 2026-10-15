package expression

import (
	"math"
	"strconv"
)

type tokenKind int

const (
	tokNumber tokenKind = iota
	tokPlus
	tokMinus
	tokStar
	tokSlash
	tokCaret
	tokPercent
	tokSqrt
	tokLParen
	tokRParen
)

type token struct {
	kind  tokenKind
	text  string
	value float64 // set for tokNumber
	pos   int     // 1-based position in the input, for error messages
}

func tokenize(input string) ([]token, error) {
	var tokens []token

	for i := 0; i < len(input); {
		c := input[i]
		switch {
		case c == ' ' || c == '\t':
			i++

		case isDigit(c) || c == '.':
			start := i
			dots := 0
			for i < len(input) && (isDigit(input[i]) || input[i] == '.') {
				if input[i] == '.' {
					dots++
				}
				i++
			}
			text := input[start:i]
			value, err := strconv.ParseFloat(text, 64)
			if dots > 1 || err != nil || math.IsInf(value, 0) {
				return nil, newError(CodeInvalidNumber, "Invalid number %q at position %d.", text, start+1)
			}
			tokens = append(tokens, token{kind: tokNumber, text: text, value: value, pos: start + 1})

		case isLetter(c):
			start := i
			for i < len(input) && isLetter(input[i]) {
				i++
			}
			word := input[start:i]
			if word != "sqrt" {
				return nil, newError(CodeInvalidCharacters, "Unknown word %q at position %d. The only supported function is sqrt.", word, start+1)
			}
			tokens = append(tokens, token{kind: tokSqrt, text: word, pos: start + 1})

		default:
			kind, ok := operatorKinds[c]
			if !ok {
				return nil, newError(CodeInvalidCharacters, "Unsupported character %q at position %d.", string(c), i+1)
			}
			tokens = append(tokens, token{kind: kind, text: string(c), pos: i + 1})
			i++
		}
	}
	return tokens, nil
}

var operatorKinds = map[byte]tokenKind{
	'+': tokPlus,
	'-': tokMinus,
	'*': tokStar,
	'/': tokSlash,
	'^': tokCaret,
	'%': tokPercent,
	'(': tokLParen,
	')': tokRParen,
}

func isDigit(c byte) bool  { return c >= '0' && c <= '9' }
func isLetter(c byte) bool { return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' }
