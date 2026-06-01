import { useQuery } from "@tanstack/react-query";
import type { GameStatus } from "../lib/types";
import { api } from "../lib/api";
import { PixelCard } from "../components/PixelCard";
import styles from "./Stats.module.css";

const STATUS_META: Record<GameStatus, { label: string; color: string }> = {
  playing: { label: "Playing", color: "var(--c-red)" },
  ongoing: { label: "Ongoing", color: "var(--c-teal)" },
  completed: { label: "Cleared", color: "var(--c-green)" },
  backlog: { label: "Backlog", color: "var(--c-blue)" },
  dropped: { label: "Dropped", color: "var(--c-grey)" },
  wishlist: { label: "Wishlist", color: "var(--c-purple)" },
};

const fmtHours = (h: number) => Math.round(h).toLocaleString();

export function Stats() {
  const { data, isLoading } = useQuery({ queryKey: ["stats"], queryFn: api.stats });

  if (isLoading || !data) {
    return <p className={styles.loading}>▚ loading stats…</p>;
  }

  const statuses = Object.keys(STATUS_META) as GameStatus[];
  const maxCount = Math.max(1, ...statuses.map((s) => data.byStatus[s]?.count ?? 0));
  const maxHours = Math.max(1, ...data.topGames.map((g) => g.hours));
  const cleared = data.byStatus.completed?.count ?? 0;
  const clearedPct = data.totalGames ? Math.round((cleared / data.totalGames) * 100) : 0;
  const maxGenreHours = Math.max(1, ...data.genres.map((g) => g.hours));

  return (
    <section>
      <h2 className={styles.heading}>Stats</h2>

      <div className={styles.cards}>
        <PixelCard className={styles.card}>
          <span className={styles.num}>{data.totalGames}</span>
          <span className={styles.cap}>games</span>
        </PixelCard>
        <PixelCard className={styles.card}>
          <span className={styles.num}>{fmtHours(data.totalHours)}</span>
          <span className={styles.cap}>hours played</span>
        </PixelCard>
        <PixelCard className={styles.card}>
          <span className={styles.num}>
            {data.ratingCount ? `${data.avgRating.toFixed(1)}★` : "—"}
          </span>
          <span className={styles.cap}>avg · {data.ratingCount} rated</span>
        </PixelCard>
        <PixelCard className={styles.card}>
          <span className={styles.num}>{clearedPct}%</span>
          <span className={styles.cap}>{cleared} cleared</span>
        </PixelCard>
      </div>

      <PixelCard className={styles.panel}>
        <h3 className={styles.panelTitle}>By status</h3>
        {statuses.map((s) => {
          const stat = data.byStatus[s] ?? { count: 0, hours: 0 };
          return (
            <div key={s} className={styles.row}>
              <span className={styles.rowLabel}>{STATUS_META[s].label}</span>
              <div className={styles.track}>
                <div
                  className={styles.fill}
                  style={{
                    width: `${(stat.count / maxCount) * 100}%`,
                    background: STATUS_META[s].color,
                  }}
                />
              </div>
              <span className={styles.rowVal}>
                {stat.count} · {fmtHours(stat.hours)}h
              </span>
            </div>
          );
        })}
      </PixelCard>

      <PixelCard className={styles.panel}>
        <h3 className={styles.panelTitle}>Most played</h3>
        {data.topGames.length === 0 ? (
          <p className={styles.empty}>No games yet.</p>
        ) : (
          data.topGames.map((g) => (
            <div key={g.title} className={styles.row}>
              <span className={styles.rowLabel} title={g.title}>
                {g.title}
              </span>
              <div className={styles.track}>
                <div
                  className={styles.fill}
                  style={{ width: `${(g.hours / maxHours) * 100}%`, background: "var(--c-yellow)" }}
                />
              </div>
              <span className={styles.rowVal}>{g.hours.toFixed(1)}h</span>
            </div>
          ))
        )}
      </PixelCard>

      {data.genres.length > 0 && (
        <PixelCard className={styles.panel}>
          <h3 className={styles.panelTitle}>Top genres</h3>
          {data.genres.map((g) => (
            <div key={g.name} className={styles.row}>
              <span className={styles.rowLabel} title={g.name}>
                {g.name}
              </span>
              <div className={styles.track}>
                <div
                  className={styles.fill}
                  style={{ width: `${(g.hours / maxGenreHours) * 100}%`, background: "var(--c-purple)" }}
                />
              </div>
              <span className={styles.rowVal}>
                {g.count} · {Math.round(g.hours)}h
              </span>
            </div>
          ))}
        </PixelCard>
      )}
    </section>
  );
}
