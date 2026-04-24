package cli

import (
	"fmt"
	"io"

	"github.com/meopedevts/revu/internal/github"
	"github.com/spf13/cobra"
)

var doctorExitCode bool

var doctorCmd = &cobra.Command{
	Use:           "doctor",
	Short:         "Valida dependências de runtime (gh auth, libs, D-Bus)",
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		client := github.NewClient(github.DefaultExecutor())
		db, err := dbPath()
		if err != nil {
			return err
		}
		results := runAllChecks(ctx, client, db)
		failed := printResults(cmd.OutOrStdout(), results)
		if failed > 0 && doctorExitCode {
			return fmt.Errorf("%d verificação(ões) falharam", failed)
		}
		return nil
	},
}

func init() {
	doctorCmd.Flags().BoolVar(&doctorExitCode, "exit-code", false,
		"sai com código 1 se alguma verificação falhar (útil em systemd/CI)")
}

func printResults(w io.Writer, results []checkResult) int {
	failed := 0
	for _, r := range results {
		if r.OK {
			if r.Detail == "" {
				fmt.Fprintf(w, "✓ %s\n", r.Name)
			} else {
				fmt.Fprintf(w, "✓ %s: %s\n", r.Name, r.Detail)
			}
			continue
		}
		failed++
		if r.Detail == "" {
			fmt.Fprintf(w, "✗ %s\n", r.Name)
		} else {
			fmt.Fprintf(w, "✗ %s: %s\n", r.Name, r.Detail)
		}
	}
	return failed
}
