package scan

import (
	"regexp"
)

type Threat struct {
	Severity string `json:"severity"` // critical | high | medium | low
	Rule     string `json:"rule"`
	Detail   string `json:"detail"`
}

type Result struct {
	Threats []Threat `json:"threats"`
	Score   int      `json:"score"` // 0-100, maior = mais perigoso
}

func (r *Result) Add(severity, rule, detail string) {
	r.Threats = append(r.Threats, Threat{Severity: severity, Rule: rule, Detail: detail})
	switch severity {
	case "critical":
		r.Score += 40
	case "high":
		r.Score += 25
	case "medium":
		r.Score += 10
	case "low":
		r.Score += 5
	}
}

type rule struct {
	severity string
	name     string
	re       *regexp.Regexp
	detail   string
}

var rules = []rule{
	{"critical", "curl_pipe_shell", regexp.MustCompile(`curl[^|]*\|[^|]*(sh|bash)`), "curl com pipe pra shell"},
	{"critical", "wget_pipe_shell", regexp.MustCompile(`wget[^|]*\|[^|]*(sh|bash)`), "wget com pipe pra shell"},
	{"critical", "eval", regexp.MustCompile(`\beval\s+[\"']`), "eval com string (possível ofuscação)"},
	{"critical", "base64_decode", regexp.MustCompile(`base64\s+(-d|--decode)`), "decodificação base64 (possível payload oculto)"},
	{"high", "pastebin", regexp.MustCompile(`(pastebin\.com|paste\.ee|rentry\.co|ghostbin)`), "URL de pastebin"},
	{"high", "raw_gist", regexp.MustCompile(`gist\.githubusercontent\.com`), "gist raw (conteúdo não versionado)"},
	{"high", "ip_direct", regexp.MustCompile(`https?://\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}`), "download por IP direto"},
	{"high", "sudo_build", regexp.MustCompile(`(?m)^\s*sudo\s+`), "sudo fora do package() — raro e perigoso"},
	{"high", "systemd_install", regexp.MustCompile(`\.service`), "referência a systemd service"},
	{"high", "write_etc", regexp.MustCompile(`/etc/`), "escrita em /etc"},
	{"medium", "chmod_777", regexp.MustCompile(`chmod\s+777`), "chmod 777"},
	{"medium", "rm_rf", regexp.MustCompile(`rm\s+-rf\s+/[^$\{]`), "rm -rf em caminho absoluto"},
	{"medium", "suspicious_domain", regexp.MustCompile(`https?://[a-z0-9-]+\.(tk|ml|ga|cf|gq|xyz|top|buzz)(/|$)`), "TLD suspeito"},
}

func Deterministic(pkgbuild string) *Result {
	res := &Result{}
	for _, r := range rules {
		if r.re.MatchString(pkgbuild) {
			res.Add(r.severity, r.name, r.detail)
		}
	}
	return res
}

type LLMVerdict struct {
	Verdict       string   `json:"verdict"`
	Reasons       []string `json:"reasons"`
	UpstreamCheck string   `json:"upstream_check"`
}

const scanPrompt = `Você é um analista de segurança de PKGBUILDs do Arch Linux.
Analise o PKGBUILD fornecido.

RETORNE APENAS JSON VÁLIDO. NADA MAIS. NÃO use markdown, não use fences de código, não use texto antes ou depois.

Formato EXATO:
{"verdict":"safe","reasons":["motivo1","motivo2"],"upstream_check":"URL upstream confere com o oficial do projeto?"}

verdict deve ser exatamente: "safe", "suspicious" ou "malicious".

Considere malicioso se houver: download de fonte não-oficial, payload ofuscado,
escrita fora do dir de staging, backdoor, mineração, exfiltração.
Seja conservador: na dúvida, "suspicious".`

func ScanPrompt() string {
	return scanPrompt
}

func FormatForLLM(pkgbuild string) string {
	return "Analise este PKGBUILD:\n\n```bash\n" + pkgbuild + "\n```"
}
