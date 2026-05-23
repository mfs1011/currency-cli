package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/mfsv/cx/internal/api"
	"github.com/mfsv/cx/internal/cache"
	"github.com/mfsv/cx/internal/config"
	"github.com/mfsv/cx/internal/display"
	"github.com/spf13/cobra"
)

var currenciesCmd = &cobra.Command{
	Use:     "currencies [QIDIRUV]",
	Aliases: []string{"cur", "list"},
	Short:   "Barcha mavjud valyutalar ro'yxati",
	Long: `CBU da mavjud barcha valyutalar ro'yxatini ko'rsatadi.

Examples:
  cx currencies           # barchasi
  cx currencies usd       # kod bo'yicha qidirish
  cx currencies dollar    # nom bo'yicha qidirish`,
	Args: cobra.MaximumNArgs(1),
	RunE: runCurrencies,
}

func init() {
	rootCmd.AddCommand(currenciesCmd)
}

func runCurrencies(cmd *cobra.Command, args []string) error {
	c, _ := cache.New(config.Dir())

	cbuKey := "cbu:all"
	var rates []api.CBURate
	ok := c != nil && c.Get(cbuKey, 24*time.Hour, &rates)
	if !ok {
		fetched, err := api.FetchCBU()
		if err != nil {
			if c != nil {
				if staleOK, _ := c.GetStale(cbuKey, &rates); !staleOK {
					return fmt.Errorf("CBU dan ma'lumot olishda xato: %w", err)
				}
			} else {
				return fmt.Errorf("CBU dan ma'lumot olishda xato: %w", err)
			}
		} else {
			rates = fetched
			if c != nil {
				_ = c.Set(cbuKey, rates)
			}
		}
	}

	query := ""
	if len(args) > 0 {
		query = strings.ToLower(args[0])
	}

	var filtered []api.CBURate
	for _, r := range rates {
		if query == "" {
			filtered = append(filtered, r)
			continue
		}
		if strings.Contains(strings.ToLower(r.Ccy), query) ||
			strings.Contains(strings.ToLower(r.CcyNmUZ), query) ||
			strings.Contains(strings.ToLower(r.CcyNmRU), query) ||
			strings.Contains(strings.ToLower(r.CcyNmEN), query) {
			filtered = append(filtered, r)
		}
	}

	if len(filtered) == 0 {
		fmt.Fprintf(os.Stderr, "topilmadi: %s\n", query)
		return nil
	}

	display.PrintCurrenciesTable(filtered)
	return nil
}
