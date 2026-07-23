package service

import (
	"context"
	"errors"
	"etarantula/internal/config"
	"etarantula/internal/models"
	"etarantula/internal/ossutil"
	"etarantula/internal/utils"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/spf13/viper"
)

// ErrProductNotFound 商品页不存在或已下架。
var ErrProductNotFound = errors.New("商品信息不存在")

type EbayCategory struct {
	// Category 请求参数
	Category models.CategoryInfoRequest

	// Errors returned error messages
	Errors []string
}

// ebayItemMissing 判断页面是否为商品不存在/已下架/站点错误页。
func ebayItemMissing(html string) bool {
	lower := strings.ToLower(html)
	markers := []string{
		"error page | ebay",
		"id=\"error-info\"",
		"something went wrong on our end",
		"dieses angebot ist nicht mehr verfügbar",
		"this listing was ended",
		"this listing has been removed",
		"the item is no longer available",
		"hubo un problema",
		"cet objet n'est plus disponible",
		"questo oggetto non è più disponibile",
	}
	for _, m := range markers {
		if strings.Contains(lower, m) {
			return true
		}
	}
	return false
}

// getPageHTML 带超时取整页，避免错误路径上 GetHtml 再拖死消费。
func getPageHTML(ctx context.Context) (string, error) {
	htmlCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	return utils.GetHtml(htmlCtx)
}

// NewCategoryService 创建service
func (ebay *EbayCategory) NewCategoryService(cat models.CategoryInfoRequest) CategoryServiceImpl {
	return &EbayCategory{
		Category: cat,
	}
}

// GetCategoryInfo Get the information about category
func (ebay *EbayCategory) GetCategoryInfo() (info models.CategoryInfo, err error) {
	start := time.Now()
	info = ebay.initCategoryInfo(ebay.Category)
	// 创建一个chrome实例
	ctx, cancel, err := ebay.createContext()
	if err != nil {
		info.Status = PageError
		info.Errors = append(info.Errors, err.Error())
		return info, err
	}
	// cancel 不为空则需要在运行结束后关闭ctx
	if cancel != nil {
		defer cancel()
	}

	// 获取web link
	url := buildEbayItemURL(ebay.Category.Country, ebay.Category.ProductNo)
	log.Println("ebay item url:", url)
	end := time.Now()
	log.Println("1. 获取weblink耗时：", end.Sub(start))

	start = time.Now()
	//err = utils.Navigate(ctx, url)
	//if err != nil {
	//	info.Status = PageError
	//	info.Errors = append(info.Errors, "1.打页面失败")
	//	return info, err
	//}
	//
	err = utils.NavigateAndWait(ctx, url, viper.GetString("ebay.content-selector"), 10*time.Second)
	if err != nil {
		log.Println("页面超时", err)
		// 商品不存在/错误页可能没有 content-selector；取整页 HTML 判断，避免一直卡在截图等待。
		if pageHTML, htmlErr := getPageHTML(ctx); htmlErr == nil && ebayItemMissing(pageHTML) {
			info.Status = PageError
			info.Errors = append(info.Errors, ErrProductNotFound.Error())
			return info, ErrProductNotFound
		}
		info.Status = PageError
		info.Errors = append(info.Errors, "页面超时")
		return info, err
	}

	end = time.Now()
	log.Println("2. 打开weblink耗时：", end.Sub(start))

	// 打开成功后先快速判断错误/下架页，避免后续 selector 轮询拖死消费。
	if pageHTML, htmlErr := getPageHTML(ctx); htmlErr == nil && ebayItemMissing(pageHTML) {
		info.Status = PageError
		info.Errors = append(info.Errors, ErrProductNotFound.Error())
		return info, ErrProductNotFound
	}

	// 下载html
	start = time.Now()
	log.Println("3. 开始下载html...")
	html, err := ebay.downloadHtml(ctx)
	if err != nil {
		// 内容区选择器未命中时，再取整页判断是否为「商品不存在」页，避免误报超时并堵住 MQ。
		if pageHTML, htmlErr := getPageHTML(ctx); htmlErr == nil && ebayItemMissing(pageHTML) {
			info.Status = PageError
			info.Errors = append(info.Errors, ErrProductNotFound.Error())
			return info, ErrProductNotFound
		}
		info.Status = PriceError
		info.Errors = append(info.Errors, "下载html失败")
		return info, err
	}
	end = time.Now()

	log.Println("3. 下载html耗时：", end.Sub(start))

	if ebayItemMissing(html) {
		info.Status = PageError
		info.Errors = append(info.Errors, ErrProductNotFound.Error())
		return info, ErrProductNotFound
	}

	start = time.Now()
	// 解析html
	err = ebay.parseProductInfo(html, &info)
	if err != nil {
		info.Status = PriceError
		if errors.Is(err, ErrProductNotFound) {
			info.Status = PageError
		}
		info.Errors = append(info.Errors, err.Error())
		// 解析失败时不再截图：商品区 selector 可能永不出现，chromedp 无超时会永久阻塞消费。
		return info, err
	}
	end = time.Now()
	log.Println("4. 解析Price耗时：", end.Sub(start))

	start = time.Now()
	// 保存截图
	filename, err := ebay.saveScreenshot(ctx)
	if err != nil {
		info.Status = ScreenshotError
		info.Errors = append(info.Errors, "保存截图失败")
		return info, err
	}
	info.Screenshot = filename
	end = time.Now()
	log.Println("5. 截图并保存总耗时：", end.Sub(start))

	if len(info.Errors) == 0 {
		info.Status = Success
	}

	return info, err
}

// 初始化返回结果
func (ebay *EbayCategory) initCategoryInfo(category models.CategoryInfoRequest) models.CategoryInfo {
	return models.CategoryInfo{
		ProductNo:    category.ProductNo,
		Country:      category.Country,
		SalesChannel: category.SalesChannel,
		PriceNo:      category.PriceNo,
		PriceId:      category.PriceId,
		Price:        category.Price,
	}
}

// 创建一个chrome实例
func (ebay *EbayCategory) createContext() (ctx context.Context, cancel context.CancelFunc, err error) {
	if config.GlobalContext {
		return config.BrowserContext, nil, nil
	} else {
		// 创建一个chrome实例
		return utils.CreateBrowserContext(viper.GetString("chromedp.url"), viper.GetBool("chromedp.headless"))
	}
}

// 下载html页面
func (ebay *EbayCategory) downloadHtml(ctx context.Context) (html string, err error) {
	contentSel := viper.GetString("ebay.content-selector")
	html, err = utils.GetHtmlBySelectors(ctx, contentSel)
	if err != nil {
		log.Println("get html error: ", err)
		return html, err
	}
	return html, err
}

// 解析html页面，获取产品信息
func (ebay *EbayCategory) parseProductInfo(html string, info *models.CategoryInfo) error {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		log.Println("goquery解析html失败", err)
		return err
	}
	priceSelectors := viper.GetString("ebay.price-selectors")
	priceSelectorsArr := strings.Split(priceSelectors, ",")

	var text string
	for _, selector := range priceSelectorsArr {
		selector = strings.TrimSpace(selector)
		if selector == "" {
			continue
		}
		log.Println("range selector: ", selector)
		ele := doc.Find(selector)
		if ele.Length() == 0 {
			continue
		}
		text = strings.TrimSpace(ele.First().Text())
		if text == "" {
			continue
		}
		log.Println("match selector: ", selector)
		break
	}
	fmt.Println("listing price text: ", text)

	if text == "" {
		return ErrProductNotFound
	}

	info.Currency = utils.ExtractPriceCurrencyFromHTML(doc)
	if info.Currency == "" {
		info.Currency = utils.CurrencyFromPriceText(text)
	}

	price, err := utils.ParseLocalizedPrice(text)
	if err != nil {
		log.Println("解析价格失败", err)
		return errors.New("解析价格失败, text: " + text)
	}
	info.NewPrice = strconv.FormatFloat(price, 'f', -1, 64)

	// 非本地货币标价时，eBay 会展示当地货币近似价（如 PL 站 Około 215,11 zł）
	approxSel := strings.TrimSpace(viper.GetString("ebay.price-approx-selector"))
	if approxSel != "" {
		if approxText := strings.TrimSpace(doc.Find(approxSel).First().Text()); approxText != "" {
			if localCur := utils.CurrencyFromPriceText(approxText); localCur != "" && localCur != info.Currency {
				if localPrice, err := utils.ParseLocalizedPrice(approxText); err == nil {
					info.LocalCurrency = localCur
					info.LocalPrice = strconv.FormatFloat(localPrice, 'f', -1, 64)
					fmt.Println("local approx price:", info.LocalPrice, info.LocalCurrency)
				}
			}
		}
	}

	return nil
}

// 以固定高度截图
func (ebay *EbayCategory) getScreenshotByHeight(ctx context.Context) (bytes []byte, err error) {
	height := viper.GetInt64("ebay.screenshot-height")

	if height == 0 {
		height = 960
	}

	start := time.Now()

	// 截图
	bys, err := utils.GetScreenshot(ctx, height)
	if err != nil {
		return bys, err
	}

	end := time.Now()
	log.Println("---- 5.1 以固定高度截图，截图耗时：", end.Sub(start))

	return bys, err
}

// 以selector截图
func (ebay *EbayCategory) getScreenshotBySelector(ctx context.Context) (bytes []byte, err error) {
	selectors := viper.GetString("ebay.screenshot-selector")
	selectorsArr := strings.Split(selectors, ",")

	for _, selector := range selectorsArr {
		bytes, err = utils.GetScreenshotBySelector(ctx, selector)
		if err == nil && len(bytes) > 0 {
			return bytes, nil
		} else {
			fmt.Println("selector: ", selector, "screenshot error: ", err)
		}
	}

	fmt.Println("---- 5.1 以selector截图，所有selector截图失败")
	log.Println("---- 5.1 以selector截图，所有selector截图失败")

	return nil, errors.New("截图失败")
}

// 保存截图
func (ebay *EbayCategory) saveScreenshot(ctx context.Context) (filename string, err error) {
	// 可根据需要选择：
	// 1. 以固定高度截图
	// 或者
	// 2. 以selector截图
	// bytes, err := ebay.getScreenshotByHeight(ctx)
	bytes, err := ebay.getScreenshotBySelector(ctx)
	if err != nil {
		ebay.Errors = append(ebay.Errors, "截图失败")
		return "", err
	}

	country := ebay.Category.Country
	productNo := ebay.Category.ProductNo

	name := "EBAY_O_" + country + "_" + productNo + "_" + time.Now().Format("060102150105") + ".png"
	if err := utils.SaveScreenshotLocal(name, bytes); err != nil {
		fmt.Println("保存截图到磁盘失败，文件名：", name, err)
	}

	filename = utils.OSSObjectKey(name)

	log.Println("开始上传截图到OSS...")
	start := time.Now()
	ali, err := ossutil.NewAliOss(viper.GetString("oss.endpoint"), viper.GetString("oss.access-key-id"), viper.GetString("oss.access-key-secret"))
	if err != nil {
		log.Println("创建OSS客户端失败", err)
		ebay.Errors = append(ebay.Errors, "创建OSS客户端失败")
		return
	}

	err = ali.UploadByte(viper.GetString("oss.bucket"), filename, bytes)
	if err != nil {
		log.Println("上传截图到OSS失败", err)
		ebay.Errors = append(ebay.Errors, "上传截图到OSS失败")
		return
	}
	end := time.Now()
	log.Println("---- 5.2 上传截图到OSS耗时：", end.Sub(start))

	return filename, err
}

// buildEbayItemURL 按国家代码生成 eBay 商品页链接。
func buildEbayItemURL(country, productNo string) string {
	country = strings.ToLower(strings.TrimSpace(country))
	if u := viper.GetString("ebay.urls." + country); u != "" {
		return strings.ReplaceAll(u, "ASIN", productNo)
	}
	return "https://www.ebay." + ebayTLD(country) + "/itm/" + productNo
}

func ebayTLD(country string) string {
	switch strings.ToUpper(strings.TrimSpace(country)) {
	case "GB", "UK":
		return "co.uk"
	case "US":
		return "com"
	case "AU":
		return "com.au"
	default:
		return strings.ToLower(country)
	}
}
