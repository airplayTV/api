package model

type Video struct {
	Id         string `json:"id"`
	Name       string `json:"name"`
	Thumb      string `json:"thumb"`
	Intro      string `json:"intro"`
	Url        string `json:"url"`
	Actors     string `json:"actors"`
	Tag        string `json:"tag"`
	Resolution string `json:"resolution"`
	UpdatedAt  string `json:"updated_at"`
	Links      []Link `json:"links"`
}
