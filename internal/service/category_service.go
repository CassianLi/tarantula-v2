package service

import (
	"etarantula/internal/models"
	"fmt"
	"log"

	"github.com/spf13/viper"
)

// NewCategoryService 创建service
func NewCategoryService(cat models.CategoryInfoRequest) CategoryServiceImpl {
	fmt.Println("debug:", viper.GetBool("debug"))
	channel := cat.SalesChannel
	switch channel {
	case "amazon":
		// Deprecated: Amazon 功能已迁移至其他项目，本工程仅支持 eBay。
		log.Println("Deprecated: channel=amazon 已弃用，请使用独立的 Amazon 项目")
		return nil
	case "ebay":
		return &EbayCategory{
			Category: cat,
		}
	default:
		log.Println("Can't find the sales channel: ", channel)
		return nil
	}
}
