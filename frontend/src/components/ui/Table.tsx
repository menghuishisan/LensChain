// Table.tsx
// 基础表格组件，统一表格容器、表头、行和单元格样式。

import { forwardRef, type HTMLAttributes, type TdHTMLAttributes, type ThHTMLAttributes } from "react";

import { cn } from "@/lib/utils";

/**
 * TableContainer 表格滚动容器属性。
 */
export interface TableContainerProps extends HTMLAttributes<HTMLDivElement> {}

/**
 * TableContainer 表格滚动容器。
 */
export const TableContainer = forwardRef<HTMLDivElement, TableContainerProps>(({ className, ...props }, ref) => (
  <div ref={ref} className={cn("w-full overflow-x-auto rounded-xl border border-border bg-card", className)} {...props} />
));

TableContainer.displayName = "TableContainer";

/**
 * Table 表格根元素。
 */
export const Table = forwardRef<HTMLTableElement, HTMLAttributes<HTMLTableElement>>(({ className, ...props }, ref) => (
  <table ref={ref} className={cn("w-full caption-bottom text-sm", className)} {...props} />
));

Table.displayName = "Table";

/**
 * TableHeader 表头区域。
 */
export const TableHeader = forwardRef<HTMLTableSectionElement, HTMLAttributes<HTMLTableSectionElement>>(
  ({ className, ...props }, ref) => <thead ref={ref} className={cn("bg-muted/70", className)} {...props} />,
);

TableHeader.displayName = "TableHeader";

/**
 * TableBody 表体区域。
 */
export const TableBody = forwardRef<HTMLTableSectionElement, HTMLAttributes<HTMLTableSectionElement>>(
  ({ className, ...props }, ref) => <tbody ref={ref} className={cn("divide-y divide-border", className)} {...props} />,
);

TableBody.displayName = "TableBody";

/**
 * TableRow 表格行。
 */
export const TableRow = forwardRef<HTMLTableRowElement, HTMLAttributes<HTMLTableRowElement>>(
  ({ className, ...props }, ref) => (
    <tr ref={ref} className={cn("transition-colors hover:bg-muted/45 data-[state=selected]:bg-muted", className)} {...props} />
  ),
);

TableRow.displayName = "TableRow";

/**
 * TableHead 表头单元格。
 */
export const TableHead = forwardRef<HTMLTableCellElement, ThHTMLAttributes<HTMLTableCellElement>>(
  ({ className, ...props }, ref) => (
    <th
      ref={ref}
      className={cn("h-11 px-4 text-left align-middle text-xs font-semibold uppercase tracking-wide text-muted-foreground", className)}
      {...props}
    />
  ),
);

TableHead.displayName = "TableHead";

/**
 * TableCell 表体单元格。
 */
export const TableCell = forwardRef<HTMLTableCellElement, TdHTMLAttributes<HTMLTableCellElement>>(
  ({ className, ...props }, ref) => <td ref={ref} className={cn("px-4 py-3 align-middle", className)} {...props} />,
);

TableCell.displayName = "TableCell";
