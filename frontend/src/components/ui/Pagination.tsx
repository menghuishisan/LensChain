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

  return (
    <nav className={cn("flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between", className)} aria-label="分页">
      <p className="text-sm text-muted-foreground">
        第 {currentPage} / {safeTotalPages} 页{typeof total === "number" ? `，共 ${total} 条` : ""}
      </p>
      <div className="flex items-center gap-2">
        <Button variant="outline" size="icon" disabled={currentPage <= 1} onClick={() => onPageChange(currentPage - 1)}>
          <ChevronLeft className="h-4 w-4" />
          <span className="sr-only">上一页</span>
        </Button>
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
