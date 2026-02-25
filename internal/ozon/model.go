package ozon

type GetStockDataRequest struct {
	Cursor string                 `json:"cursor"`
	Filter map[string]interface{} `json:"filter"`
	Limit  int                    `json:"limit"`
}

type GetStockDataResponse struct {
	Items  []Item `json:"items"`
	Total  int    `json:"total"`
	Cursor string `json:"cursor"`
}

type Item struct {
	ProductID int     `json:"product_id"`
	OfferID   string  `json:"offer_id"`
	Stocks    []Stock `json:"stocks"`
}

type Stock struct {
	Type         string   `json:"type"`
	Present      int      `json:"present"`
	Reserved     int      `json:"reserved"`
	SKU          int      `json:"sku"`
	ShipmentType string   `json:"shipment_type"`
	WarehouseIDs []string `json:"warehouse_ids"`
}

type ApiEndpoint struct {
	Endpoint string
	Method   string
}

type GetPriceDataRequest struct {
	Cursor string                 `json:"cursor"`
	Filter map[string]interface{} `json:"filter"`
	Limit  int                    `json:"limit"`
}

type GetPriceDataResponse struct {
	Items  []PriceItem `json:"items"`
	Total  int         `json:"total"`
	Cursor string      `json:"cursor"`
}

type PriceItem struct {
	OfferID string `json:"offer_id"`
	Price   Price  `json:"price"`
}

type Price struct {
	Price                float64 `json:"price"`
	CurrencyCode         string  `json:"currency_code"`
	MarketingSellerPrice float64 `json:"marketing_seller_price"`
}
