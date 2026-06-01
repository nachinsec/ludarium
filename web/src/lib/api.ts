import type {
  AISettings,
  GameStatus,
  IGDBGame,
  LibraryGame,
  MeResponse,
  Recommendation,
  Stats,
  User,
} from "./types";

/** Thrown for non-2xx responses, carrying the server's error message. */
export class ApiError extends Error {
  status: number;
  constructor(status: number, message: string) {
    super(message);
    this.status = status;
  }
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(path, {
    credentials: "include",
    headers: { "Content-Type": "application/json" },
    ...init,
  });
  const isJSON = res.headers.get("content-type")?.includes("application/json");
  const body = isJSON ? await res.json() : null;
  if (!res.ok) {
    throw new ApiError(res.status, body?.error ?? `${res.status} ${res.statusText}`);
  }
  return body as T;
}

export const api = {
  me: () => request<MeResponse>("/api/auth/me"),

  register: (input: { username: string; email: string; password: string }) =>
    request<{ user: User }>("/api/auth/register", {
      method: "POST",
      body: JSON.stringify(input),
    }),

  login: (input: { login: string; password: string }) =>
    request<{ user: User }>("/api/auth/login", {
      method: "POST",
      body: JSON.stringify(input),
    }),

  logout: () => request<{ status: string }>("/api/auth/logout", { method: "POST" }),

  updateProfile: (input: { displayName: string; avatarUrl: string }) =>
    request<{ user: User }>("/api/auth/profile", {
      method: "PATCH",
      body: JSON.stringify(input),
    }),

  disconnectSteam: () =>
    request<{ status: string }>("/api/auth/connections/steam", { method: "DELETE" }),

  getAISettings: () => request<AISettings>("/api/auth/ai"),

  setAISettings: (input: { baseUrl: string; model: string; apiKey: string }) =>
    request<{ status: string }>("/api/auth/ai", {
      method: "PATCH",
      body: JSON.stringify(input),
    }),

  library: () => request<{ games: LibraryGame[] }>("/api/library"),

  enrich: () =>
    request<{ checked: number; matched: number }>("/api/enrich", { method: "POST" }),

  stats: () => request<Stats>("/api/stats"),

  recommend: (preferred?: string[]) =>
    request<{ recommendations: Recommendation[] }>("/api/recommend", {
      method: "POST",
      body: JSON.stringify(preferred ? { preferred } : {}),
    }),

  searchGames: (q: string) =>
    request<{ games: IGDBGame[] }>(`/api/igdb/search?q=${encodeURIComponent(q)}`),

  popularGames: () => request<{ games: IGDBGame[] }>("/api/igdb/popular"),

  upcomingGames: () => request<{ games: IGDBGame[] }>("/api/igdb/upcoming"),

  oracleCandidates: () => request<{ games: IGDBGame[] }>("/api/oracle/candidates"),

  addGame: (igdbId: number, status = "wishlist") =>
    request<{ status: string }>("/api/library/add", {
      method: "POST",
      body: JSON.stringify({ igdbId, status }),
    }),

  updateGame: (
    id: number,
    input: { status: GameStatus; rating: number | null; notes: string },
  ) =>
    request<{ game: LibraryGame }>(`/api/library/${id}`, {
      method: "PATCH",
      body: JSON.stringify(input),
    }),

  syncSteam: () =>
    request<{ total: number; added: number }>("/api/sync/steam", { method: "POST" }),

  steamLoginUrl: "/api/auth/steam/login",
};
