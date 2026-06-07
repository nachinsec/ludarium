import { useState } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import type { IGDBGame } from "../lib/types";
import { api } from "../lib/api";
import { PixelButton } from "../components/PixelButton";
import styles from "./QuickAdd.module.css";

const PLATFORMS = ["PC", "PlayStation", "Xbox", "Switch", "Nintendo", "Retro", "Mobile", "Other"];
const STATUSES = ["backlog", "playing", "ongoing", "completed", "dropped", "wishlist"];

export function QuickAdd() {
  const qc = useQueryClient();
  const [platform, setPlatform] = useState("PC");
  const [status, setStatus] = useState("backlog");
  const [term, setTerm] = useState("");
  const [submitted, setSubmitted] = useState("");
  const [added, setAdded] = useState<Set<number>>(new Set());
  const [addingId, setAddingId] = useState<number | null>(null);

  const search = useQuery({
    queryKey: ["igdb-search", submitted],
    queryFn: () => api.searchGames(submitted),
    enabled: submitted.length > 0,
  });

  async function add(g: IGDBGame) {
    setAddingId(g.igdbId);
    try {
      await api.addGame(g.igdbId, status, platform);
      setAdded((s) => new Set(s).add(g.igdbId));
      qc.invalidateQueries({ queryKey: ["library"] });
    } finally {
      setAddingId(null);
    }
  }

  const meta = (g: IGDBGame) =>
    [g.releaseYear, (g.genres ?? []).slice(0, 2).join(" · ")].filter(Boolean).join("  ·  ");

  return (
    <section className={styles.page}>
      <h2 className={styles.heading}>Add games</h2>
      <p className={styles.intro}>
        Set the platform once, then search and add as many as you like — any platform, no API needed.
      </p>

      <div className={styles.controls}>
        <label className={styles.ctl}>
          <span>Platform</span>
          <select value={platform} onChange={(e) => setPlatform(e.target.value)}>
            {PLATFORMS.map((p) => (
              <option key={p}>{p}</option>
            ))}
          </select>
        </label>
        <label className={styles.ctl}>
          <span>Status</span>
          <select value={status} onChange={(e) => setStatus(e.target.value)}>
            {STATUSES.map((s) => (
              <option key={s} value={s}>
                {s}
              </option>
            ))}
          </select>
        </label>
      </div>

      <form
        className={styles.searchRow}
        onSubmit={(e) => {
          e.preventDefault();
          setSubmitted(term.trim());
        }}
      >
        <input
          className={styles.search}
          placeholder="🔍 search a game…"
          value={term}
          autoFocus
          onChange={(e) => setTerm(e.target.value)}
        />
        <PixelButton type="submit">Search</PixelButton>
      </form>

      <div className={styles.results}>
        {submitted &&
          (search.data?.games ?? []).map((g) => {
            const isAdded = added.has(g.igdbId);
            return (
              <div key={g.igdbId} className={styles.row}>
                <div className={styles.cover}>
                  {g.coverUrl && <img src={g.coverUrl} alt="" loading="lazy" />}
                </div>
                <div className={styles.info}>
                  <span className={styles.title}>{g.name}</span>
                  <span className={styles.metaLine}>{meta(g)}</span>
                </div>
                <PixelButton
                  variant={isAdded ? "default" : "primary"}
                  disabled={isAdded || addingId === g.igdbId}
                  onClick={() => add(g)}
                >
                  {isAdded ? "✓ Added" : addingId === g.igdbId ? "…" : "+ Add"}
                </PixelButton>
              </div>
            );
          })}
        {submitted && search.data && search.data.games.length === 0 && (
          <p className={styles.muted}>No matches.</p>
        )}
      </div>
    </section>
  );
}
