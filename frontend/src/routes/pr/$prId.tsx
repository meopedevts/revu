import { createFileRoute, useRouter } from "@tanstack/react-router"

import { PRDetailsView } from "@/components/pr-details/pr-details-view"
import { RouteErrorFallback } from "@/components/route-error-fallback"

export const Route = createFileRoute("/pr/$prId")({
  component: PRDetails,
  errorComponent: RouteErrorFallback,
})

function PRDetails() {
  const { prId } = Route.useParams()
  const router = useRouter()
  return <PRDetailsView prID={prId} onBack={() => router.history.back()} />
}
