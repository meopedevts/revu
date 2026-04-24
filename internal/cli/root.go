package cli

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "revu",
	Short: "Vigia silencioso de PRs de review no GitHub",
	Long: `revu — system tray notifier pra Pull Requests que te marcam
como reviewer no GitHub. Notificações desktop + histórico persistente
acessível via tray.`,
}

func init() {
	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(doctorCmd)
}

func Execute() error {
	return rootCmd.Execute()
}
