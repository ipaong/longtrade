import { createFileRoute } from "@tanstack/react-router"

import { TradeTicket } from "@/components/paper/trade-ticket"

export const Route = createFileRoute("/try-trade")({ component: TradeTicket })
