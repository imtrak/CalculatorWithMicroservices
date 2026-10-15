# ADR-0006: Add power, square root and percentage to the grammar

- Status: Accepted
- Date: 2026-10-15
- Amends: [ADR-0001](0001-free-form-arithmetic-expressions.md)

## Context

[ADR-0001](0001-free-form-arithmetic-expressions.md) limited the grammar to
numbers, parentheses and `+ - * /`, and said that adding functions or
exponentiation requires deliberate parser, validation and test updates. The
assignment lists exponentiation, square root and percentage as optional
operations.

Each of the three has conventions that differ between calculators, so they have
to be chosen explicitly rather than left implicit in the parser.

## Decision

The grammar gains a power operator, a function and a postfix operator:

```text
expr    = term { ("+" | "-") term }
term    = unary { ("*" | "/") unary }
unary   = ("+" | "-") unary | power
power   = postfix [ "^" unary ]
postfix = primary { "%" }
primary = number | "sqrt" "(" expr ")" | "(" expr ")"
```

Precedence, from strongest to weakest: `%`, `^`, unary `+`/`-`, `*` and `/`, then
`+` and `-`.

| Operation | Convention | Examples |
|---|---|---|
| Power `^` | Right associative. Stronger than unary minus, as in mathematical notation. The exponent may carry a sign. | `2^3^2` is 512, `-2^2` is -4, `(-2)^2` is 4, `2^-1` is 0.5 |
| Square root `sqrt(x)` | A function that always needs parentheses. Lowercase only. | `sqrt(16)` is 4, `sqrt 16` is a syntax error, `SQRT(16)` is not accepted |
| Percentage `x%` | A postfix operator meaning "divide by 100". When it is the right operand of `+` or `-` it means "percent of the left operand" (the running total). | `50%` is 0.5, `200*50%` is 100, `100 + 10%` is 110, `100 - 10%` is 90, `100 / 10%` is 1000 |

New error codes, all `400 Bad Request` with the usual body:

| Code | When |
|---|---|
| `negative_square_root` | `sqrt` of a negative number, e.g. `sqrt(-1)` |
| `invalid_power` | A negative base with a fractional exponent, e.g. `(-8)^0.5` |

Related cases reuse existing codes: `0^-1` is `division_by_zero`, and a power that
overflows, such as `10^400`, is `non_finite_result`. An unknown word such as
`cos(4)` is `invalid_characters`.

The frontend adds keys for `√`, `^` and `%`, accepts the `√` sign when typed, and
its character check is extended to the same alphabet. That check must always
match the backend's.

## Consequences

- Expressions such as `sqrt(2)^2`, `2^10` and `15%` work, and `^` makes it possible
  to produce results beyond the representable range, which is reported as
  `non_finite_result`. Before this change that error was unreachable.
- `%` follows the pocket-calculator convention for `+` and `-` (`100 + 10%` is
  110) and is plain "divide by 100" everywhere else. Only a `%` that directly
  follows `+` or `-` is special: `100 + 5*10%` is 100.5 and `100 + (10%)` is 100.1.
- There is still no modulo operator, so `5%2` is a syntax error rather than a
  remainder, and there is no implicit multiplication.
- Letters now appear in valid input, so the allowed alphabet is no longer just
  digits and symbols. The word `sqrt` is the only one accepted.
- Results are still rounded to 15 significant digits, so large powers lose
  precision beyond that.
- The grammar lives in three places that must change together: the backend
  tokenizer and parser, the frontend character check, and the documentation.

## Considered alternatives

### Percentage as a share of the left operand

Applying "percent of the left operand" to every operator would make `100 * 10%`
equal 10 and `100 / 10%` equal 1000 only by accident of the formula. It was
limited to `+` and `-`, where the convention is unambiguous, and the other
operators keep the plain "divide by 100" meaning. A pure postfix "divide by 100"
for all operators was the first choice, but `100 + 10%` giving 100.1 surprised
users.

### `**` or `pow(x, y)` for powers

`**` is a second spelling for the same operation, and `pow(x, y)` would require
argument lists and commas in the grammar for a single function. `^` is the
notation most users already expect on a calculator.

### Left-associative power

Would make `2^3^2` equal 64. Mathematical convention, and most calculators and
languages that have the operator, treat it as right associative, giving 512.

### `√` as the accepted function syntax

A single symbol that is hard to type on many keyboards. The frontend accepts it
and converts it to `sqrt`, while the API itself keeps the ASCII spelling.
