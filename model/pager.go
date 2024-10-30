package model

type Pager struct {
	Total int     `json:"total"` // 总记录数，前端限制分页用
	Pages int     `json:"pages"` // 总页数
	Page  int     `json:"page"`  // 当前页
	Limit int     `json:"limit"` // 每页限制数量
	List  []Video `json:"list"`  // 视频列表
}
