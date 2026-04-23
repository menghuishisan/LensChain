// api-client.ts
// 统一 HTTP 客户端：读取环境 baseURL、注入 Token、解包响应、归一错误并处理 401。

import type { ApiError, ApiResponse, QueryParams, QueryValue } from "@/types/api";
import type { AuthTokens } from "@/types/auth";
import { clearAuthSession, getStoredAuthSession, writeStoredAuthTokens } from "@/lib/auth-session";

const API_BASE_URL = process.env.NEXT_PUBLIC_API_BASE_URL ?? "";
const REFRESH_THRESHOLD_MS = 5 * 60 * 1000;

type HttpMethod = "GET" | "POST" | "PUT" | "PATCH" | "DELETE";
type JsonRequestBody = object | readonly unknown[];
type ApiRequestBody = BodyInit | JsonRequestBody | null | undefined;

/**
 * API 请求配置。
 */
export interface ApiRequestOptions {
  method?: HttpMethod;
  query?: QueryParams;
  body?: ApiRequestBody;
  headers?: HeadersInit;
  auth?: boolean;
  signal?: AbortSignal;
}

/**
 * 文件下载结果。
 */
export interface ApiDownloadResult {
  blob: Blob;
  filename: string | null;
}

/**
 * API 客户端创建选项。
 */
export interface ApiClientOptions {
  baseURL?: string;
  fetcher?: typeof fetch;
  onUnauthorized?: (error: ApiClientError) => void;
}

/**
 * 上传请求配置。
 */
export interface ApiUploadOptions {
  auth?: boolean;
  query?: QueryParams;
  onUploadProgress?: (progress: number) => void;
}

interface RefreshTokenResponse {
  access_token: string;
  refresh_token: string;
  expires_in: number;
  token_type?: string;
}

/**
 * API 客户端错误类。
 */
export class ApiClientError extends Error implements ApiError {
  code: number;
  status?: number;
  data?: unknown;
  timestamp?: string | number;

  constructor(error: ApiError) {
    super(error.message);
    this.name = "ApiClientError";
    this.code = error.code;
    this.status = error.status;
    this.data = error.data;
    this.timestamp = error.timestamp;
  }
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null;
}

function isApiResponse(value: unknown): value is ApiResponse<unknown> {
  return (
    isRecord(value) &&
    typeof value.code === "number" &&
    typeof value.message === "string" &&
    "data" in value
  );
}

function appendQuery(searchParams: URLSearchParams, key: string, value: QueryValue) {
  if (value === null || value === undefined) {
    return;
  }

  searchParams.append(key, String(value));
}

function isQueryValueArray(value: QueryValue | readonly QueryValue[]): value is readonly QueryValue[] {
  return Array.isArray(value);
}

function buildURL(baseURL: string, path: string, query?: QueryParams) {
  const normalizedBase = baseURL.replace(/\/$/, "");
  const normalizedPath = path.startsWith("/") ? path : `/${path}`;
  const url = `${normalizedBase}${normalizedPath}`;

  if (query === undefined) {
    return url;
  }

  const searchParams = new URLSearchParams();
  Object.entries(query).forEach(([key, value]) => {
    if (isQueryValueArray(value)) {
      value.forEach((item) => appendQuery(searchParams, key, item));
      return;
    }

    appendQuery(searchParams, key, value);
  });

  const queryString = searchParams.toString();
  return queryString.length > 0 ? `${url}?${queryString}` : url;
}

function extractFilename(contentDisposition: string | null) {
  if (contentDisposition === null) {
    return null;
  }

  const utf8Match = /filename\*=UTF-8''([^;]+)/i.exec(contentDisposition);
  if (utf8Match?.[1] !== undefined) {
    return decodeURIComponent(utf8Match[1]);
  }

  const plainMatch = /filename="?([^"]+)"?/i.exec(contentDisposition);
  return plainMatch?.[1] ?? null;
}

function isNativeBody(body: ApiRequestBody): body is BodyInit {
  return (
    body instanceof FormData ||
    body instanceof Blob ||
    body instanceof URLSearchParams ||
    body instanceof ArrayBuffer ||
    ArrayBuffer.isView(body)
  );
}

function createRequestBody(body: ApiRequestBody, headers: Headers): BodyInit | undefined {
  if (body === null || body === undefined) {
    return undefined;
  }

  if (isNativeBody(body)) {
    return body;
  }

  if (!headers.has("Content-Type")) {
    headers.set("Content-Type", "application/json");
  }

  return JSON.stringify(body);
}

function shouldHandleUnauthorized(error: ApiClientError) {
  return error.status === 401 || (error.code >= 40100 && error.code < 40200);
}

function defaultUnauthorizedHandler(error: ApiClientError) {
  if (typeof window === "undefined") {
    return;
  }

  const currentPath = `${window.location.pathname}${window.location.search}`;
  const loginPath = `/login?redirect=${encodeURIComponent(currentPath)}`;
  clearAuthSession(error.message);

  if (window.location.pathname !== "/login") {
    window.location.assign(loginPath);
  }
}

function toAuthTokens(data: RefreshTokenResponse): AuthTokens {
  return {
    accessToken: data.access_token,
    refreshToken: data.refresh_token,
    tokenType: data.token_type ?? "Bearer",
    expiresIn: data.expires_in,
  };
}

/**
 * API 客户端，所有 services 必须通过此客户端访问后端。
 */
export class ApiClient {
  private readonly baseURL: string;
  private readonly fetcher: typeof fetch;
  private readonly onUnauthorized?: (error: ApiClientError) => void;
  private refreshPromise: Promise<void> | null = null;

  constructor(options: ApiClientOptions = {}) {
    this.baseURL = options.baseURL ?? API_BASE_URL;
    this.fetcher = options.fetcher ?? fetch.bind(globalThis);
    this.onUnauthorized = options.onUnauthorized;
  }

  /**
   * 发起 GET 请求。
   */
  get<TData>(path: string, options: Omit<ApiRequestOptions, "method" | "body"> = {}) {
    return this.request<TData>(path, { ...options, method: "GET" });
  }

  /**
   * 发起 POST 请求。
   */
  post<TData>(path: string, body?: ApiRequestBody, options: Omit<ApiRequestOptions, "method" | "body"> = {}) {
    return this.request<TData>(path, { ...options, method: "POST", body });
  }

  /**
   * 发起 PUT 请求。
   */
  put<TData>(path: string, body?: ApiRequestBody, options: Omit<ApiRequestOptions, "method" | "body"> = {}) {
    return this.request<TData>(path, { ...options, method: "PUT", body });
  }

  /**
   * 发起 PATCH 请求。
   */
  patch<TData>(path: string, body?: ApiRequestBody, options: Omit<ApiRequestOptions, "method" | "body"> = {}) {
    return this.request<TData>(path, { ...options, method: "PATCH", body });
  }

  /**
   * 发起 DELETE 请求。
   */
  delete<TData>(path: string, options: Omit<ApiRequestOptions, "method" | "body"> = {}) {
    return this.request<TData>(path, { ...options, method: "DELETE" });
  }

  /**
   * 创建完整 API URL，用于 SSO 302 跳转等浏览器直达场景。
   */
  createURL(path: string, query?: QueryParams) {
    return buildURL(this.baseURL, path, query);
  }

  /**
   * 下载文件响应，仍统一处理 Token 注入、刷新和错误 JSON。
   */
  async download(path: string, options: Omit<ApiRequestOptions, "method" | "body"> = {}): Promise<ApiDownloadResult> {
    const shouldUseAuth = options.auth !== false;
    if (shouldUseAuth) {
      await this.refreshTokenIfNeeded(path);
    }

    const headers = new Headers(options.headers);
    const session = getStoredAuthSession();
    if (shouldUseAuth && session.accessToken !== null) {
      headers.set("Authorization", `${session.tokenType} ${session.accessToken}`);
    }

    const response = await this.fetcher(buildURL(this.baseURL, path, options.query), {
      method: "GET",
      headers,
      signal: options.signal,
    });

    const contentType = response.headers.get("Content-Type") ?? "";
    if (!response.ok || contentType.includes("application/json")) {
      await this.unwrapResponse<unknown>(response);
    }

    return {
      blob: await response.blob(),
      filename: extractFilename(response.headers.get("Content-Disposition")),
    };
  }

  /**
   * 上传 multipart/form-data 文件并返回解包后的 data。
   * 使用 XMLHttpRequest 是为了读取真实 upload progress；该能力仍集中在 api-client 内部。
   */
  async upload<TData>(path: string, formData: FormData, options: ApiUploadOptions = {}) {
    const shouldUseAuth = options.auth !== false;
    if (shouldUseAuth) {
      await this.refreshTokenIfNeeded(path);
    }

    return new Promise<TData>((resolve, reject) => {
      const xhr = new XMLHttpRequest();
      xhr.open("POST", buildURL(this.baseURL, path, options.query));

      const session = getStoredAuthSession();
      if (shouldUseAuth && session.accessToken !== null) {
        xhr.setRequestHeader("Authorization", `${session.tokenType} ${session.accessToken}`);
      }

      xhr.upload.onprogress = (event) => {
        if (event.lengthComputable && options.onUploadProgress !== undefined) {
          options.onUploadProgress(Math.round((event.loaded / event.total) * 100));
        }
      };

      xhr.onerror = () => reject(new ApiClientError({ code: 0, message: "文件上传网络错误" }));
      xhr.onload = () => {
        let payload: unknown;
        try {
          payload = JSON.parse(xhr.responseText) as unknown;
        } catch {
          reject(new ApiClientError({ code: xhr.status, status: xhr.status, message: "上传响应解析失败" }));
          return;
        }

        if (!isApiResponse(payload)) {
          reject(new ApiClientError({ code: xhr.status, status: xhr.status, message: "后端响应格式不正确", data: payload }));
          return;
        }

        if (xhr.status < 200 || xhr.status >= 300 || payload.code !== 200) {
          const error = new ApiClientError({
            code: payload.code,
            status: xhr.status,
            message: payload.message,
            data: payload.data,
            timestamp: payload.timestamp,
          });
          if (shouldHandleUnauthorized(error)) {
            clearAuthSession(error.message);
            if (this.onUnauthorized !== undefined) {
              this.onUnauthorized(error);
            } else {
              defaultUnauthorizedHandler(error);
            }
          }
          reject(error);
          return;
        }

        resolve(payload.data as TData);
      };

      xhr.send(formData);
    });
  }

  /**
   * 发起任意 HTTP 请求并返回解包后的 data。
   */
  async request<TData>(path: string, options: ApiRequestOptions = {}) {
    const method = options.method ?? "GET";
    const shouldUseAuth = options.auth !== false;

    if (shouldUseAuth) {
      await this.refreshTokenIfNeeded(path);
    }

    const headers = new Headers(options.headers);
    const session = getStoredAuthSession();

    if (shouldUseAuth && session.accessToken !== null) {
      headers.set("Authorization", `${session.tokenType} ${session.accessToken}`);
    }

    const body = createRequestBody(options.body, headers);
    const response = await this.fetcher(buildURL(this.baseURL, path, options.query), {
      method,
      headers,
      body,
      signal: options.signal,
    });

    return this.unwrapResponse<TData>(response);
  }

  private async refreshTokenIfNeeded(path: string) {
    if (path === "/auth/token/refresh") {
      return;
    }

    const session = getStoredAuthSession();
    if (session.refreshToken === null || session.expiresAt === null) {
      return;
    }

    if (session.expiresAt - Date.now() > REFRESH_THRESHOLD_MS) {
      return;
    }

    if (this.refreshPromise === null) {
      this.refreshPromise = this.refreshToken(session.refreshToken).finally(() => {
        this.refreshPromise = null;
      });
    }

    await this.refreshPromise;
  }

  private async refreshToken(refreshToken: string) {
    const data = await this.request<RefreshTokenResponse>("/auth/token/refresh", {
      method: "POST",
      auth: false,
      body: { refresh_token: refreshToken },
    });

    writeStoredAuthTokens(toAuthTokens(data));
  }

  private async unwrapResponse<TData>(response: Response) {
    const payload = await this.parseResponsePayload(response);
    if (!isApiResponse(payload)) {
      throw new ApiClientError({
        code: response.status,
        status: response.status,
        message: "后端响应格式不正确",
        data: payload,
      });
    }

    if (!response.ok || payload.code !== 200) {
      const error = new ApiClientError({
        code: payload.code,
        status: response.status,
        message: payload.message,
        data: payload.data,
        timestamp: payload.timestamp,
      });

      if (shouldHandleUnauthorized(error)) {
        clearAuthSession(error.message);
        if (this.onUnauthorized !== undefined) {
          this.onUnauthorized(error);
        } else {
          defaultUnauthorizedHandler(error);
        }
      }

      throw error;
    }

    // 后端统一响应结构由 API 文档约束，泛型 TData 由 service 层按接口文档指定。
    return payload.data as TData;
  }

  private async parseResponsePayload(response: Response) {
    const contentType = response.headers.get("Content-Type") ?? "";
    if (!contentType.includes("application/json")) {
      return {
        code: response.ok ? 200 : response.status,
        message: response.statusText || "HTTP请求失败",
        data: null,
      };
    }

    try {
      return (await response.json()) as unknown;
    } catch {
      return {
        code: response.ok ? 200 : response.status,
        message: "响应 JSON 解析失败",
        data: null,
      };
    }
  }
}

/**
 * 创建 API 客户端实例，测试或特殊场景可注入 fetcher。
 */
export function createApiClient(options: ApiClientOptions = {}) {
  return new ApiClient(options);
}

/**
 * 全局默认 API 客户端。
 */
export const apiClient = createApiClient();
