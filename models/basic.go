package models

type BasicAuthConfigurationItem struct {
	UserName string `json:"username"`
	Password string `json:"password"`
	Url      string `json:"url"`
}
