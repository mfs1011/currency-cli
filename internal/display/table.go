package display

import (
	"fmt"
	"io"
	"math"
	"os"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/fatih/color"
	"github.com/mfsv/cx/internal/api"
)

var (
	headerColor = color.New(color.Bold, color.FgCyan)
	bestColor   = color.New(color.Bold, color.FgGreen)
	warnColor   = color.New(color.FgYellow)
	dimColor    = color.New(color.FgHiBlack)
)

type RateRow struct {
	Source  string
	Buy     string
	Sell    string
	IsOnmap bool
}

// visibleWidth returns display width of s, ignoring ANSI escape sequences.
// Cyrillic and ASCII are both 1 column wide, so rune count is correct.
func visibleWidth(s string) int {
	inEsc := false
	w := 0
	for _, r := range s {
		if r == '\x1b' {
			inEsc = true
			continue
		}
		if inEsc {
			if r == 'm' {
				inEsc = false
			}
			continue
		}
		if utf8.ValidRune(r) {
			w++
		}
	}
	return w
}

// padRight pads s to width columns (ANSI-aware).
func padRight(s string, width int) string {
	pad := width - visibleWidth(s)
	if pad <= 0 {
		return s
	}
	return s + strings.Repeat(" ", pad)
}

// padLeft right-aligns s in width columns (ANSI-aware).
func padLeft(s string, width int) string {
	pad := width - visibleWidth(s)
	if pad <= 0 {
		return s
	}
	return strings.Repeat(" ", pad) + s
}

func PrintRatesTable(rows []RateRow, currency, baseCurrency string, stale bool, staleAt time.Time) {
	if stale {
		warnColor.Fprintf(os.Stderr, "offline: using cached data from %s\n", staleAt.Format("2006-01-02 15:04"))
	}

	bestBuy := bestBuyRow(rows)
	bestSell := bestSellRow(rows)

	// Pre-render cells to measure column widths
	type renderedRow struct {
		src, buy, sell string
	}
	rendered := make([]renderedRow, len(rows))
	col0w, col1w, col2w := utf8.RuneCountInString("Bank"), utf8.RuneCountInString("Buy ("+baseCurrency+")"), utf8.RuneCountInString("Sell ("+baseCurrency+")")

	for i, r := range rows {
		buyStr := formatRate(r.Buy)
		sellStr := formatRate(r.Sell)

		isBestBuy := r.Source == bestBuy && r.Buy != "" && cleanRate(r.Buy) != "0"
		isBestSell := r.Source == bestSell && r.Sell != "" && cleanRate(r.Sell) != "0"

		srcStr := r.Source
		if isBestBuy || isBestSell {
			srcStr = bestColor.Sprint(r.Source)
			if isBestBuy {
				buyStr = bestColor.Sprint(buyStr)
			}
			if isBestSell {
				sellStr = bestColor.Sprint(sellStr)
			}
		}

		rendered[i] = renderedRow{srcStr, buyStr, sellStr}

		if w := utf8.RuneCountInString(r.Source); w > col0w {
			col0w = w
		}
		if w := visibleWidth(buyStr); w > col1w {
			col1w = w
		}
		if w := visibleWidth(sellStr); w > col2w {
			col2w = w
		}
	}

	gap := 2
	totalW := col0w + gap + col1w + gap + col2w

	// Header
	h0 := headerColor.Sprint(padRight("Bank", col0w))
	h1 := headerColor.Sprint(padLeft("Buy ("+baseCurrency+")", col1w))
	h2 := headerColor.Sprint(padLeft("Sell ("+baseCurrency+")", col2w))
	fmt.Printf("%s%s%s%s%s\n", h0, strings.Repeat(" ", gap), h1, strings.Repeat(" ", gap), h2)
	fmt.Println(strings.Repeat("─", totalW))

	for _, r := range rendered {
		c0 := padRight(r.src, col0w)
		c1 := padLeft(r.buy, col1w)
		c2 := padLeft(r.sell, col2w)
		fmt.Printf("%s%s%s%s%s\n", c0, strings.Repeat(" ", gap), c1, strings.Repeat(" ", gap), c2)
	}

	fmt.Printf("\n%s %s → %s\n", dimColor.Sprint("Rates:"), currency, baseCurrency)
}

func PrintConvertTable(w io.Writer, rows []ConvertRow, amount float64, from, to string, stale bool, staleAt time.Time) {
	if stale {
		warnColor.Fprintf(os.Stderr, "offline: using cached data from %s\n", staleAt.Format("2006-01-02 15:04"))
	}

	bestBuyVal := bestConvertBuy(rows)
	bestSellVal := bestConvertSell(rows)

	type renderedRow struct {
		src, val string
	}
	rendered := make([]renderedRow, len(rows))

	header1 := fmt.Sprintf("%.2f %s → %s", amount, from, to)
	col0w := utf8.RuneCountInString("Bank")
	col1w := utf8.RuneCountInString(header1)

	for i, r := range rows {
		buyStr := formatConvertedAmount(r.BuyResult, to)
		sellStr := formatConvertedAmount(r.SellResult, to)
		line := buyStr + " / " + sellStr

		srcStr := r.Source
		if math.Abs(r.BuyResult-bestBuyVal) < 0.01 || math.Abs(r.SellResult-bestSellVal) < 0.01 {
			srcStr = bestColor.Sprint(r.Source)
			line = bestColor.Sprint(line)
		}

		rendered[i] = renderedRow{srcStr, line}

		if ww := utf8.RuneCountInString(r.Source); ww > col0w {
			col0w = ww
		}
		if ww := visibleWidth(line); ww > col1w {
			col1w = ww
		}
	}

	gap := 2
	totalW := col0w + gap + col1w

	h0 := headerColor.Sprint(padRight("Bank", col0w))
	h1 := headerColor.Sprint(padLeft(header1, col1w))
	fmt.Fprintf(w, "%s%s%s\n", h0, strings.Repeat(" ", gap), h1)
	fmt.Fprintln(w, strings.Repeat("─", totalW))

	for _, r := range rendered {
		c0 := padRight(r.src, col0w)
		c1 := padLeft(r.val, col1w)
		fmt.Fprintf(w, "%s%s%s\n", c0, strings.Repeat(" ", gap), c1)
	}
	fmt.Fprintf(w, "\n%s buy / sell\n", dimColor.Sprint("Format:"))
}

type ConvertRow struct {
	Source     string
	BuyResult  float64
	SellResult float64
}

func PrintCurrenciesTable(rates []api.CBURate) {
	col0w, col1w, col2w, col3w := 4, 1, 1, 1 // Code, UZ, RU, EN
	for _, r := range rates {
		if w := utf8.RuneCountInString(r.Ccy); w > col0w {
			col0w = w
		}
		if w := utf8.RuneCountInString(r.CcyNmUZ); w > col1w {
			col1w = w
		}
		if w := utf8.RuneCountInString(r.CcyNmRU); w > col2w {
			col2w = w
		}
		if w := utf8.RuneCountInString(r.CcyNmEN); w > col3w {
			col3w = w
		}
	}

	gap := 2
	h0 := headerColor.Sprint(padRight("Kod", col0w))
	h1 := headerColor.Sprint(padRight("O'zbekcha", col1w))
	h2 := headerColor.Sprint(padRight("Ruscha", col2w))
	h3 := headerColor.Sprint(padRight("Inglizcha", col3w))
	sp := strings.Repeat(" ", gap)
	fmt.Printf("%s%s%s%s%s%s%s\n", h0, sp, h1, sp, h2, sp, h3)
	fmt.Println(strings.Repeat("─", col0w+col1w+col2w+col3w+gap*3))

	for _, r := range rates {
		c0 := padRight(r.Ccy, col0w)
		c1 := padRight(r.CcyNmUZ, col1w)
		c2 := padRight(r.CcyNmRU, col2w)
		c3 := padRight(r.CcyNmEN, col3w)
		fmt.Printf("%s%s%s%s%s%s%s\n",
			headerColor.Sprint(c0), sp, c1, sp, dimColor.Sprint(c2), sp, dimColor.Sprint(c3))
	}
	fmt.Printf("\n%s %d ta valyuta\n", dimColor.Sprint("Manba: cbu.uz —"), len(rates))
}

func cleanRate(s string) string {
	return strings.ReplaceAll(s, " ", "")
}

func formatRate(s string) string {
	s = cleanRate(s)
	if s == "" || s == "0" {
		return dimColor.Sprint("—")
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return s
	}
	if f >= 1000 {
		return formatWithCommas(f)
	}
	return fmt.Sprintf("%.4f", f)
}

func formatWithCommas(f float64) string {
	intPart := int64(f)
	frac := f - float64(intPart)
	s := fmt.Sprintf("%d", intPart)
	result := ""
	for i, c := range reverseStr(s) {
		if i > 0 && i%3 == 0 {
			result = "," + result
		}
		result = string(c) + result
	}
	if frac > 0.0001 {
		result += fmt.Sprintf(".%02d", int(frac*100))
	}
	return result
}

func reverseStr(s string) string {
	r := []rune(s)
	for i, j := 0, len(r)-1; i < j; i, j = i+1, j-1 {
		r[i], r[j] = r[j], r[i]
	}
	return string(r)
}

func formatConvertedAmount(v float64, currency string) string {
	if v <= 0 {
		return dimColor.Sprint("—")
	}
	if v >= 1000 {
		return fmt.Sprintf("%s %s", formatWithCommas(v), currency)
	}
	return fmt.Sprintf("%.4f %s", v, currency)
}

func bestBuyRow(rows []RateRow) string {
	best := ""
	bestVal := 0.0
	for _, r := range rows {
		v, err := strconv.ParseFloat(cleanRate(r.Buy), 64)
		if err != nil || v == 0 {
			continue
		}
		if v > bestVal {
			bestVal = v
			best = r.Source
		}
	}
	return best
}

func bestSellRow(rows []RateRow) string {
	best := ""
	bestVal := math.MaxFloat64
	for _, r := range rows {
		v, err := strconv.ParseFloat(cleanRate(r.Sell), 64)
		if err != nil || v == 0 {
			continue
		}
		if v < bestVal {
			bestVal = v
			best = r.Source
		}
	}
	return best
}

func bestConvertBuy(rows []ConvertRow) float64 {
	best := 0.0
	for _, r := range rows {
		if r.BuyResult > best {
			best = r.BuyResult
		}
	}
	return best
}

func bestConvertSell(rows []ConvertRow) float64 {
	best := 0.0
	for _, r := range rows {
		if r.SellResult > best {
			best = r.SellResult
		}
	}
	return best
}
