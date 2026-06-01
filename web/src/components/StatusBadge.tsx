import type { GameStatus } from "../lib/types";
import styles from "./StatusBadge.module.css";

const LABELS: Record<GameStatus, string> = {
  playing: "PLAYING",
  ongoing: "ONGOING",
  completed: "CLEARED",
  backlog: "BACKLOG",
  dropped: "DROPPED",
  wishlist: "WISHLIST",
};

/** Small colored tag indicating where a game sits in your library. */
export function StatusBadge({ status }: { status: GameStatus }) {
  return <span className={`${styles.badge} ${styles[status]}`}>{LABELS[status]}</span>;
}
