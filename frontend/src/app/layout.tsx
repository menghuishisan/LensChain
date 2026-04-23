// layout.tsx
// Next.js 根布局，挂载全局样式、元信息与客户端 Providers。

import type { Metadata } from "next";
import type { ReactNode } from "react";

import { AppProviders } from "@/app/providers";
import "@/app/globals.css";

/**
 * 应用根元信息。
 */
export const metadata: Metadata = {
  title: "链镜平台 LensChain",
  description: "区块链教学、实验实践与 CTF 竞赛一体化平台",
};

/**
 * RootLayout 应用根布局。
 */
export default function RootLayout({ children }: { children: ReactNode }) {
  return (
    <html lang="zh-CN" suppressHydrationWarning>
      <body>
        <AppProviders>{children}</AppProviders>
      </body>
    </html>
  );
}
