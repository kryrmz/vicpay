import { createHttpApi } from './http'
import { mockApi } from './mock'
import type { Api } from './types'

export * from './types'

// Selecciona el adapter:
//  - Si VITE_API_BASE esta definido (p.ej. "/api" tras el proxy BFF), habla con
//    el backend real en esa base.
//  - En un build de produccion sin esa variable, usa "/api" por defecto (evita
//    publicar accidentalmente el mock).
//  - En desarrollo sin la variable, usa el mock en memoria (front sin backend).
const explicitBase = import.meta.env.VITE_API_BASE as string | undefined
const apiBase = explicitBase ?? (import.meta.env.PROD ? '/api' : undefined)

export const api: Api = apiBase ? createHttpApi(apiBase) : mockApi
