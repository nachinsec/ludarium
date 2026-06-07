export type GameStatus =
  | "playing"
  | "ongoing"
  | "completed"
  | "backlog"
  | "dropped"
  | "wishlist";

export interface User {
  id: number;
  username: string;
  displayName: string;
  email: string;
  avatarUrl: string;
  role: string;
  createdAt: string;
  hasPassword: boolean;
  visibility: "private" | "public";
}

export interface UserCard {
  id: number;
  username: string;
  displayName: string;
  avatarUrl: string;
  visibility: "private" | "public";
  gameCount: number;
  isFollowing: boolean;
}

export interface PublicProfile extends UserCard {
  followers: number;
  following: number;
  isSelf: boolean;
  canView: boolean;
}

export interface FeedItem {
  userId: number;
  displayName: string;
  avatarUrl: string;
  title: string;
  coverUrl: string;
  status: GameStatus;
  at: string;
}

export interface Connection {
  id: number;
  provider: string;
  externalId: string;
  createdAt: string;
}

export interface AISettings {
  baseUrl: string;
  model: string;
  hasKey: boolean;
}

export interface MeResponse {
  user: User | null;
  connections?: Connection[];
}

export interface StatusStat {
  count: number;
  hours: number;
}

export interface TopGame {
  title: string;
  hours: number;
}

export interface GenreStat {
  name: string;
  count: number;
  hours: number;
}

export interface Stats {
  totalGames: number;
  totalHours: number;
  byStatus: Record<GameStatus, StatusStat>;
  avgRating: number;
  ratingCount: number;
  topGames: TopGame[];
  genres: GenreStat[];
}

export interface Recommendation {
  title: string;
  reason: string;
  coverUrl: string;
  igdbId: number;
  releaseYear: number | null;
  genres: string[] | null;
}

export interface IGDBGame {
  igdbId: number;
  name: string;
  coverUrl: string;
  releaseYear: number | null;
  genres: string[] | null;
  developer: string;
  summary: string;
}

export interface LibraryGame {
  id: number;
  title: string;
  coverUrl: string;
  status: GameStatus;
  rating: number | null; // 1..5
  hours: number;
  platform: string;
  developer: string;
  releaseYear: number | null;
  notes: string;
  genres?: string[] | null;
  startedAt?: string | null;
  finishedAt?: string | null;
  summary?: string;
  screenshots?: string[] | null;
  score?: number; // IGDB community rating 0..100, 0 = none
  /** 0..100 completion estimate (optional, used for the progress bar). */
  progress?: number;
}
