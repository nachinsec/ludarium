import { Link } from "react-router-dom";
import type { IGDBGame } from "../lib/types";
import styles from "./IgdbExtras.module.css";

// Live IGDB detail extras, split so each page can place them in the canonical
// order: trailer before screenshots, store links + similar games at the end.

export function IgdbTrailer({ game }: { game: IGDBGame }) {
  const videos = game.videos ?? [];
  const trailer = videos.find((v) => /trailer/i.test(v.name)) ?? videos[0];
  if (!trailer) return null;
  return (
    <div className={styles.trailer}>
      <iframe
        src={`https://www.youtube-nocookie.com/embed/${trailer.youtubeId}`}
        title={trailer.name || "Trailer"}
        allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture"
        allowFullScreen
      />
    </div>
  );
}

// Placeholders shown while the live IGDB fetch is in flight, so the trailer and
// similar games fade in without a sudden layout jump.
export function IgdbTrailerSkeleton() {
  return <div className={styles.trailerSkel} />;
}

export function IgdbLinksSkeleton() {
  return (
    <>
      <label className={styles.label}>Similar games</label>
      <div className={styles.similar}>
        {Array.from({ length: 6 }).map((_, i) => (
          <div key={i} className={styles.simCard}>
            <div className={`${styles.simCover} ${styles.skeleton}`} />
            <span className={styles.simNameSkel} />
          </div>
        ))}
      </div>
    </>
  );
}

export function IgdbLinks({ game }: { game: IGDBGame }) {
  const links = game.links ?? [];
  const similar = (game.similar ?? []).filter((s) => s.coverUrl);
  if (links.length === 0 && similar.length === 0) return null;

  return (
    <>
      {links.length > 0 && (
        <>
          <label className={styles.label}>Where to get it</label>
          <div className={styles.stores}>
            {links.map((l) => (
              <a key={l.url} className={styles.store} href={l.url} target="_blank" rel="noreferrer">
                {l.store} ↗
              </a>
            ))}
          </div>
        </>
      )}

      {similar.length > 0 && (
        <>
          <label className={styles.label}>Similar games</label>
          <div className={styles.similar}>
            {similar.map((s) => (
              <Link key={s.igdbId} to={`/igdb/${s.igdbId}`} className={styles.simCard} title={s.name}>
                <div className={styles.simCover}>
                  <img src={s.coverUrl} alt="" loading="lazy" />
                </div>
                <span className={styles.simName}>{s.name}</span>
              </Link>
            ))}
          </div>
        </>
      )}
    </>
  );
}
