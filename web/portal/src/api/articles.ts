import { getJSON, getText } from './client'
import type { Article, ListResponse } from '../types'

export function listArticles(params: URLSearchParams): Promise<ListResponse<Article>> {
  return getJSON<ListResponse<Article>>(`/api/articles?${params.toString()}`)
}

export function getArticle(id: string): Promise<Article> {
  return getJSON<Article>(`/api/articles/${id}`)
}

export function getArticleContent(id: string): Promise<string> {
  return getText(`/api/articles/${id}/content`)
}
