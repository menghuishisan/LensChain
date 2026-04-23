"use client";

// authStore.ts
// Zustand 认证状态，保存客户端登录态和基础用户信息；不直接调用任何 API。

import { create } from "zustand";
import { createJSONStorage, persist } from "zustand/middleware";

import { AUTH_STORAGE_KEY } from "@/lib/auth-session";
import type { AuthSession, AuthTokens, AuthUser } from "@/types/auth";

/**
 * 认证 store 状态和操作。
 */
export interface AuthStoreState extends AuthSession {
  setSession: (tokens: AuthTokens, user: AuthUser) => void;
  setTokens: (tokens: AuthTokens) => void;
  setTempToken: (tempToken: string, expiresIn: number) => void;
  clearTempToken: () => void;
  setUser: (user: AuthUser | null) => void;
  clearSession: () => void;
}

const initialState: AuthSession = {
  accessToken: null,
  refreshToken: null,
  tokenType: "Bearer",
  expiresAt: null,
  user: null,
  tempToken: null,
  tempTokenExpiresAt: null,
};

/**
 * useAuthStore 管理本地认证态；服务端权威用户详情仍应通过 TanStack Query 获取。
 */
export const useAuthStore = create<AuthStoreState>()(
  persist(
    (set, get) => ({
      ...initialState,
      setSession: (tokens, user) =>
        set({
          accessToken: tokens.accessToken,
          refreshToken: tokens.refreshToken,
          tokenType: tokens.tokenType,
          expiresAt: Date.now() + tokens.expiresIn * 1000,
          user,
          tempToken: null,
          tempTokenExpiresAt: null,
        }),
      setTokens: (tokens) =>
        set({
          accessToken: tokens.accessToken,
          refreshToken: tokens.refreshToken,
          tokenType: tokens.tokenType,
          expiresAt: Date.now() + tokens.expiresIn * 1000,
        }),
      setTempToken: (tempToken, expiresIn) =>
        set({
          tempToken,
          tempTokenExpiresAt: Date.now() + expiresIn * 1000,
        }),
      clearTempToken: () =>
        set({
          tempToken: null,
          tempTokenExpiresAt: null,
        }),
      setUser: (user) => set({ user }),
      clearSession: () => set(initialState),
    }),
    {
      name: AUTH_STORAGE_KEY,
      storage: createJSONStorage(() => localStorage),
      partialize: (state) => ({
        accessToken: state.accessToken,
        refreshToken: state.refreshToken,
        tokenType: state.tokenType,
        expiresAt: state.expiresAt,
        user: state.user,
        tempToken: state.tempToken,
        tempTokenExpiresAt: state.tempTokenExpiresAt,
      }),
    },
  ),
);

/**
 * 获取当前是否已登录。
 */
export function selectIsAuthenticated(state: AuthStoreState) {
  return state.accessToken !== null && state.user !== null;
}
