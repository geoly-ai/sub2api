import { apiClient } from './client'

export interface UserMessage {
  id: number
  user_id: number
  type: string
  title: string
  content: string
  status: string
  metadata: Record<string, unknown>
  read_at?: string | null
  created_at: string
  updated_at: string
}

export interface UserMessagesResponse {
  items: UserMessage[]
  total: number
  page: number
  page_size: number
  pages: number
}

export async function listMessages(params: { page?: number; page_size?: number; unread_only?: boolean } = {}): Promise<UserMessagesResponse> {
  const { data } = await apiClient.get<UserMessagesResponse>('/user/messages', { params })
  return data
}

export async function markMessageRead(id: number): Promise<{ message: string }> {
  const { data } = await apiClient.post<{ message: string }>(`/user/messages/${id}/read`)
  return data
}

export const messagesAPI = {
  listMessages,
  markMessageRead,
}

export default messagesAPI
