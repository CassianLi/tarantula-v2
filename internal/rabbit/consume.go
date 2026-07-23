package rabbitmq

import (
	"encoding/json"
	"etarantula/internal/models"
	"etarantula/internal/service"
	"fmt"
	"log"
	"os"
	"runtime/debug"
	"strings"
	"time"

	"github.com/JokerLiAnother/rabbitmq"
	"github.com/spf13/viper"
)

// consumeHandler 消息处理函数
func consumeHandler(msg string) {
	// chromedp 内部 goroutine panic 仍会杀进程；这里兜底业务层 panic，避免拖垮消费者循环。
	defer func() {
		if r := recover(); r != nil {
			log.Printf("consumeHandler panic recovered: %v\n%s", r, debug.Stack())
		}
	}()

	log.Println("收到MQ消息：", msg)

	start := time.Now()
	category, err := parseConsumeMessage(msg)
	if err != nil {
		fmt.Println("Error unmarshalling message, err: ", err, category)
		return
	}

	s := service.NewCategoryService(category)
	if s == nil {
		log.Println("Create service failed")
		return

	}

	info, err := s.GetCategoryInfo()
	// 调试时，打印回传信息
	if viper.GetBool("debug") {
		// categoryInfo 转json string
		categoryInfoJson, err := json.Marshal(info)
		if err != nil {
			log.Println("即将回传截图消息MQ, categoryInfo: ", info)
		} else {
			log.Println("即将回传截图消息MQ, categoryInfo(json): ", string(categoryInfoJson))
		}
	}
	end := time.Now()
	log.Println("获取品类截图信息总耗时：", end.Sub(start))

	start = time.Now()
	// publish category info to MQ
	err = publishInfoFn(info)
	if err != nil {
		log.Println("回传截图信息，失败, err: ", err)
		return
	}
	end = time.Now()
	log.Println("回传截图信息总耗时:", end.Sub(start))

}

// publishInfoFn 供单元测试替换，避免回传 MQ。
var publishInfoFn = publishInfo

// parseConsumeMessage 解析 MQ 消息体。
func parseConsumeMessage(msg string) (models.CategoryInfoRequest, error) {
	var category models.CategoryInfoRequest
	err := json.Unmarshal([]byte(msg), &category)
	if err != nil {
		log.Println("Warning: unmarshalling message, include double quotes, try to process escaped JSON string, err: ", err, "msg: ", msg)
		var jsonStr string
		if err2 := json.Unmarshal([]byte(msg), &jsonStr); err2 == nil && jsonStr != "" {
			err = json.Unmarshal([]byte(jsonStr), &category)
		}
	}
	if err != nil {
		return category, err
	}
	if category.Country == "" || category.ProductNo == "" {
		return category, fmt.Errorf("missing country or asin")
	}
	category.Country = strings.ToUpper(category.Country)
	return category, nil
}

// publishInfo 发布回传信息到MQ
func publishInfo(info models.CategoryInfo) error {
	// type 转json string
	infoJson, err := json.Marshal(info)
	if err != nil {
		fmt.Println("Error marshalling message, err: ", err)
		return err
	}

	mq, err := rabbitmq.NewDefaultRabbitMQ(viper.GetString("mq.publish.url"),
		viper.GetString("mq.publish.exchange"),
		viper.GetString("mq.publish.exchange-type"),
		viper.GetString("mq.publish.queue"),
		true)
	if err != nil {
		fmt.Println("创建消息回传MQ链接失败，Error: ", err)
		return err
	}
	if mq == nil {
		return fmt.Errorf("publish mq is nil")
	}
	defer mq.Close()

	err = mq.Publish(infoJson)
	if err != nil {
		fmt.Println("Error publishing message, err: ", err)
		return err
	}
	return nil
}

// Consuming 启动消费者
func Consuming() {
	mq, err := rabbitmq.NewRabbitMQ(
		viper.GetString("mq.consumer.url"),
		viper.GetString("mq.consumer.exchange"),
		viper.GetString("mq.consumer.exchange-type"),
		viper.GetString("mq.consumer.queue"),
		rabbitmq.ConnectionOptions{
			Heartbeat:         time.Duration(viper.GetInt("mq.consumer.heartbeat")) * time.Second,
			ReConnectInterval: time.Duration(viper.GetInt("mq.consumer.reconnect-interval")) * time.Second,
			MaxReconnects:     viper.GetInt("mq.consumer.max-reconnects"),
		},
		true,
		rabbitmq.DeclareParams{Durable: true, PrefetchCount: viper.GetInt("mq.consumer.prefetch-count")},
	)

	if err != nil {
		fmt.Println("创建MQ链接失败，Error: ", err)
		return
	}

	// 退出时关闭MQ链接
	defer func() {
		mq.Close()
		if viper.GetBool("mq.consumer.close-exist") {
			log.Println("关闭MQ链接，退出进程...")
			os.Exit(-1)
		}
	}()

	go func() {
		log.Printf("启动消费者，监听中...")
		err := mq.Consume(false, false, true, consumeHandler)
		if err != nil {
			fmt.Println("消费MQ消息失败，Error: ", err)
			return
		}
	}()
}
