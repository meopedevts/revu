import { ArrowLeft, Loader2, RotateCcw } from 'lucide-react'
import { useCallback, useState } from 'react'

import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog'
import { Button } from '@/components/ui/button'
import { Form } from '@/components/ui/form'

import { AccountsSection } from './sections/accounts-section'
import { AppearanceSection } from './sections/appearance-section'
import { HistorySection } from './sections/history-section'
import { NotificationsSection } from './sections/notifications-section'
import { SyncSection } from './sections/sync-section'
import { SettingsSidebar, type SettingsSection } from './settings-sidebar'
import { useActiveProfile } from './use-active-profile'
import { useSettingsForm } from './use-settings-form'

interface SettingsLayoutProps {
  onBack: () => void
  initialSection?: SettingsSection
}

export function SettingsLayout({
  onBack,
  initialSection = 'sync',
}: SettingsLayoutProps) {
  const { form, loading, saving, submit, discard, restoreDefaults } =
    useSettingsForm()
  const { profile: activeProfile } = useActiveProfile()

  const [section, setSection] = useState<SettingsSection>(initialSection)
  const [pendingSection, setPendingSection] = useState<SettingsSection | null>(
    null
  )

  const onSelectSection = useCallback(
    (next: SettingsSection) => {
      if (next === section) return
      // Contas is CRUD (no form) — no unsaved guard needed in or out.
      const leavingForm = section !== 'accounts' && form.formState.isDirty
      if (leavingForm) {
        setPendingSection(next)
        return
      }
      setSection(next)
    },
    [form.formState.isDirty, section]
  )

  const confirmDiscardAndSwitch = useCallback(async () => {
    if (pendingSection) {
      await discard()
      setSection(pendingSection)
      setPendingSection(null)
    }
  }, [discard, pendingSection])

  const cancelSwitch = useCallback(() => setPendingSection(null), [])

  const canSave =
    section !== 'accounts' &&
    form.formState.isDirty &&
    form.formState.isValid &&
    !saving

  if (loading) {
    return (
      <div className="flex h-screen items-center justify-center bg-background text-muted-foreground">
        <Loader2 className="size-4 animate-spin" />
      </div>
    )
  }

  const showFooter = section !== 'accounts'

  return (
    <div className="flex h-screen flex-col bg-background text-foreground">
      <header className="flex items-center gap-2 border-b px-3 py-2">
        <Button size="sm" variant="ghost" onClick={onBack} aria-label="Voltar">
          <ArrowLeft />
        </Button>
        <div className="font-heading text-base font-medium">Configurações</div>
      </header>

      <Form {...form}>
        <form
          onSubmit={(e) => {
            e.preventDefault()
            void submit()
          }}
          className="flex flex-1 flex-col overflow-hidden"
        >
          <div className="flex flex-1 overflow-hidden">
            <SettingsSidebar
              active={section}
              onSelect={onSelectSection}
              activeProfile={activeProfile}
            />
            <div className="flex-1 overflow-y-auto px-4 py-4">
              {section === 'accounts' ? (
                <AccountsSection />
              ) : section === 'sync' ? (
                <SyncSection form={form} />
              ) : section === 'notifications' ? (
                <NotificationsSection form={form} />
              ) : section === 'history' ? (
                <HistorySection form={form} />
              ) : (
                <AppearanceSection form={form} />
              )}
            </div>
          </div>

          {showFooter ? (
            <footer className="flex items-center justify-between gap-2 border-t bg-muted/40 px-3 py-2">
              <Button
                type="button"
                variant="ghost"
                size="sm"
                onClick={restoreDefaults}
              >
                <RotateCcw />
                Restaurar padrões
              </Button>
              <div className="flex items-center gap-2">
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  onClick={() => void discard()}
                  disabled={!form.formState.isDirty || saving}
                >
                  Descartar
                </Button>
                <Button type="submit" size="sm" disabled={!canSave}>
                  {saving ? <Loader2 className="animate-spin" /> : null}
                  Salvar
                </Button>
              </div>
            </footer>
          ) : null}
        </form>
      </Form>

      <AlertDialog
        open={pendingSection !== null}
        onOpenChange={(open) => {
          if (!open) cancelSwitch()
        }}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Descartar alterações?</AlertDialogTitle>
            <AlertDialogDescription>
              Você tem alterações não salvas nesta seção. Ao trocar, elas serão
              perdidas.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel onClick={cancelSwitch}>
              Cancelar
            </AlertDialogCancel>
            <AlertDialogAction onClick={() => void confirmDiscardAndSwitch()}>
              Descartar e continuar
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  )
}
