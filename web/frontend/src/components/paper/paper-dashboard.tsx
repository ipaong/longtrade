import {
  IconAlertTriangle,
  IconRefresh,
  IconTrendingUp,
} from "@tabler/icons-react"
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { Link } from "@tanstack/react-router"
import { toast } from "sonner"

import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { paperAPI } from "@/features/paper/api"

const money = new Intl.NumberFormat("th-TH", {
  style: "currency",
  currency: "USD",
  minimumFractionDigits: 2,
})

function Pnl({ value }: { value: number }) {
  return (
    <span className={value >= 0 ? "text-emerald-600" : "text-red-600"}>
      {money.format(value)}
    </span>
  )
}

export function PaperDashboard() {
  const client = useQueryClient()
  const snapshot = useQuery({
    queryKey: ["paper-account"],
    queryFn: paperAPI.snapshot,
    refetchInterval: 5000,
  })
  const close = useMutation({
    mutationFn: paperAPI.close,
    onSuccess: () => {
      toast.success("ปิด Position แล้ว")
      void client.invalidateQueries({ queryKey: ["paper-account"] })
    },
    onError: (error: Error) => toast.error(error.message),
  })

  if (snapshot.isLoading)
    return (
      <div className="p-8" role="status">
        กำลังโหลดบัญชีทดลอง…
      </div>
    )
  if (snapshot.error)
    return (
      <div className="m-6 rounded-xl border border-red-200 bg-red-50 p-5 text-red-700">
        <IconAlertTriangle className="mb-2" />
        {snapshot.error.message}
      </div>
    )
  const data = snapshot.data!
  const metrics = [
    ["ยอดเงิน", data.account.balance],
    ["มูลค่าบัญชี", data.equity],
    ["กำไร/ขาดทุนวันนี้", data.daily_pnl],
    ["กำไร/ขาดทุนค้าง", data.unrealized_pnl],
  ] as const

  return (
    <div className="min-h-0 flex-1 overflow-y-auto bg-slate-50/60 p-4 md:p-8 dark:bg-slate-950/30">
      <div className="mx-auto max-w-6xl space-y-6">
        <header className="flex flex-wrap items-center justify-between gap-4">
          <div>
            <div className="mb-2 inline-flex items-center gap-2 rounded-full bg-amber-100 px-3 py-1 text-xs font-semibold text-amber-800">
              โหมดทดลอง • ไม่มีเงินจริง
            </div>
            <h1 className="text-2xl font-bold md:text-3xl">พอร์ตทดลอง</h1>
            <p className="text-muted-foreground">
              XAUUSD • อัปเดต {new Date(data.as_of).toLocaleTimeString("th-TH")}
            </p>
          </div>
          <div className="flex gap-2">
            <Button
              variant="outline"
              onClick={() => void snapshot.refetch()}
              aria-label="รีเฟรชพอร์ต"
            >
              <IconRefresh />
            </Button>
            <Button asChild>
              <Link to="/try-trade">
                <IconTrendingUp /> ลองเปิด Position
              </Link>
            </Button>
          </div>
        </header>
        <section className="grid grid-cols-2 gap-3 lg:grid-cols-4">
          {metrics.map(([label, value]) => (
            <Card key={label}>
              <CardHeader className="pb-1">
                <CardTitle className="text-muted-foreground text-xs font-medium md:text-sm">
                  {label}
                </CardTitle>
              </CardHeader>
              <CardContent className="text-lg font-bold md:text-2xl">
                {label.includes("กำไร") ? (
                  <Pnl value={value} />
                ) : (
                  money.format(value)
                )}
              </CardContent>
            </Card>
          ))}
        </section>
        <Card>
          <CardHeader>
            <CardTitle>
              Position ที่เปิดอยู่ ({data.open_positions.length})
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-3">
            {data.open_positions.length === 0 ? (
              <div className="text-muted-foreground rounded-xl border border-dashed p-8 text-center">
                ยังไม่มี Position — เริ่มลองเทรดโดยไม่ใช้เงินจริงได้เลย
              </div>
            ) : (
              data.open_positions.map((p) => (
                <article
                  key={p.id}
                  className="grid gap-3 rounded-xl border p-4 md:grid-cols-[1fr_auto_auto_auto] md:items-center"
                >
                  <div>
                    <strong>{p.symbol}</strong>
                    <span
                      className={`ml-2 rounded px-2 py-0.5 text-xs ${p.side === "buy" ? "bg-emerald-100 text-emerald-700" : "bg-red-100 text-red-700"}`}
                    >
                      {p.side.toUpperCase()}
                    </span>
                    <div className="text-muted-foreground text-sm">
                      {p.volume_lots} lot • เข้า {p.entry_price.toFixed(2)} •
                      ปัจจุบัน {p.current_price.toFixed(2)}
                    </div>
                  </div>
                  <Pnl value={p.unrealized_pnl} />
                  <span className="text-sm">
                    SL {p.stop_loss ?? "—"} / TP {p.take_profit ?? "—"}
                  </span>
                  <Button
                    variant="destructive"
                    size="sm"
                    disabled={close.isPending}
                    onClick={() => close.mutate(p.id)}
                  >
                    ปิด Position
                  </Button>
                </article>
              ))
            )}
          </CardContent>
        </Card>
        <Card>
          <CardHeader>
            <CardTitle>ประวัติล่าสุด</CardTitle>
          </CardHeader>
          <CardContent>
            {data.recent_trades.length === 0 ? (
              <p className="text-muted-foreground">ยังไม่มีประวัติการเทรด</p>
            ) : (
              <div className="overflow-x-auto">
                <table className="w-full min-w-xl text-sm">
                  <thead>
                    <tr className="border-b text-left">
                      <th className="py-2">สินทรัพย์</th>
                      <th>ฝั่ง</th>
                      <th>Lot</th>
                      <th>ราคาปิด</th>
                      <th>ผลลัพธ์</th>
                      <th>เวลา</th>
                    </tr>
                  </thead>
                  <tbody>
                    {data.recent_trades.map((t) => (
                      <tr key={t.id} className="border-b">
                        <td className="py-3 font-medium">{t.symbol}</td>
                        <td>{t.side.toUpperCase()}</td>
                        <td>{t.volume_lots}</td>
                        <td>{t.exit_price.toFixed(2)}</td>
                        <td>
                          <Pnl value={t.realized_pnl} />
                        </td>
                        <td>{new Date(t.closed_at).toLocaleString("th-TH")}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </CardContent>
        </Card>
      </div>
    </div>
  )
}
