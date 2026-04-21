// pagination.go
// 该文件提供与 API 规范对齐的统一分页能力，负责处理页码归一化、偏移量计算、排序白名单
// 应用和分页结果结构构造。列表接口若需要分页与排序，应优先复用这里的规则，避免各模块
// 对 `page` 和 `page_size` 的解释不一致。

package pagination

import (
	"gorm.io/gorm"
)

// 分页默认值
const (
	DefaultPage     = 1
	DefaultPageSize = 20
	MaxPageSize     = 100
)

// 允许的排序方向
var allowedSortOrders = map[string]bool{
	"asc":  true,
	"desc": true,
}

// Query 分页查询参数
type Query struct {
	Page      int    `form:"page" binding:"omitempty,min=1"`
	PageSize  int    `form:"page_size" binding:"omitempty,min=1,max=100"`
	SortBy    string `form:"sort_by"`
	SortOrder string `form:"sort_order" binding:"omitempty,oneof=asc desc"`
}

// Normalize 规范化分页参数，设置默认值
func (q *Query) Normalize() {
	if q.Page <= 0 {
		q.Page = DefaultPage
	}
	if q.PageSize <= 0 {
		q.PageSize = DefaultPageSize
	}
	if q.PageSize > MaxPageSize {
		q.PageSize = MaxPageSize
	}
	if q.SortOrder == "" {
		q.SortOrder = "desc"
	}
	if !allowedSortOrders[q.SortOrder] {
		q.SortOrder = "desc"
	}
}

// Offset 计算偏移量
func (q *Query) Offset() int {
	return (q.Page - 1) * q.PageSize
}

// NormalizeValues 规范化分页数值
// 供不直接使用 Query 结构体的 handler/service/repository 复用，避免各模块重复实现分页默认值。
func NormalizeValues(page, pageSize int) (int, int) {
	q := Query{Page: page, PageSize: pageSize}
	q.Normalize()
	return q.Page, q.PageSize
}

// Offset 计算分页偏移量
func Offset(page, pageSize int) int {
	page, pageSize = NormalizeValues(page, pageSize)
	return (page - 1) * pageSize
}

// ApplyToGORM 将分页和排序参数应用到 GORM 查询
// allowedSortFields 为允许排序的字段白名单，防止SQL注入
func (q *Query) ApplyToGORM(db *gorm.DB, allowedSortFields map[string]string) *gorm.DB {
	q.Normalize()

	// 应用排序（仅允许白名单中的字段）
	if q.SortBy != "" {
		if dbField, ok := allowedSortFields[q.SortBy]; ok {
			db = db.Order(dbField + " " + q.SortOrder)
		}
	}

	// 应用分页
	db = db.Offset(q.Offset()).Limit(q.PageSize)

	return db
}

// Result 分页结果
type Result struct {
	List       interface{} `json:"list"`
	Pagination Info        `json:"pagination"`
}

// Info 分页信息
type Info struct {
	Page       int   `json:"page"`
	PageSize   int   `json:"page_size"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"total_pages"`
}

// NewResult 创建分页结果
func NewResult(list interface{}, total int64, page, pageSize int) *Result {
	page, pageSize = NormalizeValues(page, pageSize)
	totalPage := int(total) / pageSize
	if int(total)%pageSize > 0 {
		totalPage++
	}

	return &Result{
		List: list,
		Pagination: Info{
			Page:       page,
			PageSize:   pageSize,
			Total:      total,
			TotalPages: totalPage,
		},
	}
}
