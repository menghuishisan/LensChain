// common.go
// 模型层通用 DTO 结构。
// 该文件只放跨多个模块复用、且不带业务模块语义的请求/响应结构，避免通用结构挂靠到某个模块文件下。

package dto

// PaginationResp 通用分页响应。
// 该结构用于统一承载列表接口的分页元数据，不表达任何模块业务语义。
type PaginationResp struct {
	Page       int `json:"page"`
	PageSize   int `json:"page_size"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

// SortItemReq 通用排序项请求。
// 该结构用于批量排序类接口，表达目标 ID 与排序值的对应关系。
type SortItemReq struct {
	ID        string `json:"id" binding:"required"`
	SortOrder int    `json:"sort_order" binding:"min=0"`
}

// SortReq 通用排序请求。
// 该结构用于接收一组排序项，供课程、实验、竞赛等模块复用。
type SortReq struct {
	Items []SortItemReq `json:"items" binding:"required,min=1,dive"`
}
