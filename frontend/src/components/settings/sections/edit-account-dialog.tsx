import { Eye, Loader2 } from "lucide-react"
import { useEffect, useState } from "react"
import { toast } from "sonner"

import { Button } from "@/components/ui/button"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { useUpdateProfile } from "@/hooks/mutations/use-update-profile"
import type { Profile } from "@/lib/types"

interface EditAccountDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  profile: Profile | null
}

export function EditAccountDialog({
  open,
  onOpenChange,
  profile,
}: EditAccountDialogProps) {
  const [name, setName] = useState("")
  const [token, setToken] = useState("")
  const [revealing, setRevealing] = useState(false)
  const update = useUpdateProfile()
  const submitting = update.isPending

  useEffect(() => {
    if (profile) {
      setName(profile.name)
      setToken("")
      setRevealing(false)
    }
  }, [profile])

  async function onSave() {
    if (!profile) return
    if (!name.trim()) {
      toast.error("Informe um nome")
      return
    }
    const patch: { name?: string; token?: string } = {}
    if (name.trim() !== profile.name) patch.name = name.trim()
    if (token) patch.token = token
    if (Object.keys(patch).length === 0) {
      toast.info("Nada mudou")
      onOpenChange(false)
      return
    }
    try {
      await update.mutateAsync({ id: profile.id, patch })
      toast.success("Conta atualizada")
      onOpenChange(false)
    } catch (err: unknown) {
      toast.error(
        err instanceof Error ? err.message : "Falha ao atualizar conta"
      )
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Editar conta</DialogTitle>
          <DialogDescription>
            Renomeie ou rotacione o token. Deixe o token vazio para manter o
            atual.
          </DialogDescription>
        </DialogHeader>

        <div className="flex flex-col gap-3">
          <div className="flex flex-col gap-1">
            <Label htmlFor="edit-name">Nome</Label>
            <Input
              id="edit-name"
              value={name}
              onChange={(e) => setName(e.target.value)}
              autoComplete="off"
            />
          </div>

          {profile?.authMethod === "keyring" ? (
            <div className="flex flex-col gap-1">
              <Label htmlFor="edit-token">Novo token (opcional)</Label>
              <div className="flex items-center gap-2">
                <Input
                  id="edit-token"
                  type={revealing ? "text" : "password"}
                  value={token}
                  onChange={(e) => setToken(e.target.value)}
                  placeholder="deixe vazio para manter"
                  autoComplete="off"
                  spellCheck={false}
                  data-form-type="other"
                />
                <Button
                  type="button"
                  variant="ghost"
                  size="sm"
                  aria-label="Revelar token"
                  onMouseDown={() => setRevealing(true)}
                  onMouseUp={() => setRevealing(false)}
                  onMouseLeave={() => setRevealing(false)}
                  onTouchStart={() => setRevealing(true)}
                  onTouchEnd={() => setRevealing(false)}
                >
                  <Eye className="size-3.5" />
                </Button>
              </div>
            </div>
          ) : null}
        </div>

        <DialogFooter>
          <Button
            type="button"
            variant="outline"
            size="sm"
            onClick={() => onOpenChange(false)}
            disabled={submitting}
          >
            Cancelar
          </Button>
          <Button
            type="button"
            size="sm"
            onClick={() => void onSave()}
            disabled={submitting}
          >
            {submitting ? <Loader2 className="animate-spin" /> : null}
            Salvar
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
