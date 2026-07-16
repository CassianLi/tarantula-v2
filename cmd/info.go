/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"github.com/spf13/cobra"
	"log"
	"strings"
	"etarantula/internal/config"
	"etarantula/internal/models"
	"etarantula/internal/service"
)

var channel, productNo, country string

// infoCmd represents the info command
var infoCmd = &cobra.Command{
	Use:   "info",
	Short: "请求品类信息",
	Long: `通过命令行获取 eBay 品类信息，例如：
etarantula info --config .tarantula.yaml --product 123456789 --country de --channel ebay`,
	Run: func(cmd *cobra.Command, args []string) {

		log.Println("info called")
		// 初始化配置
		err := config.InitBrowserContext("")
		if err != nil {
			log.Println("Init browser context failed, err:", err)
			return
		}

		country = strings.ToLower(country)
		category := &models.CategoryInfoRequest{
			ProductNo:    productNo,
			Country:      strings.ToUpper(country),
			SalesChannel: channel,
		}

		s := service.NewCategoryService(*category)
		if s == nil {
			log.Println("Create service failed")
			return

		}

		info, err := s.GetCategoryInfo()
		if err != nil {
			log.Println("Get category info failed, err:", err)
		}

		log.Println("Category info: ", info)
	},
}

func init() {
	rootCmd.AddCommand(infoCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	infoCmd.PersistentFlags().StringVar(&channel, "channel", "ebay", "Sales channel (amazon 已弃用，请使用 ebay)")
	infoCmd.PersistentFlags().StringVar(&country, "country", "de", "country code,default=de")
	infoCmd.PersistentFlags().StringVar(&productNo, "product", "", "product number")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// infoCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
