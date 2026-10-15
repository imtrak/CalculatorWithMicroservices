import { MAX_EXPRESSION_LENGTH } from './input';

export const CLEAR = 'clear';
export const BACKSPACE = 'backspace';
export const EQUALS = 'equals';
/** The square-root key inserts the function name and its opening parenthesis. */
export const SQRT = 'sqrt(';

// Keys that continue from a result instead of starting a new expression.
const OPERATORS = new Set(['+', '-', '*', '/', '^', '%']);
const PLAIN_NUMBER = /^-?\d+(\.\d+)?$/;

export interface CalculatorState {
  expression: string;
  /** The last computed result, shown until the expression is edited. */
  result: string | null;
}

export const INITIAL_STATE: CalculatorState = { expression: '', result: null };

/**
 * Applies a keypad key (other than "equals", which needs the backend) to the
 * state. After a result, an operator continues from that result and any other
 * character starts a new expression, like a pocket calculator.
 */
export function pressKey(state: CalculatorState, key: string): CalculatorState {
  if (key === CLEAR) return INITIAL_STATE;

  if (key === BACKSPACE) {
    // "sqrt(" went in with one key press, so it comes out with one too.
    const length = state.expression.endsWith(SQRT) ? SQRT.length : 1;
    return { expression: state.expression.slice(0, -length), result: null };
  }

  if (state.result !== null) {
    if (!OPERATORS.has(key)) return { expression: key, result: null };
    // Results in exponent form (1e+21) are not valid input for the backend.
    if (!PLAIN_NUMBER.test(state.result)) return state;
    return { expression: state.result + key, result: null };
  }

  if (state.expression.length + key.length > MAX_EXPRESSION_LENGTH) return state;
  return { expression: state.expression + key, result: null };
}
