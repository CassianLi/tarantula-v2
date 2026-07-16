package utils

import (
	"encoding/json"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

var isoCurrencyRE = regexp.MustCompile(`\b([A-Z]{3})\b`)

// ExtractPriceCurrencyFromHTML reads currency from meta tags or JSON-LD on the page.
func ExtractPriceCurrencyFromHTML(doc *goquery.Document) string {
	if content, ok := doc.Find(`meta[property="product:price:currency"]`).Attr("content"); ok {
		if c := normalizeCurrencyCode(content); c != "" {
			return c
		}
	}

	var currency string
	doc.Find(`script[type="application/ld+json"]`).EachWithBreak(func(_ int, s *goquery.Selection) bool {
		currency = priceCurrencyFromJSONLD(strings.TrimSpace(s.Text()))
		return currency == ""
	})
	return currency
}

// CurrencyFromPriceText infers ISO 4217 code from visible price text.
func CurrencyFromPriceText(text string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}
	if m := isoCurrencyRE.FindStringSubmatch(text); len(m) > 1 {
		return m[1]
	}
	switch {
	case strings.Contains(text, "€"):
		return "EUR"
	case strings.Contains(text, "zł"), strings.Contains(strings.ToUpper(text), "ZL"):
		return "PLN"
	case strings.Contains(text, "£"):
		return "GBP"
	case strings.Contains(text, "$"):
		return "USD"
	default:
		return ""
	}
}

func priceCurrencyFromJSONLD(jsonText string) string {
	if jsonText == "" {
		return ""
	}
	var raw any
	if err := json.Unmarshal([]byte(jsonText), &raw); err != nil {
		return ""
	}
	return findPriceCurrency(raw)
}

func findPriceCurrency(v any) string {
	switch node := v.(type) {
	case []any:
		for _, item := range node {
			if c := findPriceCurrency(item); c != "" {
				return c
			}
		}
	case map[string]any:
		if offers, ok := node["offers"]; ok {
			if c := currencyFromOffers(offers); c != "" {
				return c
			}
		}
		if graph, ok := node["@graph"]; ok {
			if c := findPriceCurrency(graph); c != "" {
				return c
			}
		}
		if pc, ok := node["priceCurrency"].(string); ok {
			return normalizeCurrencyCode(pc)
		}
	}
	return ""
}

func currencyFromOffers(offers any) string {
	switch o := offers.(type) {
	case map[string]any:
		if pc, ok := o["priceCurrency"].(string); ok {
			return normalizeCurrencyCode(pc)
		}
	case []any:
		for _, item := range o {
			if c := currencyFromOffers(item); c != "" {
				return c
			}
		}
	}
	return ""
}

func normalizeCurrencyCode(code string) string {
	code = strings.ToUpper(strings.TrimSpace(code))
	if len(code) == 3 {
		return code
	}
	return ""
}
