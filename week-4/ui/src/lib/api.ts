const BASE_URL = import.meta.env.VITE_API_BASE_URL
const TOKEN_KEY = 'paas_token'

export function getToken(): string | null {
  return localStorage.getItem(TOKEN_KEY)
}

export function setToken(token: string | null): void {
  if (token) {
    localStorage.setItem(TOKEN_KEY, token)
  } else {
    localStorage.removeItem(TOKEN_KEY)
  }
}

export class ApiError extends Error {
  constructor(
    public status: number,
    message: string,
  ) {
    super(message)
    this.name = 'ApiError'
  }
}

async function request<T>(path: string, options: RequestInit = {}): Promise<T> {
  const headers = new Headers(options.headers)
  headers.set('Content-Type', 'application/json')
  const token = getToken()
  if (token) {
    headers.set('Authorization', `Bearer ${token}`)
  }

  const res = await fetch(`${BASE_URL}${path}`, { ...options, headers })

  if (!res.ok) {
    let message = res.statusText
    try {
      const body = (await res.json()) as { message?: string }
      if (body.message) message = body.message
    } catch {
      // response had no JSON body
    }
    throw new ApiError(res.status, message)
  }

  if (res.status === 202 || res.status === 204) {
    return undefined as T
  }
  return (await res.json()) as T
}

export interface Credentials {
  email: string
  password: string
}

export interface RegisterRequest extends Credentials {
  firstName: string
  lastName: string
}

export interface AuthResponse {
  token: string
  expiresAt: string
}

export type InstancePhase = 'Provisioning' | 'Healthy' | 'Degraded' | 'Deleting'

export interface Instance {
  id: string
  name: string
  phase: InstancePhase
  instances: number
  readyInstances: number
  /** Has a dedicated external LoadBalancer Service — doesn't by itself mean the IP is assigned yet. Absent on older API deploys. */
  external?: boolean
  createdAt: string
}

export interface CreateInstanceRequest {
  name: string
  instances: number
  storageSize: string
  /** IP networks in CIDR notation allowed to reach this instance externally. Omit/empty = internal-only. */
  allowedIPs?: string[]
}

export interface ConnectionInfo {
  /** External LoadBalancer IP if the instance is external, otherwise an in-cluster DNS name. */
  host: string
  port: number
  database: string
  username: string
  password: string
}

export const authApi = {
  register: (body: RegisterRequest) =>
    request<AuthResponse>('/v1/auth/register', { method: 'POST', body: JSON.stringify(body) }),
  login: (body: Credentials) =>
    request<AuthResponse>('/v1/auth/login', { method: 'POST', body: JSON.stringify(body) }),
}

export const instancesApi = {
  list: () => request<Instance[]>('/v1/instances'),
  create: (body: CreateInstanceRequest) =>
    request<Instance>('/v1/instances', { method: 'POST', body: JSON.stringify(body) }),
  get: (id: string) => request<Instance>(`/v1/instances/${encodeURIComponent(id)}`),
  remove: (id: string) =>
    request<void>(`/v1/instances/${encodeURIComponent(id)}`, { method: 'DELETE' }),
  connection: (id: string) =>
    request<ConnectionInfo>(`/v1/instances/${encodeURIComponent(id)}/connection`),
}
