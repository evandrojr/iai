package cmd

import (
	"fmt"
	"os"

	"github.com/evandrojr/iai/internal/install"
	"github.com/spf13/cobra"
)

var verifyCmd = &cobra.Command{
	Use:   "verify <arquivo.ttf|otf|woff2>",
	Short: "Verifica magic bytes de um arquivo de fonte",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		path := args[0]
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()

		header := make([]byte, 4)
		if _, err := f.Read(header); err != nil {
			return err
		}

		valid := false
		format := ""
		switch {
		case len(header) >= 4 && string(header[:4]) == "\x00\x01\x00\x00":
			format, valid = "TrueType (TTF)", true
		case len(header) >= 4 && string(header[:4]) == "OTTO":
			format, valid = "OpenType (CFF/OTF)", true
		case len(header) >= 4 && string(header[:4]) == "ttcf":
			format, valid = "TrueType Collection (TTC)", true
		case len(header) >= 4 && string(header[:4]) == "wOFF":
			format, valid = "WOFF", true
		case len(header) >= 4 && string(header[:4]) == "wOF2":
			format, valid = "WOFF2", true
		}

		if !valid {
			return fmt.Errorf("magic bytes inválidos: %x %x %x %x — não é uma fonte válida", header[0], header[1], header[2], header[3])
		}

		fmt.Printf("✓ %s: %s\n", path, format)

		// pergunta se quer instalar
		fmt.Print("Instalar em ~/.local/share/fonts? [y/N] ")
		var resp string
		fmt.Scanln(&resp)
		if resp == "y" || resp == "Y" {
			return install.InstallFontFile(path)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(verifyCmd)
}
