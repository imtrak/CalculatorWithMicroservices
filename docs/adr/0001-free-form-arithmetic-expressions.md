# ADR-0001: Free-form arithmetic expressions

- Status: Accepted, amended by [ADR-0006](0006-power-square-root-and-percentage.md) (adds power, square root and percentage)
- Date: 2026-10-15

## Context

The calculator should feel natural to regular users. Requiring them to select an
operation and fill separate operand fields makes common calculations such as
`(12 + 3) * 4` unnecessarily cumbersome and prevents them from expressing a
calculation as they normally would.

The backend remains responsible for evaluating calculations so that validation,
arithmetic rules, and errors are consistent for every client.

## Decision

The backend will expose one calculation endpoint that accepts a free-form
arithmetic expression:

```http
POST /api/v1/calculations
Content-Type: application/json

{
  "expression": "(12 + 3) * 4"
}
```

The successful response will include both the original expression and its
result:

```json
{
  "expression": "(12 + 3) * 4",
  "result": 60
}
```

The service will tokenize and parse expressions with an allowlisted grammar; it
must never pass user input to a language runtime, `eval`, shell, or database.
Initially, the grammar supports finite decimal numbers, parentheses, whitespace,
and the `+`, `-`, `*`, and `/` operators. Standard precedence applies:
parentheses first, then multiplication and division, then addition and
subtraction. Unary positive and negative signs are supported where valid.

Malformed expressions, unsupported characters, non-finite values, unmatched
parentheses, and division by zero return `400 Bad Request` with a readable error
code and message. The frontend may provide immediate input feedback, but the
backend is authoritative for validation and evaluation.

## Consequences

- Users can enter calculations in a familiar form, including multi-step
  expressions and parentheses.
- The frontend has one simple input model and one API contract.
- The backend needs a tokenizer and parser, which introduces more implementation
  and test surface than dispatching individual binary operations.
- The supported grammar must be documented and deliberately expanded; adding
  functions such as `sqrt` or exponentiation requires parser, validation, and
  test updates.
- Expression length and parser nesting limits should be enforced to keep
  resource usage predictable.

## Considered alternatives

### One operation and two operands per request

For example, `{ "operation": "add", "leftOperand": 12, "rightOperand": 3 }`.
This is straightforward to implement, but it makes multi-step calculations
awkward and is less aligned with how users expect a calculator to work.

### One endpoint per operation

For example, `POST /add` and `POST /divide`. This similarly forces the client
to orchestrate multi-step calculations and duplicates HTTP contracts and
validation as operations grow.

### Evaluate the expression directly in the frontend or runtime

This would be unsafe for untrusted input and would duplicate or bypass the
backend's business rules. A constrained parser keeps evaluation deterministic
and safe.
