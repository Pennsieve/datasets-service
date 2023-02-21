package models

type TrashcanPage struct {
	Limit      int            `json:"limit"`
	Offset     int            `json:"offset"`
	TotalCount int            `json:"totalCount"`
	Packages   []TrashcanItem `json:"packages"`
	Messages   []string       `json:"messages"`
}

type TrashcanItem struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Path string `json:"path"`
}
