package install

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func OfficialRepo(pkg string) (string, bool) {
	cmd := exec.Command("pacman", "-Si", pkg)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", false
	}
	// pacman -Si imprime "Repositório : extra"
	for _, line := range strings.Split(string(out), "\n") {
		if strings.HasPrefix(line, "Reposit") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				return strings.TrimSpace(parts[1]), true
			}
		}
	}
	return "", false
}

func InstallOfficial(pkg string) error {
	cmd := exec.Command("sudo", "pacman", "-S", "--noconfirm", pkg)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

func DetectAURHelper() string {
	for _, h := range []string{"paru", "yay"} {
		if p, err := exec.LookPath(h); err == nil {
			return p
		}
	}
	return ""
}

func InstallAUR(pkg string) error {
	helper := DetectAURHelper()
	if helper != "" {
		cmd := exec.Command(helper, "-S", "--noconfirm", pkg)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin
		return cmd.Run()
	}
	// fallback: clonar e makepkg
	return fmt.Errorf("nenhum helper AUR (paru/yay) encontrado — instale um ou use makepkg manualmente")
}

func InstallFontFile(path string) error {
	home, _ := os.UserHomeDir()
	dest := home + "/.local/share/fonts"
	if err := os.MkdirAll(dest, 0755); err != nil {
		return err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	base := strings.TrimSuffix(path[strings.LastIndex(path, "/")+1:], ".")
	// preserva extensão
	ext := ""
	if i := strings.LastIndex(path, "."); i >= 0 {
		ext = path[i:]
	}
	dst := dest + "/" + base + ext
	if err := os.WriteFile(dst, data, 0644); err != nil {
		return err
	}
	return RefreshFontCache()
}

func RefreshFontCache() error {
	cmd := exec.Command("fc-cache", "-f")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
