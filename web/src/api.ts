import { ref } from 'vue'

export const notebookRevision = ref(0)

export interface Entry {
  id: number
  spelling: string
  definition: string
  createdAt: string
}

export interface Group {
  id: number
  type: string
  note: string
  createdAt: string
  members: Entry[]
}

interface ErrorBody {
  error: string
  existingEntry?: Entry
  existingGroup?: Group
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
  const result = await response.json() as T
  if (options?.method && options.method !== 'GET') notebookRevision.value++
  return result
}

export const getStats = () => request<{ entries: number; groups: number }>('/stats')

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
export const deleteEntry = (id: string | number) =>
  request<{ deleted: boolean }>(`/entries/${encodeURIComponent(id)}`, { method: 'DELETE' })

export const getGroupTypes = () => request<string[]>('/group-types')
export const search = (sort: string, query: string) =>
  request<{ entries: Entry[]; groups: Group[] }>(`/search?${new URLSearchParams({ sort, q: query })}`)
export const listGroupsByType = (type: string) =>
  request<Group[]>(`/groups?${new URLSearchParams({ type })}`)
export const listEntryGroups = (id: string | number) =>
  request<Group[]>('/entries/' + encodeURIComponent(id) + '/groups')
export const getGroup = (id: string | number) =>
  request<Group>('/groups/' + encodeURIComponent(id))
export const createGroup = (originId: number, type: string, note: string, memberIds: number[]) =>
  request<Group>('/entries/' + originId + '/groups', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ type, note, memberIds }),
  })
export const updateGroup = (id: number, type: string, note: string, memberIds: number[]) =>
  request<Group>('/groups/' + id, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ type, note, memberIds }),
  })
export const deleteGroup = (id: number) =>
  request<{ deleted: boolean }>('/groups/' + id, { method: 'DELETE' })

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
