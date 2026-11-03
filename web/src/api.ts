export interface Entry {
  id: number
  spelling: string
  definition: string
  createdAt: string
}

interface ErrorBody {
  error: string
  existingEntry?: Entry
}

export class ApiError extends Error {
  constructor(
    public status: number,
    public body: ErrorBody,
  ) {
    super(body.error)
  }
}

async function request<T>(path: string, options?: RequestInit): Promise<T> {
  const response = await fetch(`/api${path}`, options)
  if (!response.ok) {
    throw new ApiError(response.status, await response.json())
  }
  return response.json() as Promise<T>
}

export const listEntries = (sort: string, query: string) =>
  request<Entry[]>(`/entries?${new URLSearchParams({ sort, q: query })}`)
export const getEntry = (id: string | number) =>
  request<Entry>(`/entries/${encodeURIComponent(id)}`)
export const createEntry = (spelling: string, definition: string) =>
  request<Entry>('/entries', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ spelling, definition }),
  })
export const updateEntry = (id: string | number, spelling: string, definition: string) =>
  request<Entry>(`/entries/${encodeURIComponent(id)}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ spelling, definition }),
  })

export function errorMessage(error: unknown): string {
  return error instanceof ApiError
    ? error.message
    : '暂时无法连接生词本，请确认程序仍在运行后重试。'
}

export function formatDate(value: string): string {
  return new Intl.DateTimeFormat('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
  }).format(new Date(value))
}
