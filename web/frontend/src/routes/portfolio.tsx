import { createFileRoute } from "@tanstack/react-router"

import { PaperDashboard } from "@/components/paper/paper-dashboard"

export const Route = createFileRoute("/portfolio")({
  component: PaperDashboard,
})
