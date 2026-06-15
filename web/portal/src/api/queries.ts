import { getJSON } from './client'
import type { ListResponse, QueryTask } from '../types'

export function listQueries(params: URLSearchParams): Promise<ListResponse<QueryTask>> {
  return getJSON<ListResponse<QueryTask>>(`/api/queries?${params.toString()}`)
}
