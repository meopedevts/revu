import { MoreHorizontal, Plus } from "lucide-react"
import { useState } from "react"
import { toast } from "sonner"

import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import { useDeleteProfile } from "@/hooks/mutations/use-delete-profile"
import { useSetActiveProfile } from "@/hooks/mutations/use-set-active-profile"
import { useProfiles } from "@/hooks/use-profiles"
import type { Profile } from "@/lib/types"

import { AddAccountDialog } from "./add-account-dialog"
import { EditAccountDialog } from "./edit-account-dialog"

export function AccountsSection() {
  const profilesQ = useProfiles()
  const profiles = profilesQ.data ?? []
  const loading = profilesQ.isLoading
  const setActive = useSetActiveProfile()
  const remove = useDeleteProfile()

  const [addOpen, setAddOpen] = useState(false)
  const [editing, setEditing] = useState<Profile | null>(null)
  const [confirmDelete, setConfirmDelete] = useState<Profile | null>(null)

  if (profilesQ.error) {
    toast.error(
      profilesQ.error instanceof Error
        ? profilesQ.error.message
        : "Falha ao listar contas"
    )
  }

  async function onMakeActive(id: string) {
    try {
      await setActive.mutateAsync(id)
    } catch (err: unknown) {
      toast.error(err instanceof Error ? err.message : "Falha ao trocar conta")
    }
  }

  async function onConfirmDelete() {
    if (!confirmDelete) return
    try {
      await remove.mutateAsync(confirmDelete.id)
      toast.success("Conta removida")
      setConfirmDelete(null)
    } catch (err: unknown) {
      toast.error(err instanceof Error ? err.message : "Falha ao remover conta")
    }
  }

  return (
    <div className="flex flex-col gap-4">
      <header className="flex items-center justify-between gap-2">
        <div className="flex flex-col gap-0.5">
          <h2 className="text-sm font-semibold">Contas</h2>
          <p className="text-xs text-muted-foreground">
            Gerencie as credenciais GitHub usadas pelo revu.
          </p>
        </div>
        <Button type="button" size="sm" onClick={() => setAddOpen(true)}>
          <Plus />
          Adicionar conta
        </Button>
      </header>

      {loading ? (
        <p className="text-xs text-muted-foreground">Carregando…</p>
      ) : profiles.length === 0 ? (
        <p className="text-xs text-muted-foreground">Nenhuma conta ainda.</p>
      ) : (
        <div className="flex flex-col gap-2">
          {profiles.map((p) => (
            <Card key={p.id} size="sm">
              <CardHeader className="flex items-start gap-2">
                <div className="flex flex-1 flex-col gap-0.5">
                  <CardTitle className="flex items-center gap-2">
                    {p.name}
                    {p.is_active ? (
                      <Badge variant="default">Ativo</Badge>
                    ) : null}
                  </CardTitle>
                  <CardDescription>
                    {p.auth_method === "keyring"
                      ? "Token (keyring)"
                      : "gh auth login"}
                    {p.github_username ? ` · @${p.github_username}` : ""}
                  </CardDescription>
                </div>
              </CardHeader>
              <CardContent className="flex items-center justify-end gap-2">
                {!p.is_active ? (
                  <Button
                    type="button"
                    variant="outline"
                    size="sm"
                    onClick={() => void onMakeActive(p.id)}
                    disabled={setActive.isPending}
                  >
                    Tornar ativa
                  </Button>
                ) : null}
                <Button
                  type="button"
                  variant="ghost"
                  size="sm"
                  onClick={() => setEditing(p)}
                  aria-label={`Editar ${p.name}`}
                >
                  Editar
                </Button>
                <Button
                  type="button"
                  variant="ghost"
                  size="sm"
                  onClick={() => setConfirmDelete(p)}
                  aria-label={`Remover ${p.name}`}
                  disabled={p.is_active}
                >
                  <MoreHorizontal />
                </Button>
              </CardContent>
            </Card>
          ))}
        </div>
      )}

      <AddAccountDialog open={addOpen} onOpenChange={setAddOpen} />
      <EditAccountDialog
        open={editing !== null}
        onOpenChange={(v) => {
          if (!v) setEditing(null)
        }}
        profile={editing}
      />

      <AlertDialog
        open={confirmDelete !== null}
        onOpenChange={(v) => {
          if (!v) setConfirmDelete(null)
        }}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Remover conta?</AlertDialogTitle>
            <AlertDialogDescription>
              Remove o profile <strong>{confirmDelete?.name}</strong> e, se usar
              keyring, apaga o token do Secret Service. Essa ação não pode ser
              desfeita.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel onClick={() => setConfirmDelete(null)}>
              Cancelar
            </AlertDialogCancel>
            <AlertDialogAction onClick={() => void onConfirmDelete()}>
              Remover
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  )
}
