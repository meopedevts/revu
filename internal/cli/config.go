package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Imprime path do arquivo de configuração",
	RunE: func(_ *cobra.Command, _ []string) error {
		path, err := configPath()
		if err != nil {
			return err
		}
		fmt.Println(path)
		return nil
	},
}

func configPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("resolve user config dir: %w", err)
	}
	return filepath.Join(dir, "revu", "config.json"), nil
}
