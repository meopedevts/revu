import { Eye, Loader2 } from 'lucide-react'
import { useState } from 'react'
import { toast } from 'sonner'

import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { RadioGroup, RadioGroupItem } from '@/components/ui/radio-group'
import { createProfile, validateToken } from '@/src/lib/bridge'
import type { AuthMethod } from '@/src/lib/types'

interface AddAccountDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  onCreated: () => void
}

export function AddAccountDialog({
  open,
  onOpenChange,
  onCreated,
}: AddAccountDialogProps) {
  const [name, setName] = useState('')
  const [method, setMethod] = useState<AuthMethod>('keyring')
  const [token, setToken] = useState('')
  const [makeActive, setMakeActive] = useState(true)
  const [revealing, setRevealing] = useState(false)
  const [submitting, setSubmitting] = useState(false)
  const [previewedUsername, setPreviewedUsername] = useState<string | null>(
    null
  )

  function reset() {
    setName('')
    setMethod('keyring')
    setToken('')
    setMakeActive(true)
    setRevealing(false)
    setPreviewedUsername(null)
  }

  async function onValidate() {
    if (!token) {
      toast.error('Informe o token primeiro')
      return
    }
    try {
      const username = await validateToken(token)
      setPreviewedUsername(username || '(username indisponível)')
      toast.success(
        username ? `Token válido para @${username}` : 'Token válido'
      )
    } catch (err: unknown) {
      setPreviewedUsername(null)
      toast.error(
        err instanceof Error ? err.message : 'Token inválido ou sem permissão'
      )
    }
  }

  async function onSave() {
    if (!name.trim()) {
      toast.error('Informe um nome para a conta')
      return
    }
    if (method === 'keyring' && !token) {
      toast.error('Informe o token')
      return
    }
    setSubmitting(true)
    try {
      await createProfile({
        name: name.trim(),
        auth_method: method,
        token,
        make_active: makeActive,
      })
      toast.success('Conta adicionada')
      reset()
      onOpenChange(false)
      onCreated()
    } catch (err: unknown) {
      toast.error(
        err instanceof Error ? err.message : 'Falha ao adicionar conta'
      )
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <Dialog
      open={open}
      onOpenChange={(v) => {
        if (!v) reset()
        onOpenChange(v)
      }}
    >
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Adicionar conta</DialogTitle>
          <DialogDescription>
            Salve uma credencial separada por conta (trabalho, pessoal, etc).
          </DialogDescription>
        </DialogHeader>

        <div className="flex flex-col gap-3">
          <div className="flex flex-col gap-1">
            <Label htmlFor="acct-name">Nome</Label>
            <Input
              id="acct-name"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="ex.: trabalho"
              autoComplete="off"
            />
          </div>

          <div className="flex flex-col gap-1">
            <Label>Método</Label>
            <RadioGroup
              value={method}
              onValueChange={(v) => setMethod(v as AuthMethod)}
              className="flex gap-4"
            >
              <label
                htmlFor="m-keyring"
                className="flex items-center gap-2 text-xs"
              >
                <RadioGroupItem value="keyring" id="m-keyring" />
                Token (keyring)
              </label>
              <label
                htmlFor="m-ghcli"
                className="flex items-center gap-2 text-xs"
              >
                <RadioGroupItem value="gh-cli" id="m-ghcli" />
                gh auth login
              </label>
            </RadioGroup>
          </div>

          {method === 'keyring' ? (
            <div className="flex flex-col gap-1">
              <Label htmlFor="acct-token">Personal Access Token</Label>
              <div className="flex items-center gap-2">
                <Input
                  id="acct-token"
                  type={revealing ? 'text' : 'password'}
                  value={token}
                  onChange={(e) => setToken(e.target.value)}
                  placeholder="ghp_…"
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
              <p className="text-[11px] text-muted-foreground">
                Use um{' '}
                <a
                  className="underline underline-offset-3 hover:text-foreground"
                  href="https://github.com/settings/personal-access-tokens/new"
                >
                  fine-grained PAT
                </a>{' '}
                com escopos <code>Pull requests: read</code> e{' '}
                <code>Metadata: read</code>. Ao salvar, o keyring pode pedir
                autorização.
              </p>
              <div className="flex items-center gap-2">
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  onClick={() => void onValidate()}
                  disabled={!token}
                >
                  Validar
                </Button>
                {previewedUsername ? (
                  <span className="text-xs text-muted-foreground">
                    @{previewedUsername}
                  </span>
                ) : null}
              </div>
            </div>
          ) : (
            <p className="text-xs text-muted-foreground">
              Usa a sessão ambiente do <code>gh auth login</code>. Nada é
              armazenado no keyring.
            </p>
          )}

          <label className="flex items-center gap-2 text-xs">
            <input
              type="checkbox"
              checked={makeActive}
              onChange={(e) => setMakeActive(e.target.checked)}
            />
            Tornar ativa agora
          </label>
        </div>

        <DialogFooter>
          <Button
            type="button"
            variant="outline"
            size="sm"
            onClick={() => {
              reset()
              onOpenChange(false)
            }}
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
