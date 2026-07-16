package utils

import (
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
)

var localizedPriceRE = regexp.MustCompile(`(\d{1,3}(?:\.\d{3})*,\d{2}|\d+,\d{2}|\d+\.\d{2})`)

// ParseLocalizedPrice 从 eBay 价格文本解析数值（支持 56,99 / 1.459,19 / 56.99）。
func ParseLocalizedPrice(text string) (float64, error) {
	text = strings.TrimSpace(text)
	m := localizedPriceRE.FindString(text)
	if m == "" {
		return 0, fmt.Errorf("no price number in %q", text)
	}
	if strings.Contains(m, ",") {
		m = strings.ReplaceAll(m, ".", "")
		m = strings.ReplaceAll(m, ",", ".")
	}
	return strconv.ParseFloat(m, 64)
}

// GetFloat64sFromString 获取字符串中的数字
func GetFloat64sFromString(str string) []float64 {
	var result []float64
	var temp string
	for _, char := range str {
		if char >= '0' && char <= '9' {
			temp += string(char)
		} else {
			if temp != "" {
				last := temp[len(temp)-1]
				if char == '.' && last >= '0' && last <= '9' {
					temp += string(char)
				} else {
					result = append(result, ParseFloat64(temp))
					temp = ""
				}
			}
		}
	}
	if temp != "" {
		result = append(result, ParseFloat64(temp))
	}
	return result
}

// ParseFloat64 将字符串转换为float64
func ParseFloat64(str string) float64 {
	rel, err := strconv.ParseFloat(str, 64)
	if err != nil {
		log.Println("StrToFloat64 err:", err)
		return 0
	}
	return rel
}

// GetPriceFromString 从字符串中获取价格，返回价格的浮点数切片
// str: 待解析的字符串
// currencySymbol: 货币符号，如：€，$，￥等
func GetPriceFromString(str string, currencySymbol string) []float64 {
	var prices []float64

	cur := regexp.QuoteMeta(currencySymbol)
	// 构建正则表达式，匹配前后有货币符号的价格信息
	re := regexp.MustCompile(fmt.Sprintf(`(?:%s\s*([\d.]+)|([\d.]+)\s*%s)`, cur, cur))
	fmt.Println("re: ", re)
	priceMatches := re.FindAllStringSubmatch(str, -1)

	// 遍历匹配到的字符串片段，将数字部分解析为浮点数
	for _, match := range priceMatches {
		// 匹配分组中有匹配则使用，否则使用另一个分组
		priceStr := match[1]
		if priceStr == "" {
			priceStr = match[2]
		}

		// 解析为浮点数
		price, err := strconv.ParseFloat(priceStr, 64)
		if err == nil {
			prices = append(prices, price)
		}
	}
	fmt.Println("parse all prices: ", prices)

	return prices
}
