package utils

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/chromedp/chromedp"
)

// GetUserAgent Generative random User-Agent
func GetUserAgent() string {
	userAgents := []string{
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/58.0.3029.110 Safari/537.36",
		"Mozilla/5.0 (Windows NT 6.1; WOW64; rv:54.0) Gecko/20100101 Firefox/54.0",
		"Mozilla/5.0 (Windows NT 10.0; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/51.0.2704.63 Safari/537.36",
		"Mozilla/5.0 (Windows NT 6.3; WOW64; Trident/7.0; AS; rv:11.0) like Gecko",
		"Mozilla/5.0 (Windows NT 6.1; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/51.0.2704.79 Safari/537.36 Edge/14.14393",
		"Mozilla/5.0 (Windows NT 6.1; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/58.0.3029.110 Safari/537.36",
		"Mozilla/5.0 (Windows NT 6.1; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/51.0.2704.79 Safari/537.36 Edge/14.14393",
		"Mozilla/5.0 (Windows NT 6.1; Trident/7.0; rv:11.0) like Gecko",
		"Mozilla/5.0 (Windows NT 6.1; WOW64; Trident/7.0; AS; rv:11.0) like Gecko",
		"Mozilla/5.0 (Windows NT 6.1; Win64; x64; Trident/7.0; AS; rv:11.0) like Gecko",
	}
	return userAgents[rand.Intn(len(userAgents))]
}

// CreateContext create context
func CreateContext(headless bool, timeout time.Duration) (context.Context, context.CancelFunc) {
	ctx, cancel := chromedp.NewContext(context.Background())

	// Set timeout for the context
	ctx, timeoutCancel := context.WithTimeout(ctx, timeout)

	// Set the options for the Chrome browser
	chromeOpts := []chromedp.ExecAllocatorOption{
		// Add more options as needed
	}

	// Add Headless option if needed
	if headless {
		chromeOpts = append(chromeOpts, chromedp.Headless)
	}

	// Create a new context with custom configuration
	allocCtx, allocCancel := chromedp.NewExecAllocator(ctx, chromeOpts...)
	ctx, _ = chromedp.NewContext(allocCtx)

	// Return the context and cancel functions
	return ctx, func() {
		timeoutCancel()
		allocCancel()
		cancel()
	}

}

// CreateBrowserContext 创建一个chrome实例
func CreateBrowserContext(debugUrl string) (ctx context.Context, cancel context.CancelFunc, err error) {
	if debugUrl == "" {
		debugUrl = "http://localhost:9222"
	}
	// 创建一个chrome实例
	ctx, cancel = chromedp.NewRemoteAllocator(context.Background(), debugUrl)

	// create a new chrome instance
	ctx, cancel = chromedp.NewContext(ctx)

	return ctx, cancel, ctx.Err()
}

// Navigate 打开链接
func Navigate(ctx context.Context, url string) error {
	// navigate to the URL
	return chromedp.Run(ctx, chromedp.Navigate(url))
}

// GetScreenshot chromedp 获取当前页面截图，并返回base64编码
func GetScreenshot(ctx context.Context, height int64) ([]byte, error) {
	if height == 0 {
		err := chromedp.Run(ctx, chromedp.EvaluateAsDevTools("Math.max(document.body.scrollHeight, document.documentElement.scrollHeight, document.body.offsetHeight, document.documentElement.offsetHeight, document.body.clientHeight, document.documentElement.clientHeight)", &height))
		if err != nil {
			fmt.Println("chromedp.Run EvaluateAsDevTools err(get page max height):", err)
		}
	}

	var buf []byte
	err := chromedp.Run(ctx,
		// 设置窗口
		chromedp.EmulateViewport(1920, int64(int(height))),
		chromedp.CaptureScreenshot(&buf))
	if err != nil {
		return nil, err
	}

	return buf, nil
}

// GetHtml chromedp 获取当前页面html
func GetHtml(ctx context.Context) (string, error) {
	var html string
	err := chromedp.Run(ctx, chromedp.OuterHTML("html", &html))
	if err != nil {
		return "", err
	}
	return html, nil
}

// GetElementBottomRightHeight 获取元素底部距离页面顶部的高度
func GetElementBottomRightHeight(ctx context.Context, selector string) (int64, error) {
	// 指定超时时间，等待元素加载完成
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	err := chromedp.Run(ctx, chromedp.WaitVisible(selector))
	if err != nil {
		log.Println("chromedp.Run WaitVisible err:", selector)
		return 0, err
	}

	var height float64

	// 定义要执行的JavaScript代码
	script := fmt.Sprintf(`
		(function() {
			const elem = document.querySelector('%s');
			const rect = elem.getBoundingClientRect();
			return rect.bottom;
		})();
	`, selector)

	// 执行JavaScript代码并获取结果
	if err := chromedp.Run(ctx, chromedp.Evaluate(script, &height)); err != nil {
		return 0, err
	}

	return int64(height), nil
}