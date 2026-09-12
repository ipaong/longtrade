import { useMutation, useQueryClient } from "@tanstack/react-query"
import { Link, useNavigate } from "@tanstack/react-router"
import { useState } from "react"
import { toast } from "sonner"

import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { type Proposal, paperAPI } from "@/features/paper/api"

export function TradeTicket() {
  const navigate = useNavigate()
  const client = useQueryClient()
  const [side, setSide] = useState<"buy" | "sell">("buy")
  const [volume, setVolume] = useState("0.01")
  const [sl, setSl] = useState("")
  const [tp, setTp] = useState("")
  const [proposal, setProposal] = useState<Proposal>()
  const prepare = useMutation({
    mutationFn: () =>
      paperAPI.prepare({
        symbol: "XAUUSD",
        side,
        volume_lots: Number(volume),
        ...(sl && { stop_loss: Number(sl) }),
        ...(tp && { take_profit: Number(tp) }),
      }),
    onSuccess: setProposal,
    onError: (e: Error) => toast.error(e.message),
  })
  const confirm = useMutation({
    mutationFn: () => paperAPI.confirm(proposal!.id),
    onSuccess: async () => {
      await client.invalidateQueries({ queryKey: ["paper-account"] })
      toast.success("เปิด Position ทดลองแล้ว")
      void navigate({ to: "/portfolio" })
    },
    onError: (e: Error) => {
      setProposal(undefined)
      toast.error(e.message)
    },
  })

  return (
    <div className="min-h-0 flex-1 overflow-y-auto bg-slate-50/60 p-4 md:p-8 dark:bg-slate-950/30">
      <div className="mx-auto max-w-2xl">
        <header className="mb-6">
          <div className="mb-2 inline-flex rounded-full bg-amber-100 px-3 py-1 text-xs font-semibold text-amber-800">
            PAPER • ไม่ส่งคำสั่งไปโบรกเกอร์
          </div>
          <h1 className="text-3xl font-bold">ลองเทรด XAUUSD</h1>
          <p className="text-muted-foreground">
            ตรวจรายละเอียดและราคาก่อนยืนยันทุกครั้ง
          </p>
        </header>
        <Card>
          <CardHeader>
            <CardTitle>
              {proposal ? "ยืนยันรายการทดลอง" : "เตรียมรายการ"}
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-5">
            {proposal ? (
              <>
                <div className="bg-muted/40 rounded-xl border p-5">
                  <dl className="grid grid-cols-2 gap-4 text-sm">
                    <dt className="text-muted-foreground">ฝั่ง</dt>
                    <dd className="text-right font-bold">
                      {proposal.request.side.toUpperCase()}
                    </dd>
                    <dt className="text-muted-foreground">ขนาด</dt>
                    <dd className="text-right">
                      {proposal.request.volume_lots} lot
                    </dd>
                    <dt className="text-muted-foreground">ราคาที่ใช้ได้</dt>
                    <dd className="text-right text-xl font-bold">
                      ${proposal.open_price.toFixed(2)}
                    </dd>
                    <dt className="text-muted-foreground">ความเสี่ยง</dt>
                    <dd className="text-right text-red-600">
                      ขาดทุนได้ตามการเคลื่อนไหวของราคา
                    </dd>
                  </dl>
                </div>
                <p className="text-sm text-amber-700">
                  ข้อเสนอนี้มีอายุ 30 วินาที การกด “ยืนยัน”
                  จะเปลี่ยนเฉพาะบัญชีจำลอง
                </p>
                <div className="flex gap-3">
                  <Button
                    variant="outline"
                    className="flex-1"
                    onClick={() => setProposal(undefined)}
                  >
                    กลับไปแก้ไข
                  </Button>
                  <Button
                    className="flex-1"
                    disabled={confirm.isPending}
                    onClick={() => confirm.mutate()}
                  >
                    {confirm.isPending ? "กำลังยืนยัน…" : "ยืนยันเปิด Position"}
                  </Button>
                </div>
              </>
            ) : (
              <>
                <div className="grid grid-cols-2 gap-3">
                  <Button
                    type="button"
                    variant={side === "buy" ? "default" : "outline"}
                    className={
                      side === "buy"
                        ? "bg-emerald-600 hover:bg-emerald-700"
                        : ""
                    }
                    onClick={() => setSide("buy")}
                  >
                    BUY
                  </Button>
                  <Button
                    type="button"
                    variant={side === "sell" ? "destructive" : "outline"}
                    onClick={() => setSide("sell")}
                  >
                    SELL
                  </Button>
                </div>
                <div>
                  <Label htmlFor="volume">ขนาด (lot)</Label>
                  <Input
                    id="volume"
                    inputMode="decimal"
                    value={volume}
                    onChange={(e) => setVolume(e.target.value)}
                  />
                </div>
                <div className="grid grid-cols-2 gap-3">
                  <div>
                    <Label htmlFor="sl">Stop Loss</Label>
                    <Input
                      id="sl"
                      inputMode="decimal"
                      placeholder="ไม่บังคับ"
                      value={sl}
                      onChange={(e) => setSl(e.target.value)}
                    />
                  </div>
                  <div>
                    <Label htmlFor="tp">Take Profit</Label>
                    <Input
                      id="tp"
                      inputMode="decimal"
                      placeholder="ไม่บังคับ"
                      value={tp}
                      onChange={(e) => setTp(e.target.value)}
                    />
                  </div>
                </div>
                <div className="rounded-lg bg-blue-50 p-3 text-sm text-blue-800">
                  ระบบไม่ใช้ leverage และตรวจวงเงิน/ความเสี่ยงที่ Backend
                </div>
                <Button
                  className="w-full"
                  disabled={prepare.isPending}
                  onClick={() => prepare.mutate()}
                >
                  {prepare.isPending ? "กำลังดึงราคา…" : "ดูราคาก่อนยืนยัน"}
                </Button>
              </>
            )}
          </CardContent>
        </Card>
        <Button variant="link" asChild className="mt-3">
          <Link to="/portfolio">← กลับพอร์ต</Link>
        </Button>
      </div>
    </div>
  )
}
