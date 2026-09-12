// API client for update availability checks.

export interface UpdateStatus {
  is_outdated: boolean
  current_version: string
  latest_version: string
  release_url: string
}

async function request<T>(path: string): Promise<T> {
  const res = await fetch(path)
  if (!res.ok) {
    throw new Error(`API error: ${res.status} ${res.statusText}`)
  }
  return res.json() as Promise<T>
}

export async function getUpdateStatus(): Promise<UpdateStatus> {
  return request<UpdateStatus>("/api/update/status")
}

export async function getVersion(): Promise<string> {
  const res = await request<{ version: string }>("/api/version")
  return res.version
}

export async function getGatewayBinaryVersion(): Promise<string> {
  const res = await request<{ version: string }>("/api/gateway/binary-version")
  return res.version
}

export interface ApplyUpdateResult {
  success: boolean
  up_to_date?: boolean
  version: string
  launcher_updated?: boolean
}

export class PermissionDeniedError extends Error {
  permissionDenied = true as const
  binaryPath: string
  constructor(msg: string, binaryPath: string) {
    super(msg)
    this.name = "PermissionDeniedError"
    this.binaryPath = binaryPath
  }
}

export async function applyUpdate(): Promise<ApplyUpdateResult> {
  const res = await fetch("/api/update/apply", { method: "POST" })
  if (!res.ok) {
    if (res.status === 403) {
      let body: { error?: string; permission_denied?: boolean; binary_path?: string } | null = null
      try { body = await res.json() } catch { /* ignore */ }
      if (body?.permission_denied) {
        throw new PermissionDeniedError(body.error ?? "Permission denied", body.binary_path ?? "")
      }
    }
    const text = await res.text()
    throw new Error(text || `HTTP ${res.status}`)
  }
  return res.json() as Promise<ApplyUpdateResult>
}
