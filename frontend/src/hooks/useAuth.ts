"use client";

// useAuth.ts
// 模块01认证 hook，封装登录、登出、SSO回调、强制改密和本地认证态。

import { useMutation, useQuery } from "@tanstack/react-query";
import { useRouter } from "next/navigation";

import { getNavigationForRoles, getPrimaryRole } from "@/lib/permissions";
import { selectIsAuthenticated, useAuthStore } from "@/stores/authStore";
import {
  changePassword,
  forceChangePassword,
  getSsoLoginUrl,
  listSsoSchools,
  login,
  logout,
  ssoCallback,
} from "@/services/auth";
import type { AuthTokens, LoginSuccessResponse, LoginResponse, UserRole } from "@/types/auth";

function isLoginSuccess(response: LoginResponse): response is LoginSuccessResponse {
  return "access_token" in response && "user" in response;
}

function toAuthTokens(response: LoginSuccessResponse): AuthTokens {
  return {
    accessToken: response.access_token,
    refreshToken: response.refresh_token,
    tokenType: response.token_type,
    expiresIn: response.expires_in,
  };
}

/**
 * useAuth 读取客户端认证态，并派生主角色、导航和登录状态。
 */
export function useAuth(defaultRole: UserRole = "student") {
  const user = useAuthStore((state) => state.user);
  const isAuthenticated = useAuthStore(selectIsAuthenticated);
  const clearSession = useAuthStore((state) => state.clearSession);
  const tempToken = useAuthStore((state) => state.tempToken);
  const tempTokenExpiresAt = useAuthStore((state) => state.tempTokenExpiresAt);
  const roles = user?.roles ?? [defaultRole];
  const primaryRole = getPrimaryRole(roles);
  const navigation = getNavigationForRoles(roles);

  return {
    user,
    roles,
    primaryRole,
    navigation,
    isAuthenticated,
    tempToken,
    tempTokenExpiresAt,
    clearSession,
  };
}

/**
 * useLoginMutation 调用 POST /auth/login，并将登录成功结果写入认证 store。
 */
export function useLoginMutation() {
  const setSession = useAuthStore((state) => state.setSession);
  const setTempToken = useAuthStore((state) => state.setTempToken);

  return useMutation({
    mutationFn: login,
    onSuccess: (response) => {
      if (isLoginSuccess(response)) {
        setSession(toAuthTokens(response), response.user);
        return;
      }

      setTempToken(response.temp_token, response.temp_token_expires_in);
    },
  });
}

/**
 * useForceChangePasswordMutation 调用 POST /auth/force-change-password，并写入正式登录态。
 */
export function useForceChangePasswordMutation() {
  const setSession = useAuthStore((state) => state.setSession);

  return useMutation({
    mutationFn: forceChangePassword,
    onSuccess: (response) => {
      if (isLoginSuccess(response)) {
        setSession(toAuthTokens(response), response.user);
      }
    },
  });
}

/**
 * useChangePasswordMutation 调用 POST /auth/change-password，用于已登录用户修改密码。
 */
export function useChangePasswordMutation() {
  return useMutation({
    mutationFn: changePassword,
  });
}

/**
 * useLogoutMutation 调用 POST /auth/logout；无论后端是否成功，前端都会清理本地会话并跳转登录页。
 */
export function useLogoutMutation() {
  const clearSession = useAuthStore((state) => state.clearSession);
  const router = useRouter();

  return useMutation({
    mutationFn: logout,
    onSettled: () => {
      clearSession();
      router.replace("/login");
    },
  });
}

/**
 * useSsoSchools 查询已配置 SSO 的学校列表。
 */
export function useSsoSchools(keyword: string) {
  return useQuery({
    queryKey: ["auth", "sso-schools", keyword],
    queryFn: () => listSsoSchools(keyword),
  });
}

/**
 * useSsoLoginUrl 返回 SSO 登录跳转 URL 构造函数。
 */
export function useSsoLoginUrl() {
  return getSsoLoginUrl;
}

/**
 * useSsoCallbackMutation 调用 GET /auth/sso/callback，并写入登录态或临时 Token。
 */
export function useSsoCallbackMutation() {
  const setSession = useAuthStore((state) => state.setSession);
  const setTempToken = useAuthStore((state) => state.setTempToken);

  return useMutation({
    mutationFn: ssoCallback,
    onSuccess: (response) => {
      if (isLoginSuccess(response)) {
        setSession(toAuthTokens(response), response.user);
        return;
      }

      setTempToken(response.temp_token, response.temp_token_expires_in);
    },
  });
}

/**
 * getAuthHomePath 根据最高权限角色决定登录后的首页。
 * 直接落到首个业务页，避免再经过四端工作台。
 */
export function getAuthHomePath(roles: readonly UserRole[]) {
  const primaryRole = getPrimaryRole(roles);
  if (primaryRole === "super_admin") {
    return "/super/schools";
  }
  if (primaryRole === "school_admin") {
    return "/admin/users";
  }
  if (primaryRole === "teacher") {
    return "/teacher/courses";
  }

  return "/student/courses";
}

/**
 * isForceChangeResponse 判断登录响应是否为强制改密。
 */
export function isForceChangeResponse(response: LoginResponse) {
  return !isLoginSuccess(response);
}
