import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import App from './App';

const fetchMock = vi.fn<typeof fetch>();

beforeEach(() => {
  fetchMock.mockReset();
  vi.stubGlobal('fetch', fetchMock);
});

afterEach(() => {
  vi.unstubAllGlobals();
});

function json(body: unknown, status = 200): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { 'Content-Type': 'application/json' },
  });
}

const key = (name: string) => screen.getByRole('button', { name });
const expression = () => screen.getByLabelText('Expression');

async function press(user: ReturnType<typeof userEvent.setup>, ...names: string[]) {
  for (const name of names) await user.click(key(name));
}

describe('App', () => {
  it('builds an expression with the keypad and shows the backend result', async () => {
    fetchMock.mockResolvedValue(json({ expression: '12+3', result: 15 }));
    const user = userEvent.setup();
    render(<App />);

    await press(user, '1', '2', 'add', '3', 'equals');

    expect(await screen.findByRole('status')).toHaveTextContent('= 15');
    expect(JSON.parse(String(fetchMock.mock.calls[0]?.[1]?.body))).toEqual({ expression: '12+3' });
  });

  it('submits a typed expression with Enter', async () => {
    fetchMock.mockResolvedValue(json({ expression: '10/4', result: 2.5 }));
    const user = userEvent.setup();
    render(<App />);

    await user.type(expression(), '10/4{Enter}');

    expect(await screen.findByRole('status')).toHaveTextContent('= 2.5');
  });

  it('converts typographic operators typed into the field', async () => {
    const user = userEvent.setup();
    render(<App />);

    await user.type(expression(), '6×7');

    expect(expression()).toHaveValue('6*7');
  });

  it('shows the backend error message and no result', async () => {
    fetchMock.mockResolvedValue(
      json({ error: { code: 'division_by_zero', message: 'Division by zero.' } }, 400),
    );
    const user = userEvent.setup();
    render(<App />);

    await user.type(expression(), '1/0{Enter}');

    expect(await screen.findByRole('alert')).toHaveTextContent('Division by zero.');
    expect(screen.getByRole('status')).not.toHaveTextContent('=');
  });

  it('shows a friendly message when the backend is unreachable', async () => {
    fetchMock.mockRejectedValue(new TypeError('Failed to fetch'));
    const user = userEvent.setup();
    render(<App />);

    await user.type(expression(), '1+1{Enter}');

    expect(await screen.findByRole('alert')).toHaveTextContent(/could not reach the calculator/i);
  });

  it('warns about invalid characters and does not call the backend', async () => {
    const user = userEvent.setup();
    render(<App />);

    await user.type(expression(), '2+a{Enter}');

    expect(screen.getByText(/only digits/i)).toBeVisible();
    expect(expression()).toHaveAttribute('aria-invalid', 'true');
    expect(key('equals')).toBeDisabled();
    expect(fetchMock).not.toHaveBeenCalled();
  });

  it('disables equals while the expression is empty', () => {
    render(<App />);

    expect(key('equals')).toBeDisabled();
  });

  it('continues from the result after an operator and starts fresh after a digit', async () => {
    fetchMock.mockResolvedValue(json({ expression: '12+3', result: 15 }));
    const user = userEvent.setup();
    render(<App />);
    await press(user, '1', '2', 'add', '3', 'equals');
    await screen.findByText('= 15');

    await press(user, 'multiply');
    expect(expression()).toHaveValue('15*');

    fetchMock.mockResolvedValue(json({ expression: '15*2', result: 30 }));
    await press(user, '2', 'equals');
    await screen.findByText('= 30');

    await press(user, '7');
    expect(expression()).toHaveValue('7');
  });

  it('clears and deletes with C and backspace', async () => {
    const user = userEvent.setup();
    render(<App />);

    await press(user, '1', '2', '3', 'backspace');
    expect(expression()).toHaveValue('12');

    await press(user, 'clear');
    expect(expression()).toHaveValue('');
  });

  it('locks the keypad while a calculation is in flight', async () => {
    let finish: (response: Response) => void = () => {};
    fetchMock.mockReturnValue(new Promise<Response>((resolve) => (finish = resolve)));
    const user = userEvent.setup();
    render(<App />);

    await user.type(expression(), '1+1{Enter}');

    expect(screen.getByRole('status')).toHaveTextContent('Calculating…');
    expect(key('7')).toBeDisabled();
    expect(key('equals')).toBeDisabled();

    finish(json({ expression: '1+1', result: 2 }));

    await waitFor(() => expect(screen.getByRole('status')).toHaveTextContent('= 2'));
    expect(key('7')).toBeEnabled();
  });

  it('clears an old error as soon as the expression is edited', async () => {
    fetchMock.mockResolvedValue(
      json({ error: { code: 'invalid_syntax', message: 'Unexpected end of expression.' } }, 400),
    );
    const user = userEvent.setup();
    render(<App />);
    await user.type(expression(), '1+{Enter}');
    await screen.findByRole('alert');

    await user.type(expression(), '1');

    expect(screen.queryByRole('alert')).not.toBeInTheDocument();
  });
});
