package model

type ProviderItem struct {
	Name string      `json:"name"`
	Sort int         `json:"sort"`
	Tags interface{} `json:"tags"`
}
