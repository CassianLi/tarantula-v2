package models

// CategoryInfoRequest The request of category info
type CategoryInfoRequest struct {
	ProductNo    string `json:"asin"`
	SalesChannel string `json:"channel"`
	PriceNo      string `json:"priceNo"` // Deprecated: 业务自定义售价主键，可能一对多；请优先使用 priceId
	PriceId      string `json:"priceId"`
	Country      string `json:"country"`
	Price        string `json:"price"`
}

// CategoryInfo The information about category
type CategoryInfo struct {
	ProductNo     string   `json:"asin"`
	SalesChannel  string   `json:"channel"`
	PriceNo       string   `json:"priceNo"` // Deprecated: 业务自定义售价主键，可能一对多；请优先使用 priceId
	PriceId       string   `json:"priceId"`
	Country       string   `json:"country"`
	Price         string   `json:"price"`
	// ebay 可能会将价格转换为美元， local 为原始价格
	NewPrice      string   `json:"newPrice"`
	Currency      string   `json:"currency"`
	LocalPrice    string   `json:"localPrice,omitempty"`
	LocalCurrency string   `json:"localCurrency,omitempty"`
	Screenshot    string   `json:"screenshot"`
	Status        string   `json:"status"`
	Errors        []string `json:"errors"`
}
