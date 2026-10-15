import { MAX_EXPRESSION_LENGTH } from './input';
import { BACKSPACE, CLEAR, INITIAL_STATE, pressKey, SQRT, type CalculatorState } from './keys';

const typed = (expression: string): CalculatorState => ({ expression, result: null });
const computed = (expression: string, result: string): CalculatorState => ({ expression, result });

describe('pressKey', () => {
  it('appends characters to the expression', () => {
    expect(pressKey(typed('1+'), '2')).toEqual(typed('1+2'));
  });

  it('clears everything', () => {
    expect(pressKey(computed('1+2', '3'), CLEAR)).toEqual(INITIAL_STATE);
  });

  it('deletes the last character', () => {
    expect(pressKey(typed('12+3'), BACKSPACE)).toEqual(typed('12+'));
    expect(pressKey(INITIAL_STATE, BACKSPACE)).toEqual(INITIAL_STATE);
  });

  it('inserts the square-root function and its parenthesis with one key', () => {
    expect(pressKey(typed('2*'), SQRT)).toEqual(typed('2*sqrt('));
  });

  it('deletes the whole "sqrt(" with one backspace, and otherwise one character', () => {
    expect(pressKey(typed('2*sqrt('), BACKSPACE)).toEqual(typed('2*'));
    expect(pressKey(typed('sqrt(16'), BACKSPACE)).toEqual(typed('sqrt(1'));
    expect(pressKey(typed('sqrt(16)'), BACKSPACE)).toEqual(typed('sqrt(16'));
  });

  it('edits the original expression when deleting after a result', () => {
    expect(pressKey(computed('12+3', '15'), BACKSPACE)).toEqual(typed('12+'));
  });

  describe('after a result', () => {
    it.each(['+', '-', '*', '/', '^', '%'])('continues from the result with %s', (op) => {
      expect(pressKey(computed('12+3', '15'), op)).toEqual(typed(`15${op}`));
    });

    it('continues from a negative or decimal result', () => {
      expect(pressKey(computed('0-5', '-5'), '*')).toEqual(typed('-5*'));
      expect(pressKey(computed('10/4', '2.5'), '+')).toEqual(typed('2.5+'));
    });

    it.each(['7', '.', '(', ')', SQRT])('starts a new expression with %s', (key) => {
      expect(pressKey(computed('12+3', '15'), key)).toEqual(typed(key));
    });

    it('ignores operators when the result is in exponent form', () => {
      const state = computed('10^21', '1e+21');
      expect(pressKey(state, '+')).toBe(state);
    });
  });

  it('stops appending at the maximum length', () => {
    const full = typed('1'.repeat(MAX_EXPRESSION_LENGTH));
    expect(pressKey(full, '1')).toBe(full);
  });

  it('does not let a multi-character key overflow the maximum length', () => {
    const almostFull = typed('1'.repeat(MAX_EXPRESSION_LENGTH - SQRT.length + 1));
    expect(pressKey(almostFull, SQRT)).toBe(almostFull);

    const fits = typed('1'.repeat(MAX_EXPRESSION_LENGTH - SQRT.length));
    expect(pressKey(fits, SQRT).expression).toHaveLength(MAX_EXPRESSION_LENGTH);
  });
});
