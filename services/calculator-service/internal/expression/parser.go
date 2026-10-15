package expression

import "math"

// node is an evaluable piece of the syntax tree.
type node interface {
	eval() (float64, error)
}

type (
	number float64

	unary struct {
		op      tokenKind
		operand node
	}

	binary struct {
		op          tokenKind
		left, right node
	}

	// percent is the postfix % operator: it divides its operand by 100.
	percent struct{ operand node }

	squareRoot struct{ operand node }

	// group marks a parenthesized percent so that "100 + (10%)" is a plain sum
	// rather than "10% of the left operand".
	group struct{ operand node }
)

func (g group) eval() (float64, error) { return g.operand.eval() }

func (n number) eval() (float64, error) { return float64(n), nil }

func (u unary) eval() (float64, error) {
	v, err := u.operand.eval()
	if err != nil {
		return 0, err
	}
	if u.op == tokMinus {
		return -v, nil
	}
	return v, nil
}

func (p percent) eval() (float64, error) {
	v, err := p.operand.eval()
	if err != nil {
		return 0, err
	}
	return v / 100, nil
}

func (s squareRoot) eval() (float64, error) {
	v, err := s.operand.eval()
	if err != nil {
		return 0, err
	}
	if v < 0 {
		return 0, newError(CodeNegativeSquareRoot, "Cannot take the square root of a negative number.")
	}
	return math.Sqrt(v), nil
}

func (b binary) eval() (float64, error) {
	l, err := b.left.eval()
	if err != nil {
		return 0, err
	}
	r, err := b.right.eval()
	if err != nil {
		return 0, err
	}

	var v float64
	switch b.op {
	case tokPlus:
		if _, ok := b.right.(percent); ok {
			r *= l // "a + b%" adds b percent of a
		}
		v = l + r
	case tokMinus:
		if _, ok := b.right.(percent); ok {
			r *= l // "a - b%" subtracts b percent of a
		}
		v = l - r
	case tokStar:
		v = l * r
	case tokSlash:
		if r == 0 {
			return 0, newError(CodeDivisionByZero, "Division by zero.")
		}
		v = l / r
	case tokCaret:
		if l == 0 && r < 0 {
			return 0, newError(CodeDivisionByZero, "Division by zero.")
		}
		v = math.Pow(l, r)
		if math.IsNaN(v) {
			return 0, newError(CodeInvalidPower, "Cannot raise a negative number to a fractional power.")
		}
	}

	if math.IsInf(v, 0) || math.IsNaN(v) {
		return 0, newError(CodeNonFiniteResult, "The result is too large to represent.")
	}
	return v, nil
}

type parser struct {
	tokens []token
	pos    int
	depth  int
}

func parse(tokens []token) (node, error) {
	p := &parser{tokens: tokens}
	tree, err := p.expr()
	if err != nil {
		return nil, err
	}
	if tok, ok := p.peek(); ok {
		// Only a ")" can be left over once a complete expression has been read
		// without it being an unexpected token, so name that case precisely.
		if tok.kind == tokRParen {
			return nil, newError(CodeUnmatchedParenthesis, "Unmatched closing parenthesis at position %d.", tok.pos)
		}
		return nil, unexpected(tok)
	}
	return tree, nil
}

func (p *parser) peek() (token, bool) {
	if p.pos >= len(p.tokens) {
		return token{}, false
	}
	return p.tokens[p.pos], true
}

func (p *parser) enter() error {
	p.depth++
	if p.depth > MaxDepth {
		return newError(CodeNestingTooDeep, "Expression is nested more than %d levels deep.", MaxDepth)
	}
	return nil
}

func (p *parser) leave() { p.depth-- }

func (p *parser) expr() (node, error) {
	return p.binaryLevel(p.term, tokPlus, tokMinus)
}

func (p *parser) term() (node, error) {
	return p.binaryLevel(p.unary, tokStar, tokSlash)
}

// binaryLevel parses operand { op operand } for the given operators, building a
// left-associative tree.
func (p *parser) binaryLevel(operand func() (node, error), ops ...tokenKind) (node, error) {
	left, err := operand()
	if err != nil {
		return nil, err
	}
	for {
		tok, ok := p.peek()
		if !ok || !contains(ops, tok.kind) {
			return left, nil
		}
		p.pos++
		right, err := operand()
		if err != nil {
			return nil, err
		}
		left = binary{op: tok.kind, left: left, right: right}
	}
}

// unary binds more weakly than ^, so -2^2 is -(2^2).
func (p *parser) unary() (node, error) {
	tok, ok := p.peek()
	if ok && (tok.kind == tokPlus || tok.kind == tokMinus) {
		if err := p.enter(); err != nil {
			return nil, err
		}
		defer p.leave()
		p.pos++
		operand, err := p.unary()
		if err != nil {
			return nil, err
		}
		return unary{op: tok.kind, operand: operand}, nil
	}
	return p.power()
}

// power is right associative, and its exponent may carry a sign: 2^3^2 is
// 2^(3^2) and 2^-1 is 0.5.
func (p *parser) power() (node, error) {
	base, err := p.postfix()
	if err != nil {
		return nil, err
	}
	tok, ok := p.peek()
	if !ok || tok.kind != tokCaret {
		return base, nil
	}
	p.pos++
	exponent, err := p.unary()
	if err != nil {
		return nil, err
	}
	return binary{op: tokCaret, left: base, right: exponent}, nil
}

// postfix applies any number of % operators to a primary.
func (p *parser) postfix() (node, error) {
	operand, err := p.primary()
	if err != nil {
		return nil, err
	}
	for {
		tok, ok := p.peek()
		if !ok || tok.kind != tokPercent {
			return operand, nil
		}
		p.pos++
		operand = percent{operand}
	}
}

func (p *parser) primary() (node, error) {
	tok, ok := p.peek()
	if !ok {
		return nil, newError(CodeInvalidSyntax, "Unexpected end of expression.")
	}

	switch tok.kind {
	case tokNumber:
		p.pos++
		return number(tok.value), nil

	case tokLParen:
		return p.parenthesized()

	case tokSqrt:
		p.pos++
		if next, ok := p.peek(); !ok || next.kind != tokLParen {
			return nil, newError(CodeInvalidSyntax, "Expected \"(\" after sqrt at position %d.", tok.pos)
		}
		inner, err := p.parenthesized()
		if err != nil {
			return nil, err
		}
		return squareRoot{inner}, nil

	default:
		return nil, unexpected(tok)
	}
}

// parenthesized parses "(" expr ")". The caller guarantees the next token is
// the opening parenthesis.
func (p *parser) parenthesized() (node, error) {
	open := p.tokens[p.pos]
	if err := p.enter(); err != nil {
		return nil, err
	}
	defer p.leave()
	p.pos++

	inner, err := p.expr()
	if err != nil {
		return nil, err
	}
	closing, ok := p.peek()
	if !ok {
		return nil, newError(CodeUnmatchedParenthesis, "Missing closing parenthesis for the one at position %d.", open.pos)
	}
	if closing.kind != tokRParen {
		return nil, unexpected(closing)
	}
	p.pos++
	if _, ok := inner.(percent); ok {
		return group{inner}, nil
	}
	return inner, nil
}

func unexpected(tok token) *Error {
	return newError(CodeInvalidSyntax, "Unexpected %q at position %d.", tok.text, tok.pos)
}

func contains(kinds []tokenKind, k tokenKind) bool {
	for _, c := range kinds {
		if c == k {
			return true
		}
	}
	return false
}
