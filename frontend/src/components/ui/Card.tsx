// Card.tsx
// 基础卡片组件，统一内容容器、标题区和页内信息块视觉。

import { forwardRef, type HTMLAttributes } from "react";

import { cn } from "@/lib/utils";

/**
 * Card 容器属性。
 */
export interface CardProps extends HTMLAttributes<HTMLDivElement> {}

/**
 * Card 基础卡片容器。
 */
export const Card = forwardRef<HTMLDivElement, CardProps>(({ className, ...props }, ref) => (
  <div
    ref={ref}
    className={cn("rounded-xl border border-border/80 bg-card/92 text-card-foreground shadow-panel backdrop-blur", className)}
    {...props}
  />
));

Card.displayName = "Card";

/**
 * CardHeader 卡片标题区域。
 */
export const CardHeader = forwardRef<HTMLDivElement, CardProps>(({ className, ...props }, ref) => (
  <div ref={ref} className={cn("space-y-1.5 p-5", className)} {...props} />
));

CardHeader.displayName = "CardHeader";

/**
 * CardTitle 卡片标题。
 */
export const CardTitle = forwardRef<HTMLHeadingElement, HTMLAttributes<HTMLHeadingElement>>(
  ({ className, ...props }, ref) => (
    <h3 ref={ref} className={cn("font-display text-xl font-semibold tracking-tight", className)} {...props} />
  ),
);

CardTitle.displayName = "CardTitle";

/**
 * CardDescription 卡片描述。
 */
export const CardDescription = forwardRef<HTMLParagraphElement, HTMLAttributes<HTMLParagraphElement>>(
  ({ className, ...props }, ref) => (
    <p ref={ref} className={cn("text-sm leading-6 text-muted-foreground", className)} {...props} />
  ),
);

CardDescription.displayName = "CardDescription";

/**
 * CardContent 卡片内容区域。
 */
export const CardContent = forwardRef<HTMLDivElement, CardProps>(({ className, ...props }, ref) => (
  <div ref={ref} className={cn("p-5 pt-0", className)} {...props} />
));

CardContent.displayName = "CardContent";

/**
 * CardFooter 卡片底部区域。
 */
export const CardFooter = forwardRef<HTMLDivElement, CardProps>(({ className, ...props }, ref) => (
  <div ref={ref} className={cn("flex items-center gap-3 p-5 pt-0", className)} {...props} />
));

CardFooter.displayName = "CardFooter";
