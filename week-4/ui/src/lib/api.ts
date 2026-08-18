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
  /** Kubernetes object UID — stable, globally unique, distinct from the name-based `id`. Absent on older API deploys. */
  uid?: string
  name: string
  phase: InstancePhase
  instances: number
  readyInstances: number
  /** Has a dedicated external LoadBalancer Service — doesn't by itself mean the IP is assigned yet. Absent on older API deploys. */
  external?: boolean
  /** CIDR ranges currently allowed to reach this instance externally. Absent/empty when internal-only. */
  allowedIPs?: string[]
  /** Postgres version tag currently running. Empty until the operator reports it. Absent on older API deploys. */
  version?: string
  /** Disk size per pod, as requested at creation. Absent on older API deploys. */
  storageSize?: string
  createdAt: string
}

/** Kubernetes StorageClasses offered for a new instance's volumes — a fixed selection, not free text. */
export const STORAGE_CLASSES = ['premium-perf4-stackit'] as const

export interface CreateInstanceRequest {
  name: string
  instances: number
  storageSize: string
  storageClass: string
  database: string
  username: string
  /** IP networks in CIDR notation allowed to reach this instance externally. Omit/empty = internal-only. */
  allowedIPs?: string[]
}

export interface UpdateInstanceRequest {
  /** New desired pod count. Omit to leave unchanged. */
  instances?: number
  /** New disk size per pod — must be strictly greater than the current size. Omit to leave unchanged. */
  storageSize?: string
  /** Replaces the full allowedIPs list. Empty array removes external access; omit to leave as-is. */
  allowedIPs?: string[]
}

export interface ServicePort {
  name?: string
  port?: number
  protocol?: string
}

/** One of the Postgres endpoints CNPG exposes for an instance (e.g. `<name>-rw`, `-ro`, `-r`). */
export interface ServiceInfo {
  name?: string
  type?: string
  clusterIP?: string
  externalIP?: string
  ports?: ServicePort[]
}

export interface ConnectionInfo {
  /** External LoadBalancer IP if the instance is external, otherwise an in-cluster DNS name. */
  host: string
  port: number
  database: string
  username: string
  password: string
}

export type AuditAction =
  | 'user.registered'
  | 'user.login'
  | 'login.failed'
  | 'instance.created'
  | 'instance.updated'
  | 'instance.deleted'
  | 'credentials.viewed'

/** One recorded action, newest-first. A permanent per-account history —
 * distinct from operational/infrastructure logging, scoped to only the
 * authenticated user's own activity. */
export interface AuditLogEntry {
  id: number
  action: AuditAction
  /** Absent for account-level events (registration, login) not tied to a specific instance. */
  instanceName?: string
  /** Action-specific detail, e.g. which fields changed on an update. */
  metadata?: Record<string, unknown>
  createdAt: string
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
  update: (id: string, body: UpdateInstanceRequest) =>
    request<Instance>(`/v1/instances/${encodeURIComponent(id)}`, {
      method: 'PATCH',
      body: JSON.stringify(body),
    }),
  remove: (id: string) =>
    request<void>(`/v1/instances/${encodeURIComponent(id)}`, { method: 'DELETE' }),
  connection: (id: string) =>
    request<ConnectionInfo>(`/v1/instances/${encodeURIComponent(id)}/connection`),
  services: (id: string) =>
    request<ServiceInfo[]>(`/v1/instances/${encodeURIComponent(id)}/services`),
}

export const auditLogsApi = {
  list: (params: { limit?: number; offset?: number } = {}) =>
    request<AuditLogEntry[]>(`/v1/audit-logs?${new URLSearchParams(params as Record<string, string>)}`),
  listForInstance: (id: string, params: { limit?: number; offset?: number } = {}) =>
    request<AuditLogEntry[]>(
      `/v1/instances/${encodeURIComponent(id)}/audit-logs?${new URLSearchParams(params as Record<string, string>)}`,
    ),
}
