// auth-session.ts
// 认证会话持久化工具，供 Zustand store 与 api-client 共享 Token 存取逻辑。

import { USER_ROLES, type AuthSession, type AuthTokens, type AuthUser, type UserRole } from "@/types/auth";

/**
 * Zustand 持久化认证状态的 localStorage key。
 */
export const AUTH_STORAGE_KEY = "lenschain-auth";

/**
 * 认证跳转提示的 sessionStorage key。
 */
export const AUTH_REDIRECT_MESSAGE_KEY = "lenschain-auth-message";

const DEFAULT_TOKEN_TYPE = "Bearer";

interface PersistedAuthStorage {
  state: AuthSession;
  version?: number;
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null;
}

function getLocalStorage() {
  return typeof window === "undefined" ? null : window.localStorage;
}

function getSessionStorage() {
  return typeof window === "undefined" ? null : window.sessionStorage;
}

function readString(value: unknown): string | null {
  return typeof value === "string" && value.length > 0 ? value : null;
}

function readNumber(value: unknown): number | null {
  return typeof value === "number" && Number.isFinite(value) ? value : null;
}

function isUserRole(value: unknown): value is UserRole {
  return typeof value === "string" && USER_ROLES.includes(value as UserRole);
}

function readUser(value: unknown): AuthUser | null {
  if (!isRecord(value)) {
    return null;
  }

  const id = readString(value.id);
  const name = readString(value.name);
  const rawRoles = Array.isArray(value.roles) ? value.roles : [];
  const roles = rawRoles.filter(isUserRole);

  if (id === null || name === null || roles.length === 0) {
    return null;
  }

  return {
    id,
    name,
    roles,
    phone: readString(value.phone) ?? undefined,
    school_id: readString(value.school_id),
    school_name: readString(value.school_name),
    avatar_url: readString(value.avatar_url),
    is_first_login: typeof value.is_first_login === "boolean" ? value.is_first_login : undefined,
  };
}

function createEmptySession(): AuthSession {
  return {
    accessToken: null,
    refreshToken: null,
    tokenType: DEFAULT_TOKEN_TYPE,
    expiresAt: null,
    user: null,
    tempToken: null,
    tempTokenExpiresAt: null,
  };
}

function readPersistedAuth(): PersistedAuthStorage {
  const storage = getLocalStorage();
  if (storage === null) {
    return { state: createEmptySession(), version: 0 };
  }

  const raw = storage.getItem(AUTH_STORAGE_KEY);
  if (raw === null) {
    return { state: createEmptySession(), version: 0 };
  }

  try {
    const parsed: unknown = JSON.parse(raw);
    if (!isRecord(parsed) || !isRecord(parsed.state)) {
      return { state: createEmptySession(), version: 0 };
    }

    const state = parsed.state;
    return {
      state: {
        accessToken: readString(state.accessToken),
        refreshToken: readString(state.refreshToken),
        tokenType: readString(state.tokenType) ?? DEFAULT_TOKEN_TYPE,
        expiresAt: readNumber(state.expiresAt),
        user: readUser(state.user),
        tempToken: readString(state.tempToken),
        tempTokenExpiresAt: readNumber(state.tempTokenExpiresAt),
      },
      version: readNumber(parsed.version) ?? 0,
    };
  } catch {
    return { state: createEmptySession(), version: 0 };
  }
}

function writePersistedAuth(session: AuthSession) {
  const storage = getLocalStorage();
  if (storage === null) {
    return;
  }

  const payload: PersistedAuthStorage = {
    state: session,
    version: 0,
  };

  storage.setItem(AUTH_STORAGE_KEY, JSON.stringify(payload));
}

/**
 * 读取本地认证会话。
 */
export function getStoredAuthSession(): AuthSession {
  return readPersistedAuth().state;
}

/**
 * 写入完整本地认证会话。
 */
export function writeStoredAuthSession(session: AuthSession) {
  writePersistedAuth(session);
}

/**
 * 写入刷新后的 Token，并保留当前用户信息。
 */
export function writeStoredAuthTokens(tokens: AuthTokens) {
  const current = getStoredAuthSession();
  writeStoredAuthSession({
    ...current,
    accessToken: tokens.accessToken,
    refreshToken: tokens.refreshToken,
    tokenType: tokens.tokenType,
    expiresAt: Date.now() + tokens.expiresIn * 1000,
  });
}

/**
 * 清理认证会话；传入 message 时会保存登录页提示。
 */
export function clearAuthSession(message?: string) {
  const storage = getLocalStorage();
  if (storage !== null) {
    storage.removeItem(AUTH_STORAGE_KEY);
  }

  const sessionStorage = getSessionStorage();
  if (sessionStorage !== null && message !== undefined) {
    sessionStorage.setItem(AUTH_REDIRECT_MESSAGE_KEY, message);
  }
}

/**
 * 读取并消费认证跳转提示。
 */
export function consumeAuthRedirectMessage() {
  const storage = getSessionStorage();
  if (storage === null) {
    return null;
  }

  const message = storage.getItem(AUTH_REDIRECT_MESSAGE_KEY);
  storage.removeItem(AUTH_REDIRECT_MESSAGE_KEY);
  return message;
}
