import { useState } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import type { IGDBGame } from "../lib/types";
import { api } from "../lib/api";
import { DiscoverCard, type AddState } from "../components/DiscoverCard";
import { PixelButton } from "../components/PixelButton";
import styles from "./Discover.module.css";

interface Card {
  igdbId: number;
  title: string;
  coverUrl: string;
  subtitle?: string;
}

const meta = (year: number | null, genres: string[] | null) =>
  [year, (genres ?? []).slice(0, 2).join(" · ")].filter(Boolean).join(" · ");

const fromGame = (g: IGDBGame): Card => ({
  igdbId: g.igdbId,
  title: g.name,
  coverUrl: g.coverUrl,
  subtitle: meta(g.releaseYear, g.genres),
});

export function Discover() {
  const qc = useQueryClient();
  const [added, setAdded] = useState<Set<number>>(new Set());
  const [addingId, setAddingId] = useState<number | null>(null);
  const [term, setTerm] = useState("");
  const [submitted, setSubmitted] = useState("");

  const popular = useQuery({ queryKey: ["igdb-popular"], queryFn: api.popularGames, staleTime: 3.6e6 });
  const upcoming = useQuery({ queryKey: ["igdb-upcoming"], queryFn: api.upcomingGames, staleTime: 3.6e6 });
  const search = useQuery({
    queryKey: ["igdb-search", submitted],
    queryFn: () => api.searchGames(submitted),
    enabled: submitted.length > 0,
  });

  async function handleAdd(igdbId: number) {
    setAddingId(igdbId);
    try {
      await api.addGame(igdbId);
      setAdded((s) => new Set(s).add(igdbId));
      qc.invalidateQueries({ queryKey: ["library"] });
    } finally {
      setAddingId(null);
    }
  }

  const stateOf = (id: number): AddState =>
    added.has(id) ? "added" : addingId === id ? "adding" : "idle";

  const grid = (cards: Card[]) => (
    <div className={styles.grid}>
      {cards
        .filter((c) => c.igdbId > 0)
        .map((c) => (
          <DiscoverCard
            key={c.igdbId}
            title={c.title}
            coverUrl={c.coverUrl}
            subtitle={c.subtitle}
            state={stateOf(c.igdbId)}
            onAdd={() => handleAdd(c.igdbId)}
          />
        ))}
    </div>
  );

  return (
    <section>
      <h2 className={styles.heading}>Discover</h2>

      <form
        className={styles.searchRow}
        onSubmit={(e) => {
          e.preventDefault();
          setSubmitted(term.trim());
        }}
      >
        <input
          className={styles.search}
          placeholder="🔍 search any game…"
          value={term}
          onChange={(e) => setTerm(e.target.value)}
        />
        <PixelButton type="submit">Search</PixelButton>
      </form>
      {submitted && search.data && grid(search.data.games.map(fromGame))}

      <h3 className={styles.section}>🔥 Trending</h3>
      {popular.data ? grid(popular.data.games.map(fromGame)) : <p className={styles.muted}>…</p>}

      <h3 className={styles.section}>📅 Upcoming</h3>
      {upcoming.data ? grid(upcoming.data.games.map(fromGame)) : <p className={styles.muted}>…</p>}
    </section>
  );
}
