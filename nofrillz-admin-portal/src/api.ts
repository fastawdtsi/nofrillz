import type {
  AIAccountForm,
  AIStatus,
  AIAccountRecord,
  AIPostResult,
  AdminUser,
  AdminPostsResponse,
  AdminUsersResponse,
  ConnectionSettings,
  Stats,
} from "./types";

type RequestOptions = {
  method?: string;
  body?: unknown;
};

export class ApiError extends Error {
  status: number;

  constructor(message: string, status: number) {
    super(message);
    this.name = "ApiError";
    this.status = status;
  }
}

function normalizeBaseUrl(baseUrl: string) {
  const trimmed = baseUrl.trim();
  if (trimmed === "") {
    return "";
  }

  const withProtocol = /^https?:\/\//i.test(trimmed) ? trimmed : `http://${trimmed}`;
  return withProtocol.endsWith("/") ? withProtocol : `${withProtocol}/`;
}

async function request<T>(
  settings: ConnectionSettings,
  path: string,
  options: RequestOptions = {},
) {
  const baseUrl = normalizeBaseUrl(settings.baseUrl);
  if (baseUrl === "") {
    throw new ApiError("Missing API base URL", 0);
  }

  const response = await fetch(new URL(path.replace(/^\//, ""), baseUrl), {
    method: options.method ?? "GET",
    headers: {
      Accept: "application/json",
      "Content-Type": "application/json",
      "X-Admin-API-Key": settings.adminApiKey.trim(),
    },
    body: options.body === undefined ? undefined : JSON.stringify(options.body),
  });

  const text = await response.text();
  if (!response.ok) {
    throw new ApiError(text || `Request failed with status ${response.status}`, response.status);
  }

  if (text === "") {
    return undefined as T;
  }

  return JSON.parse(text) as T;
}

export function getStats(settings: ConnectionSettings) {
  return request<Stats>(settings, "/admin/stats");
}

export function listUsers(
  settings: ConnectionSettings,
  params: { q?: string; accountType?: string; limit?: number; cursor?: string } = {},
) {
  const searchParams = new URLSearchParams();
  if (params.q) searchParams.set("q", params.q);
  if (params.accountType) searchParams.set("account_type", params.accountType);
  if (params.limit) searchParams.set("limit", String(params.limit));
  if (params.cursor) searchParams.set("cursor", params.cursor);

  const suffix = searchParams.toString();
  return request<AdminUsersResponse>(settings, `/admin/users${suffix ? `?${suffix}` : ""}`);
}

export function blockUser(settings: ConnectionSettings, userId: string) {
  return request<AdminUser>(settings, `/admin/users/${userId}/block`, {
    method: "POST",
  });
}

export function listPosts(
  settings: ConnectionSettings,
  params: { userId?: string; limit?: number; cursor?: string } = {},
) {
  const searchParams = new URLSearchParams();
  if (params.userId) searchParams.set("user_id", String(params.userId));
  if (params.limit) searchParams.set("limit", String(params.limit));
  if (params.cursor) searchParams.set("cursor", params.cursor);

  const suffix = searchParams.toString();
  return request<AdminPostsResponse>(settings, `/admin/posts${suffix ? `?${suffix}` : ""}`);
}

export function deletePost(settings: ConnectionSettings, postId: string) {
  return request<void>(settings, `/admin/posts/${postId}`, {
    method: "DELETE",
  });
}

export function createAIAccount(settings: ConnectionSettings, payload: AIAccountForm) {
  return request<AIAccountRecord>(settings, "/admin/ai/accounts", {
    method: "POST",
    body: payload,
  });
}

export function createAIPost(settings: ConnectionSettings, accountId: string, body: string) {
  return request<AIPostResult>(settings, `/admin/ai/accounts/${accountId}/posts`, {
    method: "POST",
    body: { body },
  });
}

export function getAIAccount(settings: ConnectionSettings, id: string) {
  return request<AIAccountRecord>(settings, `/admin/ai/accounts/${id}`);
}
export function updateAIAccount(settings: ConnectionSettings, id: string, payload: Partial<Omit<AIAccountForm, "email" | "username">>) {
  return request<AIAccountRecord>(settings, `/admin/ai/accounts/${id}`, { method: "PATCH", body: payload });
}
export function getAIStatus(settings: ConnectionSettings) {
  return request<AIStatus>(settings, "/admin/ai/status");
}
export function previewAIPost(settings: ConnectionSettings, id: string) {
  return request<{ generation: AIPostResult["generation"] }>(settings, `/admin/ai/accounts/${id}/preview`, { method: "POST" });
}
