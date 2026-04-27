import { createFileRoute, useRouter } from "@tanstack/react-router"

import { PRDetailsView } from "@/components/pr-details-view"

export const Route = createFileRoute("/pr/$prId")({
  component: PRDetails,
})

function PRDetails() {
  const { prId } = Route.useParams()
  const router = useRouter()
  return <PRDetailsView prID={prId} onBack={() => router.history.back()} />
}
