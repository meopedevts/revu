import { EmptyState } from "@/components/empty-state"
import { PRCard } from "@/components/pr-card"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import type { PRRecord } from "@/lib/types"

interface PRListTabsProps {
  pending: PRRecord[]
  history: PRRecord[]
  onOpenPR: (prID: string) => void
}

export function PRListTabs({ pending, history, onOpenPR }: PRListTabsProps) {
  return (
    <Tabs
      defaultValue="pending"
      className="flex flex-1 flex-col overflow-hidden"
    >
      <TabsList>
        <TabsTrigger value="pending">
          Pendentes
          {pending.length > 0 && (
            <span className="ml-1.5 rounded-full bg-primary/15 px-1.5 text-[10px] font-medium text-primary">
              {pending.length}
            </span>
          )}
        </TabsTrigger>
        <TabsTrigger value="history">Histórico</TabsTrigger>
      </TabsList>

      <TabsContent value="pending" className="flex-1 overflow-y-auto">
        {pending.length === 0 ? (
          <EmptyState variant="pending" />
        ) : (
          <div className="flex flex-col gap-2">
            {pending.map((pr) => (
              <PRCard key={pr.id} pr={pr} onOpen={onOpenPR} />
            ))}
          </div>
        )}
      </TabsContent>

      <TabsContent value="history" className="flex-1 overflow-y-auto">
        {history.length === 0 ? (
          <EmptyState variant="history" />
        ) : (
          <div className="flex flex-col gap-2">
            {history.map((pr) => (
              <PRCard key={pr.id} pr={pr} onOpen={onOpenPR} />
            ))}
          </div>
        )}
      </TabsContent>
    </Tabs>
  )
}
