// Client-side input rules. They give immediate feedback only; the backend is
// authoritative and re-validates every expression.

export const MAX_EXPRESSION_LENGTH = 256;

// Must match the backend's alphabet exactly (calculator-service tokenizer):
// only a space or a tab counts as whitespace, not \s, and the only word is the
// lowercase function name below.
const ALLOWED_CHARACTERS = /^[0-9+\-*/^%(). \t]*$/;
const FUNCTION_NAME = 'sqrt';

export type InputIssue = 'invalid_characters' | 'too_long';

export const INPUT_ISSUE_MESSAGES: Record<InputIssue, string> = {
  invalid_characters:
    'Only digits, spaces, decimal points, parentheses, sqrt and the operators + − × ÷ ^ % are allowed.',
  too_long: `Expression must be at most ${MAX_EXPRESSION_LENGTH} characters.`,
};

/** Maps the typographic symbols users may paste or type to the ASCII ones. */
export function normalizeOperators(input: string): string {
  return input
    .replaceAll('×', '*')
    .replaceAll('÷', '/')
    .replaceAll('−', '-')
    .replaceAll('√', FUNCTION_NAME);
}

export function checkInput(expression: string): InputIssue | null {
  if (!ALLOWED_CHARACTERS.test(expression.replaceAll(FUNCTION_NAME, ''))) {
    return 'invalid_characters';
  }
  if (expression.length > MAX_EXPRESSION_LENGTH) return 'too_long';
  return null;
}
