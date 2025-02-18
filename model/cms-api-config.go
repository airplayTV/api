package model

type CmsApiConfig struct {
	Id   string `mapstructure:"id"`
	Name string `mapstructure:"name"`
	Host string `mapstructure:"host"`
	Api  string `mapstructure:"api"`
}
