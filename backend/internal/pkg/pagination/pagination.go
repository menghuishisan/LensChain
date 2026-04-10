// pagination.go
// 通用分页封装
// 遵循 API 规范：page 从1开始，page_size 默认20，最大100
// 支持 sort_by 和 sort_order 排序参数

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
	Page      int   `json:"page"`
	PageSize  int   `json:"page_size"`
	Total     int64 `json:"total"`
	TotalPage int   `json:"total_page"`
}

// NewResult 创建分页结果
func NewResult(list interface{}, total int64, page, pageSize int) *Result {
	totalPage := int(total) / pageSize
	if int(total)%pageSize > 0 {
		totalPage++
	}

	return &Result{
		List: list,
		Pagination: Info{
			Page:      page,
			PageSize:  pageSize,
			Total:     total,
			TotalPage: totalPage,
		},
	}
}
