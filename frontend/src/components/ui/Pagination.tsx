"use client";

// Pagination.tsx
// 基础分页组件，统一列表页分页交互。

import { ChevronLeft, ChevronRight } from "lucide-react";

import { Button } from "@/components/ui/Button";
import { cn } from "@/lib/utils";

/**
 * Pagination 组件属性。
 */
export interface PaginationProps {
  page: number;
  totalPages: number;
  total?: number;
  className?: string;
  onPageChange: (page: number) => void;
}

function getVisiblePages(page: number, totalPages: number) {
  const start = Math.max(1, page - 2);
  const end = Math.min(totalPages, start + 4);
  return Array.from({ length: end - start + 1 }, (_, index) => start + index);
}

/**
 * Pagination 基础分页组件。
 */
export function Pagination({ page, totalPages, total, className, onPageChange }: PaginationProps) {
  const safeTotalPages = Math.max(totalPages, 1);
  const currentPage = Math.min(Math.max(page, 1), safeTotalPages);
  const pages = getVisiblePages(currentPage, safeTotalPages);

  if (safeTotalPages <= 1) {
    return typeof total === "number" ? (
      <p className={cn("text-sm text-muted-foreground", className)}>共 {total} 条</p>
    ) : null;
  }

  return (
    <nav className={cn("flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between", className)} aria-label="分页">
      <p className="text-sm text-muted-foreground">
        {typeof total === "number" ? `共 ${total} 条` : `第 ${currentPage} 页`}
      </p>
      <div className="flex items-center gap-1">
        <Button variant="outline" size="icon" disabled={currentPage <= 1} onClick={() => onPageChange(currentPage - 1)}>
          <ChevronLeft className="h-4 w-4" />
          <span className="sr-only">上一页</span>
        </Button>
        {pages[0] > 1 ? (
          <>
            <Button variant="outline" size="sm" onClick={() => onPageChange(1)}>1</Button>
            {pages[0] > 2 ? <span className="px-1 text-sm text-muted-foreground">…</span> : null}
          </>
        ) : null}
        {pages.map((item) => (
          <Button
            key={item}
            variant={item === currentPage ? "primary" : "outline"}
            size="sm"
            onClick={() => onPageChange(item)}
          >
            {item}
          </Button>
        ))}
        {pages[pages.length - 1] < safeTotalPages ? (
          <>
            {pages[pages.length - 1] < safeTotalPages - 1 ? <span className="px-1 text-sm text-muted-foreground">…</span> : null}
            <Button variant="outline" size="sm" onClick={() => onPageChange(safeTotalPages)}>{safeTotalPages}</Button>
          </>
        ) : null}
        <Button
          variant="outline"
          size="icon"
          disabled={currentPage >= safeTotalPages}
          onClick={() => onPageChange(currentPage + 1)}
        >
          <ChevronRight className="h-4 w-4" />
          <span className="sr-only">下一页</span>
        </Button>
      </div>
    </nav>
  );
}
