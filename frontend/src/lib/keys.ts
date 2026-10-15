import { MAX_EXPRESSION_LENGTH } from './input';

export const CLEAR = 'clear';
export const BACKSPACE = 'backspace';
export const EQUALS = 'equals';

const OPERATORS = new Set(['+', '-', '*', '/']);
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
    return { expression: state.expression.slice(0, -1), result: null };
  }

  if (state.result !== null) {
    if (!OPERATORS.has(key)) return { expression: key, result: null };
    // Results in exponent form (1e+21) are not valid input for the backend.
    if (!PLAIN_NUMBER.test(state.result)) return state;
    return { expression: state.result + key, result: null };
  }

  if (state.expression.length >= MAX_EXPRESSION_LENGTH) return state;
  return { expression: state.expression + key, result: null };
}
