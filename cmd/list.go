package cmd

import (
	"fmt"
	"strings"

	"github.com/evandrojr/iai/internal/search"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list [termo]",
	Short: "Busca pacotes online (pacman + AUR) como yay -Ss",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		query := strings.Join(args, " ")
		fmt.Printf("→ Buscando %q...\n\n", query)
		results := search.Search(query, 20)
		if len(results) == 0 {
			return fmt.Errorf("nenhum pacote encontrado para %q", query)
		}
		for _, r := range results {
			repo := r.Repo
			if r.Installed {
				repo += " [já instalado]"
			}
			fmt.Printf("%s/%s %s\n    %s\n", repo, r.Package, r.Version, r.Description)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
}
