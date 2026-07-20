package config

import (
	"context"
	"fmt"

	"etarantula/internal/utils"

	"github.com/spf13/viper"
)

var (
	// GlobalContext 全局上下文
	GlobalContext bool

	// BrowserContext 浏览器上下文
	BrowserContext context.Context
	// BrowserCancel 浏览器上下文取消函数
	BrowserCancel context.CancelFunc
)

// InitBrowserContext 从配置 chromedp.url / chromedp.headless 初始化浏览器上下文
func InitBrowserContext() error {
	fmt.Println("初始化浏览器上下文...")
	var err error
	BrowserContext, BrowserCancel, err = utils.CreateBrowserContext(
		viper.GetString("chromedp.url"),
		viper.GetBool("chromedp.headless"),
	)
	if err != nil {
		return err
	}
	return BrowserContext.Err()
}
