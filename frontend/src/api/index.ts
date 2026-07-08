import { createHttpApi } from './http'
import { mockApi } from './mock'
import type { Api } from './types'

export * from './types'

// Selecciona el adapter: si VITE_API_BASE esta definido (p.ej. "/api" tras el
// proxy BFF), habla con el backend real; si no, usa el mock en memoria para
// desarrollar el front sin backend.
const apiBase = import.meta.env.VITE_API_BASE as string | undefined

export const api: Api = apiBase ? createHttpApi(apiBase) : mockApi
