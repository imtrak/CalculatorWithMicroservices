import { ApiError, calculate } from './calculatorApi';

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

async function failureOf(promise: Promise<unknown>): Promise<ApiError> {
  const err = await promise.then(
    () => undefined,
    (e: unknown) => e,
  );
  expect(err).toBeInstanceOf(ApiError);
  return err as ApiError;
}

describe('calculate', () => {
  it('posts the expression as JSON and returns the result', async () => {
    fetchMock.mockResolvedValue(json({ expression: '(12 + 3) * 4', result: 60 }));

    await expect(calculate('(12 + 3) * 4')).resolves.toBe(60);

    expect(fetchMock).toHaveBeenCalledOnce();
    const [url, init] = fetchMock.mock.calls[0] ?? [];
    expect(url).toBe('/api/v1/calculations');
    expect(init?.method).toBe('POST');
    expect(new Headers(init?.headers).get('Content-Type')).toBe('application/json');
    expect(JSON.parse(String(init?.body))).toEqual({ expression: '(12 + 3) * 4' });
  });

  it('turns a backend error into an ApiError with its code and message', async () => {
    fetchMock.mockResolvedValue(
      json({ error: { code: 'division_by_zero', message: 'Division by zero.' } }, 400),
    );

    const err = await failureOf(calculate('1/0'));

    expect(err).toMatchObject({ code: 'division_by_zero', message: 'Division by zero.', status: 400 });
  });

  it('does not expose non-JSON error pages', async () => {
    fetchMock.mockResolvedValue(new Response('<html>502 Bad Gateway</html>', { status: 502 }));

    const err = await failureOf(calculate('1+1'));

    expect(err.code).toBe('unexpected_response');
    expect(err.status).toBe(502);
    expect(err.message).not.toContain('html');
  });

  it('rejects a success response without a numeric result', async () => {
    fetchMock.mockResolvedValue(json({ expression: '1+1', result: '2' }));

    expect((await failureOf(calculate('1+1'))).code).toBe('unexpected_response');
  });

  it('reports network failures', async () => {
    fetchMock.mockRejectedValue(new TypeError('Failed to fetch'));

    const err = await failureOf(calculate('1+1'));

    expect(err).toMatchObject({ code: 'network_error', status: 0 });
  });

  it('passes the abort signal on and lets aborts through unchanged', async () => {
    const controller = new AbortController();
    fetchMock.mockRejectedValue(new DOMException('aborted', 'AbortError'));

    await expect(calculate('1+1', { signal: controller.signal })).rejects.toMatchObject({
      name: 'AbortError',
    });
    expect(fetchMock.mock.calls[0]?.[1]?.signal).toBe(controller.signal);
  });
});
