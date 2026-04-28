"use client";

// layoutStore.ts
// 已登录布局偏好状态，保存侧边栏收起等纯客户端界面习惯。

import { create } from "zustand";
import { createJSONStorage, persist } from "zustand/middleware";

const LAYOUT_STORAGE_KEY = "lenschain-layout";

/**
 * LayoutStoreState 布局偏好状态。
 */
export interface LayoutStoreState {
  isSidebarCollapsed: boolean;
  setSidebarCollapsed: (collapsed: boolean) => void;
  toggleSidebarCollapsed: () => void;
}

/**
 * useLayoutStore 管理布局壳的本地 UI 偏好。
 */
export const useLayoutStore = create<LayoutStoreState>()(
  persist(
    (set) => ({
      isSidebarCollapsed: false,
      setSidebarCollapsed: (collapsed) => set({ isSidebarCollapsed: collapsed }),
      toggleSidebarCollapsed: () => set((state) => ({ isSidebarCollapsed: !state.isSidebarCollapsed })),
    }),
    {
      name: LAYOUT_STORAGE_KEY,
      storage: createJSONStorage(() => localStorage),
      partialize: (state) => ({
        isSidebarCollapsed: state.isSidebarCollapsed,
      }),
    },
  ),
);

