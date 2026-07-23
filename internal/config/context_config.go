package config

import (
	"context"
	"fmt"
	"sync"

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

	// BrowserMu 串行化全局浏览器访问。
	// chromedp 同一 BrowserContext 在未分配完成时并发 Run 会 panic: close of closed channel。
	BrowserMu sync.Mutex
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
	return nil
}
