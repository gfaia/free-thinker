export interface QueryTask {
  id: number
  keyword: string
  platform: string
  last_run: string
  status: string
}

export interface Article {
  id: number
  url: string
  title: string
  author: string
  published_at: string
  source: string
  query_keyword: string
  content_path: string
  summary: string
  created_at: string
}

export interface ListResponse<T> {
  items: T[]
  total?: number
  limit?: number
  offset?: number
}
