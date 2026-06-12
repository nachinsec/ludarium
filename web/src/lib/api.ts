import type {
  AISettings,
  BingoBoard,
  BingoData,
  FeedItem,
  GameStatus,
  IGDBGame,
  LibraryGame,
  MeResponse,
  PublicProfile,
  Recommendation,
  ShowcaseEvent,
  Stats,
  User,
  UserCard,
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

  updateProfile: (input: {
    displayName: string;
    avatarUrl: string;
    visibility: "private" | "public";
  }) =>
    request<{ user: User }>("/api/auth/profile", {
      method: "PATCH",
      body: JSON.stringify(input),
    }),

  disconnectSteam: () =>
    request<{ status: string }>("/api/auth/connections/steam", { method: "DELETE" }),

  connectPSN: (npsso: string) =>
    request<{ status: string }>("/api/auth/connections/psn", {
      method: "POST",
      body: JSON.stringify({ npsso }),
    }),

  disconnectPSN: () =>
    request<{ status: string }>("/api/auth/connections/psn", { method: "DELETE" }),

  syncPSN: () =>
    request<{ total: number; added: number }>("/api/sync/psn", { method: "POST" }),

  getAISettings: () => request<AISettings>("/api/auth/ai"),

  setAISettings: (input: { baseUrl: string; model: string; apiKey: string }) =>
    request<{ status: string }>("/api/auth/ai", {
      method: "PATCH",
      body: JSON.stringify(input),
    }),

  library: () => request<{ games: LibraryGame[] }>("/api/library"),

  game: (id: number) => request<{ game: LibraryGame }>(`/api/library/${id}`),

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

  showcases: () => request<{ events: ShowcaseEvent[] }>("/api/showcases"),

  showcaseGames: (id: number) => request<{ games: IGDBGame[] }>(`/api/showcases/${id}`),

  calendar: () => request<{ games: IGDBGame[] }>("/api/calendar"),

  igdbGame: (id: number) => request<{ game: IGDBGame }>(`/api/igdb/game/${id}`),

  // --- bingo ---
  bingoBoards: () => request<{ boards: BingoBoard[] }>("/api/bingo"),

  bingoBoard: (id: number) => request<{ board: BingoBoard }>(`/api/bingo/${id}`),

  createBingo: (input: { title: string; data: BingoData }) =>
    request<{ board: BingoBoard }>("/api/bingo", { method: "POST", body: JSON.stringify(input) }),

  updateBingo: (id: number, input: { title: string; data: BingoData; visibility: string }) =>
    request<{ ok: boolean }>(`/api/bingo/${id}`, { method: "PATCH", body: JSON.stringify(input) }),

  deleteBingo: (id: number) =>
    request<{ ok: boolean }>(`/api/bingo/${id}`, { method: "DELETE" }),

  oracleCandidates: () => request<{ games: IGDBGame[] }>("/api/oracle/candidates"),

  addGame: (igdbId: number, status = "wishlist", platform = "") =>
    request<{ status: string }>("/api/library/add", {
      method: "POST",
      body: JSON.stringify({ igdbId, status, platform }),
    }),

  addManual: (input: { title: string; status: string; platform: string; coverUrl?: string }) =>
    request<{ status: string }>("/api/library/manual", {
      method: "POST",
      body: JSON.stringify(input),
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

  // --- social ---
  feed: () => request<{ items: FeedItem[] }>("/api/feed"),

  users: () => request<{ users: UserCard[] }>("/api/users"),

  profile: (id: number) => request<{ profile: PublicProfile }>(`/api/users/${id}`),

  userLibrary: (id: number) => request<{ games: LibraryGame[] }>(`/api/users/${id}/library`),

  userStats: (id: number) => request<{ stats: Stats }>(`/api/users/${id}/stats`),

  follow: (id: number) =>
    request<{ ok: boolean }>(`/api/users/${id}/follow`, { method: "POST" }),

  unfollow: (id: number) =>
    request<{ ok: boolean }>(`/api/users/${id}/follow`, { method: "DELETE" }),

  steamLoginUrl: "/api/auth/steam/login",
};
