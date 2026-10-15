import { useEffect, useId, useRef, useState, type FormEvent } from 'react';
import { ApiError, calculate } from './api/calculatorApi';
import { Keypad } from './components/Keypad';
import { checkInput, INPUT_ISSUE_MESSAGES, normalizeOperators } from './lib/input';
import { EQUALS, INITIAL_STATE, pressKey, type CalculatorState } from './lib/keys';

export default function App() {
  const [state, setState] = useState<CalculatorState>(INITIAL_STATE);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const inFlight = useRef<AbortController | null>(null);
  const inputId = useId();
  const hintId = useId();

  useEffect(() => () => inFlight.current?.abort(), []);

  const issue = checkInput(state.expression);
  const canSubmit = !loading && state.expression.trim() !== '' && issue === null;

  function edit(next: CalculatorState) {
    setState(next);
    setError(null);
  }

  async function submit() {
    if (!canSubmit) return;

    const controller = new AbortController();
    inFlight.current = controller;
    setLoading(true);
    setError(null);

    try {
      const value = await calculate(state.expression, { signal: controller.signal });
      setState((current) => ({ ...current, result: String(value) }));
    } catch (err) {
      if (controller.signal.aborted) return;
      setError(err instanceof ApiError ? err.message : 'Something went wrong. Please try again.');
    } finally {
      if (inFlight.current === controller) setLoading(false);
    }
  }

  function handleKey(key: string) {
    if (key === EQUALS) {
      void submit();
    } else {
      edit(pressKey(state, key));
    }
  }

  function handleSubmit(event: FormEvent) {
    event.preventDefault();
    void submit();
  }

  return (
    <main className="app">
      <h1>Calculator</h1>

      <form className="calculator" onSubmit={handleSubmit} noValidate>
        <label htmlFor={inputId}>Expression</label>
        <input
          id={inputId}
          type="text"
          value={state.expression}
          placeholder="(12 + 3) × 4"
          autoComplete="off"
          spellCheck={false}
          aria-invalid={issue !== null}
          aria-describedby={hintId}
          onChange={(event) =>
            edit({ expression: normalizeOperators(event.target.value), result: null })
          }
        />

        <p id={hintId} className="hint">
          {issue ? INPUT_ISSUE_MESSAGES[issue] : ' '}
        </p>

        <output className="result" aria-live="polite">
          {loading ? 'Calculating…' : state.result !== null ? `= ${state.result}` : ' '}
        </output>

        {error && (
          <p role="alert" className="error">
            {error}
          </p>
        )}

        <Keypad disabled={loading} canSubmit={canSubmit} onKey={handleKey} />
      </form>
    </main>
  );
}
