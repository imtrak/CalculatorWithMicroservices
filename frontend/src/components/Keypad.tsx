import { BACKSPACE, CLEAR, EQUALS } from '../lib/keys';

type KeyKind = 'digit' | 'operator' | 'action' | 'equals';

interface KeyDefinition {
  /** Value passed to onKey: the character to insert, or an action name. */
  value: string;
  label: string;
  /** Accessible name; defaults to the label. */
  name?: string;
  kind: KeyKind;
}

const KEYS: readonly KeyDefinition[] = [
  { value: CLEAR, label: 'C', name: 'clear', kind: 'action' },
  { value: BACKSPACE, label: '⌫', name: 'backspace', kind: 'action' },
  { value: '(', label: '(', name: 'open parenthesis', kind: 'action' },
  { value: ')', label: ')', name: 'close parenthesis', kind: 'action' },
  { value: '7', label: '7', kind: 'digit' },
  { value: '8', label: '8', kind: 'digit' },
  { value: '9', label: '9', kind: 'digit' },
  { value: '/', label: '÷', name: 'divide', kind: 'operator' },
  { value: '4', label: '4', kind: 'digit' },
  { value: '5', label: '5', kind: 'digit' },
  { value: '6', label: '6', kind: 'digit' },
  { value: '*', label: '×', name: 'multiply', kind: 'operator' },
  { value: '1', label: '1', kind: 'digit' },
  { value: '2', label: '2', kind: 'digit' },
  { value: '3', label: '3', kind: 'digit' },
  { value: '-', label: '−', name: 'subtract', kind: 'operator' },
  { value: '0', label: '0', kind: 'digit' },
  { value: '.', label: '.', name: 'decimal point', kind: 'digit' },
  { value: EQUALS, label: '=', name: 'equals', kind: 'equals' },
  { value: '+', label: '+', name: 'add', kind: 'operator' },
];

interface KeypadProps {
  /** Disables every key, e.g. while a calculation is in flight. */
  disabled: boolean;
  /** Whether "=" can be pressed. */
  canSubmit: boolean;
  onKey: (value: string) => void;
}

export function Keypad({ disabled, canSubmit, onKey }: KeypadProps) {
  return (
    <div className="keypad">
      {KEYS.map((key) => (
        <button
          key={key.value}
          type="button"
          className={`key key-${key.kind}`}
          aria-label={key.name}
          disabled={disabled || (key.kind === 'equals' && !canSubmit)}
          onClick={() => onKey(key.value)}
        >
          {key.label}
        </button>
      ))}
    </div>
  );
}
