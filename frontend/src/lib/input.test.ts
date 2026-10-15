import { checkInput, MAX_EXPRESSION_LENGTH, normalizeOperators } from './input';

describe('normalizeOperators', () => {
  it('maps typographic operators to ASCII', () => {
    expect(normalizeOperators('6 × 7 ÷ 2 − 1')).toBe('6 * 7 / 2 - 1');
  });

  it('leaves ASCII input untouched', () => {
    expect(normalizeOperators('(1+2)*3/4-5')).toBe('(1+2)*3/4-5');
  });
});

describe('checkInput', () => {
  it.each(['', '1+1', ' (12 + 3) * 4 ', '-.5 / 2', '1\t+ 1'])('accepts %j', (input) => {
    expect(checkInput(input)).toBeNull();
  });

  it.each(['2 + abc', '1;2', 'sqrt(4)', '1 + 1\n', '1\u00a0+ 1', '1 × 2', '1e5'])(
    'flags invalid characters in %j',
    (input) => {
      expect(checkInput(input)).toBe('invalid_characters');
    },
  );

  it('accepts the maximum length and flags anything longer', () => {
    expect(checkInput('1'.repeat(MAX_EXPRESSION_LENGTH))).toBeNull();
    expect(checkInput('1'.repeat(MAX_EXPRESSION_LENGTH + 1))).toBe('too_long');
  });
});
