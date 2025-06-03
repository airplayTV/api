package util

import "github.com/microcosm-cc/bluemonday"

func HtmlToText(html string) string {
	// 创建一个严格策略（去除所有标签）
	pStrict := bluemonday.StrictPolicy()
	return pStrict.Sanitize(html)
}
