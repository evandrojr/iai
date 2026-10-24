package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "iai",
	Short: "iai — Install by AI (instalação segura de pacotes via LLM)",
	Long: `iai (Install by AI) ajuda a instalar pacotes de forma segura usando um LLM
para descoberta, mas com verificação determinística de ameaças antes de qualquer
instalação. O LLM nunca decide segurança sozinho — ele sugere, o código verifica.`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().String("config", "", "caminho do config (default: ~/.config/iai/config.json)")
}
