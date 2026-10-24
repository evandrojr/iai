package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var auditCmd = &cobra.Command{
	Use:   "audit <pacote>",
	Short: "Analisa um pacote sem instalar",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		pkg := args[0]
		fmt.Printf("Auditing %s...\n\n", pkg)

		res, err := auditPackage(pkg)
		if err != nil {
			return err
		}

		fmt.Println("\n─── Resumo ───")
		fmt.Printf("Pacote: %s\n", res.Package)
		if res.Official {
			fmt.Printf("Origem: repo oficial (%s)\n", res.Repo)
		} else {
			fmt.Println("Origem: AUR")
		}
		fmt.Printf("Score de ameaça: %d\n", res.Score)
		if res.LLMVerdict != "" {
			fmt.Printf("LLM: %s\n", res.LLMVerdict)
			for _, r := range res.LLMReasons {
				fmt.Printf("  - %s\n", r)
			}
		}
		if res.Safe {
			fmt.Printf("✓ SEGURO: %s\n", res.Reason)
		} else {
			fmt.Printf("✗ NÃO SEGURO (sem confirmação): %s\n", res.Reason)
			if len(res.Threats) > 0 {
				fmt.Println("\nAmeaças:")
				for _, t := range res.Threats {
					fmt.Printf("  [%s] %s: %s\n", strings.ToUpper(t.Severity), t.Rule, t.Detail)
				}
			}
			if len(res.LLMReasons) > 0 {
				fmt.Println("\nLLM:")
				for _, r := range res.LLMReasons {
					fmt.Printf("  - %s\n", r)
				}
			}
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(auditCmd)
}
