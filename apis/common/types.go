package common

type Image struct {
	Repository string `json:"repository"`
	Image      string `json:"image"`
	Tag        string `json:"tag"`
}
