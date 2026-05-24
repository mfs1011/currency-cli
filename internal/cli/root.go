package cli

import (
	"fmt"
	"os"
	"runtime/debug"

	"github.com/mfs1011/currency-cli/internal/config"
	"github.com/spf13/cobra"
)

var version = func() string {
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
		return info.Main.Version
	}
	return "dev"
}()

var rootCmd = &cobra.Command{
	Use:   "cx",
	Short: "Currency rates and converter for Uzbekistan",
	Long: `cx — CLI currency converter

Data sources:
  onmap.uz — commercial bank rates (updated 4x daily)
  cbu.uz   — official Central Bank rate (daily)

Examples:
  cx rates USD
  cx convert 100 USD UZS
  cx best 500 EUR UZS`,
	Version: version,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(func() {
		if err := config.Ensure(); err != nil {
			fmt.Fprintf(os.Stderr, "warning: could not create config dir: %v\n", err)
		}
	})

	rootCmd.AddCommand(ratesCmd)
	rootCmd.AddCommand(convertCmd)
	rootCmd.AddCommand(bestCmd)
}
