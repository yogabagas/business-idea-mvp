export interface FilterConfig {
  blocklist: string[]
  repetitionThreshold: number
  linkDetection: boolean
  customRegex: string
  hideFiltered: boolean
}

export interface ChatMessage {
  id: string
  text: string
  authorName: string
  authorId: string
  profileImageUrl?: string
  publishedAt: string
  isSpam: boolean
  reasons?: string[]
}

export interface MessagesResponse {
  messages: ChatMessage[]
  nextPageToken: string
  pollingIntervalMillis: number
}

export const defaultFilterConfig: FilterConfig = {
  blocklist: [],
  repetitionThreshold: 0,
  linkDetection: true,
  customRegex: '',
  hideFiltered: false,
}
