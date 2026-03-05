import type { Message, StatusResponse, ErrorResponse } from '../types'

const API_BASE = import.meta.env.VITE_API_URL || ''

async function request<T>(endpoint: string, options?: RequestInit): Promise<T> {
  const res = await fetch(`${API_BASE}${endpoint}`, options)
  if (!res.ok) {
    const err: ErrorResponse = await res.json().catch(() => ({ error: res.statusText }))
    throw { status: res.status, ...err }
  }
  return res.json()
}

export function fetchMessages(limit = 50, before?: string): Promise<Message[]> {
  const params = new URLSearchParams({ limit: String(limit) })
  if (before) params.set('before', before)
  return request<Message[]>(`/api/messages?${params}`)
}

export function fetchMessagesSince(afterId: string): Promise<Message[]> {
  return request<Message[]>(`/api/messages/since?after=${afterId}`)
}

export function postMessage(content: string): Promise<Message> {
  return request<Message>('/api/messages', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ content }),
  })
}

export function fetchStatus(): Promise<StatusResponse> {
  return request<StatusResponse>('/api/status')
}
