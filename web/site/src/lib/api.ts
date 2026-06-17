import type { APIResponse, Content, Inquiry } from './types'

async function request<T>(url: string, options?: RequestInit): Promise<T> {
  const res = await fetch(url, {
    headers: { 'Content-Type': 'application/json', ...options?.headers },
    ...options,
  })
  const body: APIResponse<T> = await res.json()
  if (!res.ok || body.code !== 0) {
    throw new Error(body.message || '요청에 실패했습니다')
  }
  return body.data as T
}

export function fetchContent(): Promise<Content> {
  return request<Content>('/api/content')
}

export function submitContact(data: { name: string; email: string; message: string }): Promise<void> {
  return request<void>('/api/contact', {
    method: 'POST',
    body: JSON.stringify(data),
  })
}

export function adminLogin(username: string, password: string): Promise<{ token: string }> {
  return request<{ token: string }>('/api/admin/login', {
    method: 'POST',
    body: JSON.stringify({ username, password }),
  })
}

export function adminUpdateContent(content: Content, token: string): Promise<Content> {
  return request<Content>('/api/admin/content', {
    method: 'PUT',
    headers: { Authorization: `Bearer ${token}` },
    body: JSON.stringify(content),
  })
}

export function adminFetchInquiries(token: string): Promise<Inquiry[]> {
  return request<Inquiry[]>('/api/admin/inquiries', {
    headers: { Authorization: `Bearer ${token}` },
  })
}
