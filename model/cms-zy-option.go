package model

type KV1 struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type CmsZyOption struct {
	Name string
	Api  string
	Tags map[string]string // 换其他类型就不行。。。
}

func (x *CmsZyOption) SetName(name string) {
	x.Name = name
}
func (x *CmsZyOption) GetName() string {
	return x.Name
}

func (x *CmsZyOption) SetApi(api string) {
	x.Api = api
}
func (x *CmsZyOption) GetApi() string {
	return x.Api
}

func (x *CmsZyOption) SetTags(tags map[string]string) {
	x.Tags = tags
}
func (x *CmsZyOption) GetTags() map[string]string {
	return x.Tags
}
