export interface Position {
  id: string
  symbol: string
  side: "buy" | "sell"
  volume_lots: number
  entry_price: number
  current_price: number
  stop_loss?: number
  take_profit?: number
  unrealized_pnl: number
  opened_at: string
}

export interface Trade {
  id: string
  symbol: string
  side: "buy" | "sell"
  volume_lots: number
  entry_price: number
  exit_price: number
  realized_pnl: number
  close_reason: string
  closed_at: string
}

export interface AccountSnapshot {
  mode: "paper"
  as_of: string
  account: { balance: number; currency: string }
  equity: number
  daily_pnl: number
  unrealized_pnl: number
  open_positions: Position[]
  recent_trades: Trade[]
}

export interface Proposal {
  id: string
  open_price: number
  expires_at: string
  request: {
    symbol: string
    side: "buy" | "sell"
    volume_lots: number
    stop_loss?: number
    take_profit?: number
  }
}

export class PaperAPIError extends Error {
  code: string
  field?: string

  constructor(code: string, message: string, field?: string) {
    super(message)
    this.code = code
    this.field = field
  }
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(path, {
    ...init,
    headers: { "Content-Type": "application/json", ...init?.headers },
  })
  const payload = (await response.json()) as T & {
    error?: { code: string; message: string; field?: string }
  }
  if (!response.ok) {
    throw new PaperAPIError(
      payload.error?.code ?? "INTERNAL_ERROR",
      payload.error?.message ?? "ไม่สามารถเชื่อมต่อบริการลองเทรดได้",
      payload.error?.field,
    )
  }
  return payload
}

export const paperAPI = {
  snapshot: () => request<AccountSnapshot>("/api/paper/account"),
  prepare: (input: Record<string, unknown>) =>
    request<Proposal>("/api/paper/trade/prepare", {
      method: "POST",
      body: JSON.stringify(input),
    }),
  confirm: (proposalId: string) =>
    request<{ position: Position }>("/api/paper/trade/confirm", {
      method: "POST",
      body: JSON.stringify({
        proposal_id: proposalId,
        client_request_id: crypto.randomUUID(),
      }),
    }),
  close: (positionId: string) =>
    request<Trade>(`/api/paper/positions/${positionId}/close`, {
      method: "POST",
      body: JSON.stringify({
        client_request_id: crypto.randomUUID(),
        reason: "manual",
      }),
    }),
  protect: (positionId: string, stopLoss?: number, takeProfit?: number) =>
    request<Position>(`/api/paper/positions/${positionId}/protection`, {
      method: "POST",
      body: JSON.stringify({ stop_loss: stopLoss, take_profit: takeProfit }),
    }),
  reset: () =>
    request<AccountSnapshot>("/api/paper/account/reset", {
      method: "POST",
      body: "{}",
    }),
}
