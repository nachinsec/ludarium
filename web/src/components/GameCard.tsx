import type { LibraryGame } from "../lib/types";
import { PixelCard } from "./PixelCard";
import { StatusBadge } from "./StatusBadge";
import { StarRating } from "./StarRating";
import { ProgressBlocks } from "./ProgressBlocks";
import styles from "./GameCard.module.css";

/** Deterministic accent color from the title, for the cover placeholder. */
function coverHue(title: string): number {
  let h = 0;
  for (let i = 0; i < title.length; i++) h = (h * 31 + title.charCodeAt(i)) % 360;
  return h;
}

export function GameCard({ game, onClick }: { game: LibraryGame; onClick?: () => void }) {
  const hue = coverHue(game.title);
  return (
    <PixelCard className={styles.card} onClick={onClick}>
      <div
        className={styles.cover}
        style={{ background: `linear-gradient(135deg, hsl(${hue} 50% 22%), hsl(${hue} 60% 12%))` }}
      >
        <span className={styles.coverGlyph}>▚</span>
        {game.coverUrl && (
          <img
            className={styles.coverImg}
            src={game.coverUrl}
            alt=""
            loading="lazy"
            // Hide on 404 so the gradient placeholder shows through.
            onError={(e) => {
              e.currentTarget.style.display = "none";
            }}
          />
        )}
        <div className={styles.badge}>
          <StatusBadge status={game.status} />
        </div>
      </div>

      <div className={styles.body}>
        <h3 className={styles.title}>{game.title}</h3>
        <div className={styles.meta}>
          {game.developer && <span>{game.developer}</span>}
          {game.releaseYear && <span>· {game.releaseYear}</span>}
        </div>

        <div className={styles.statsRow}>
          <span className={styles.hours}>⏱ {game.hours.toFixed(1)}h</span>
          <StarRating value={game.rating} />
        </div>

        {game.progress !== undefined && (
          <div className={styles.progress}>
            <ProgressBlocks percent={game.progress} />
            <span className={styles.pct}>{game.progress}%</span>
          </div>
        )}
      </div>
    </PixelCard>
  );
}
