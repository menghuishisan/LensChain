// api.ts
// 全局 API 类型定义，统一约束后端响应、分页结构、错误对象与雪花 ID。

/**
 * 雪花 ID 类型别名。
 * 后端 BIGINT 雪花 ID 超出 JavaScript 安全整数范围，前端一律以字符串处理。
 */
export type ID = string;

/**
 * 后端统一响应结构。
 */
export interface ApiResponse<TData> {
  code: number;
  message: string;
  data: TData;
  timestamp?: string | number;
}

/**
 * 分页元信息。
 */
export interface Pagination {
  page: number;
  page_size: number;
  total: number;
  total_pages: number;
}

/**
 * 分页列表响应 data 结构。
 */
export interface PaginatedData<TItem> {
  list: TItem[];
  pagination: Pagination;
}

/**
 * API 归一化错误对象。
 */
export interface ApiError {
  code: number;
  message: string;
  status?: number;
  data?: unknown;
  timestamp?: string | number;
}

/**
 * URL 查询参数值。
 */
export type QueryValue = string | number | boolean | null | undefined;

/**
 * URL 查询参数集合，数组值会展开为重复 query key。
 */
export type QueryParams = Record<string, QueryValue | readonly QueryValue[]>;
