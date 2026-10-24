package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/evandrojr/iai/internal/aur"
	"github.com/evandrojr/iai/internal/install"
	"github.com/evandrojr/iai/internal/llm"
	"github.com/evandrojr/iai/internal/notify"
	"github.com/evandrojr/iai/internal/scan"
	"github.com/evandrojr/iai/internal/search"
	"github.com/spf13/cobra"
)

type AuditResult struct {
	Package    string
	Official   bool
	Repo       string
	Threats    []scan.Threat
	Score      int
	LLMVerdict string
	LLMReasons []string
	Safe       bool
	Reason     string
}

var installCmd = &cobra.Command{
	Use:   "install [termo]",
	Short: "Busca e instala um pacote com verificação de ameaças",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		query := strings.Join(args, " ")
		safe, _ := cmd.Flags().GetBool("safe")

		// 1. Busca local (pacman) + AUR RPC — instantâneo, sem LLM
		fmt.Println("→ Buscando pacotes...")
		results := search.Search(query, 8)
		if len(results) == 0 {
			return fmt.Errorf("nenhum pacote encontrado para %q", query)
		}

		fmt.Println("\nResultados:")
		for i, r := range results {
			repo := r.Repo
			if r.Installed {
				repo += " [já instalado]"
			}
			fmt.Printf("  [%d] %s (%s) — %s\n      %s | %s\n",
				i+1, r.Package, repo, r.Version, shortDesc(r.Description), r.URL)
		}

		var sel search.Result
		if safe && len(results) == 1 {
			sel = results[0]
			fmt.Printf("\n→ Usando única sugestão %s\n", sel.Package)
		} else {
			fmt.Print("\nEscolha o número para instalar (0 = cancelar): ")
			var choice int
			fmt.Scanln(&choice)
			if choice < 1 || choice > len(results) {
				fmt.Println("Cancelado.")
				return nil
			}
			sel = results[choice-1]
		}

		if sel.Installed {
			fmt.Printf("%s já está instalado.\n", sel.Package)
			return nil
		}

		// 2. Auditoria (só pra AUR — oficial já é PGP assinado)
		fmt.Printf("\n→ Verificando %s...\n", sel.Package)
		res, err := auditPackage(sel.Package)
		if err != nil {
			notify.Send("iai: pacote NÃO instalado",
				fmt.Sprintf("%s: %s", sel.Package, err.Error()))
			return err
		}

		if !res.Safe {
			var parts []string
			parts = append(parts, res.Reason)
			if len(res.Threats) > 0 {
				parts = append(parts, "Ameaças determinísticas:")
				for _, t := range res.Threats {
					parts = append(parts, fmt.Sprintf("  [%s] %s: %s", strings.ToUpper(t.Severity), t.Rule, t.Detail))
				}
			}
			if len(res.LLMReasons) > 0 {
				parts = append(parts, "Motivos LLM:")
				for _, r := range res.LLMReasons {
					parts = append(parts, "  - "+r)
				}
			}
			msg := fmt.Sprintf("%s não é seguro sem dúvidas:\n%s", sel.Package, strings.Join(parts, "\n"))
			notify.Send("iai: pacote NÃO instalado", strings.ReplaceAll(msg, "\n", " | "))
			return fmt.Errorf("%s", msg)
		}

		fmt.Println("\n→ Todas as verificações passaram, instalando...")
		if res.Official {
			return install.InstallOfficial(sel.Package)
		}
		return install.InstallAUR(sel.Package)
	},
}

func shortDesc(s string) string {
	if len(s) > 80 {
		return s[:77] + "..."
	}
	return s
}

func auditPackage(pkg string) (*AuditResult, error) {
	res := &AuditResult{Package: pkg}

	// 1. Repo oficial via pacman
	if repo, ok := install.OfficialRepo(pkg); ok {
		res.Official = true
		res.Repo = repo
		res.Safe = true
		res.Reason = "repo oficial (PGP assinado)"
		fmt.Printf("  ✓ Repo oficial: %s (PGP assinado)\n", repo)
		return res, nil
	}
	fmt.Println("  ! Não está em repo oficial — verificando AUR...")

	// 2. AUR
	info, err := aur.Check(pkg)
	if err != nil {
		return nil, err
	}
	if info == nil {
		return nil, fmt.Errorf("pacote %s não encontrado nem no repo oficial nem no AUR", pkg)
	}
	fmt.Printf("  AUR: v%s | maintainer: %s | votos: %d | popularidade: %.2f\n",
		info.Version, info.Maintainer, info.NumVotes, info.Popularity)
	if info.OutOfDateBool {
		fmt.Println("  ⚠  Pacote está out-of-date")
	}

	// 3. PKGBUILD
	pkgbuild, err := aur.FetchPKGBUILD(pkg)
	if err != nil {
		return nil, err
	}

	fmt.Println("\n  Análise determinística:")
	det := scan.Deterministic(pkgbuild)
	res.Threats = det.Threats
	res.Score = det.Score
	if len(det.Threats) == 0 {
		fmt.Println("    ✓ Nenhuma ameaça detectada pelas regras")
	} else {
		for _, t := range det.Threats {
			fmt.Printf("    [%s] %s: %s\n", strings.ToUpper(t.Severity), t.Rule, t.Detail)
		}
	}

	// 4. LLM
	configPath, _ := rootCmd.PersistentFlags().GetString("config")
	cfg, err := llm.LoadConfig(configPath)
	llmOK := false
	if err == nil {
		client := llm.NewClient(cfg)
		fmt.Println("\n  Análise via LLM...")
		raw, err := client.Chat(scan.ScanPrompt(), scan.FormatForLLM(pkgbuild))
		if err != nil {
			fmt.Printf("    ! LLM falhou: %v\n", err)
		} else {
			var v scan.LLMVerdict
			if err := json.Unmarshal([]byte(extractJSON(raw)), &v); err == nil {
				llmOK = true
				res.LLMVerdict = v.Verdict
				res.LLMReasons = v.Reasons
				fmt.Printf("    Veredito: %s\n", strings.ToUpper(v.Verdict))
				for _, r := range v.Reasons {
					fmt.Printf("      - %s\n", r)
				}
				if v.Verdict == "malicious" {
					return res, fmt.Errorf("LLM classificou como MALICIOSO — abortando")
				}
			} else {
				fmt.Println("    ! Resposta do LLM não é JSON válido")
			}
		}
	}

	if det.Score >= 40 {
		res.Safe = false
		res.Reason = fmt.Sprintf("score de ameaça alto (%d)", det.Score)
		return res, fmt.Errorf("score de ameaça alto (%d) — abortando", det.Score)
	}

	if llmOK && res.LLMVerdict == "safe" {
		hasCritical := false
		for _, t := range det.Threats {
			if t.Severity == "critical" {
				hasCritical = true
				break
			}
		}
		if !hasCritical {
			res.Safe = true
			res.Reason = fmt.Sprintf("AUR: LLM confirmou seguro (score determinístico %d)", det.Score)
			return res, nil
		}
	}

	if det.Score == 0 && len(det.Threats) == 0 && llmOK && res.LLMVerdict == "safe" {
		res.Safe = true
		res.Reason = "AUR: sem ameaças determinísticas, LLM vereditou safe"
		return res, nil
	}

	res.Safe = false
	switch {
	case !llmOK:
		res.Reason = "LLM indisponível — não é possível confirmar segurança sem dúvidas"
	case res.LLMVerdict == "suspicious":
		res.Reason = "LLM vereditou SUSPICIOUS"
	case det.Score > 0:
		res.Reason = fmt.Sprintf("score de ameaça %d (não zero)", det.Score)
	default:
		res.Reason = "não passou em todas as verificações"
	}
	return res, nil
}

func extractJSON(s string) string {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "```") {
		s = strings.TrimPrefix(s, "```json")
		s = strings.TrimPrefix(s, "```")
		if i := strings.LastIndex(s, "```"); i >= 0 {
			s = s[:i]
		}
	}
	if i := strings.Index(s, "{"); i >= 0 {
		if j := strings.LastIndex(s, "}"); j > i {
			s = s[i : j+1]
		}
	}
	return strings.TrimSpace(s)
}

func init() {
	installCmd.Flags().BoolP("safe", "s", false, "com 1 resultado, pula a escolha e segue direto pra auditoria")
	rootCmd.AddCommand(installCmd)
}
