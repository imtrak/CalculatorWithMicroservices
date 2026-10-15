const CALCULATIONS_URL = '/api/v1/calculations';

/** A failure to obtain a result, with a message that is safe to show to users. */
export class ApiError extends Error {
  constructor(
    message: string,
    readonly code: string,
    readonly status: number,
  ) {
    super(message);
    this.name = 'ApiError';
  }
}

interface ErrorBody {
  error: { code: string; message: string };
}

function isErrorBody(value: unknown): value is ErrorBody {
  if (typeof value !== 'object' || value === null || !('error' in value)) return false;
  const { error } = value;
  return (
    typeof error === 'object' &&
    error !== null &&
    'code' in error &&
    'message' in error &&
    typeof error.code === 'string' &&
    typeof error.message === 'string'
  );
}

function isResultBody(value: unknown): value is { result: number } {
  return (
    typeof value === 'object' &&
    value !== null &&
    'result' in value &&
    typeof value.result === 'number' &&
    Number.isFinite(value.result)
  );
}

async function readJson(response: Response): Promise<unknown> {
  try {
    return await response.json();
  } catch {
    return undefined;
  }
}

/** Evaluates expression on the backend and returns the numeric result. */
export async function calculate(
  expression: string,
  options: { signal?: AbortSignal } = {},
): Promise<number> {
  let response: Response;
  try {
    response = await fetch(CALCULATIONS_URL, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', Accept: 'application/json' },
      body: JSON.stringify({ expression }),
      signal: options.signal,
    });
  } catch (err) {
    if (err instanceof DOMException && err.name === 'AbortError') throw err;
    throw new ApiError(
      'Could not reach the calculator. Check your connection and try again.',
      'network_error',
      0,
    );
  }

  const body = await readJson(response);

  if (response.ok) {
    if (isResultBody(body)) return body.result;
    throw new ApiError('The calculator sent an unexpected response.', 'unexpected_response', response.status);
  }

  if (isErrorBody(body)) {
    throw new ApiError(body.error.message, body.error.code, response.status);
  }
  throw new ApiError(
    'The calculator is unavailable right now. Please try again.',
    'unexpected_response',
    response.status,
  );
}
