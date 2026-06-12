import { useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import type { IGDBGame, ShowcaseEvent } from "../lib/types";
import { api } from "../lib/api";
import { DiscoverCard, type AddState } from "../components/DiscoverCard";
import { PixelCard } from "../components/PixelCard";
import styles from "./Showcases.module.css";

const fmtDate = (unix: number) =>
  new Date(unix * 1000).toLocaleDateString("en-GB", { day: "numeric", month: "short", year: "numeric" });

const subtitle = (g: IGDBGame) => {
  const plats = (g.platforms ?? []).slice(0, 4).join(" · ");
  return [plats || "Platform TBA", g.releaseYear].filter(Boolean).join("  ·  ");
};

export function Showcases() {
  const qc = useQueryClient();
  const navigate = useNavigate();
  const [selected, setSelected] = useState<number | null>(null);
  const [added, setAdded] = useState<Set<number>>(new Set());
  const [addingId, setAddingId] = useState<number | null>(null);

  const events = useQuery({ queryKey: ["showcases"], queryFn: api.showcases, staleTime: 3.6e6 });

  // Auto-select the most recent showcase once loaded.
  useEffect(() => {
    if (selected === null && events.data?.events.length) {
      setSelected(events.data.events[0].id);
    }
  }, [events.data, selected]);

  const games = useQuery({
    queryKey: ["showcase-games", selected],
    queryFn: () => api.showcaseGames(selected!),
    enabled: selected !== null,
    staleTime: 3.6e6,
  });

  async function add(igdbId: number) {
    setAddingId(igdbId);
    try {
      await api.addGame(igdbId, "wishlist");
      setAdded((s) => new Set(s).add(igdbId));
      qc.invalidateQueries({ queryKey: ["library"] });
    } finally {
      setAddingId(null);
    }
  }
  const stateOf = (id: number): AddState =>
    added.has(id) ? "added" : addingId === id ? "adding" : "idle";

  if (events.isLoading) return <p className={styles.muted}>▚ loading showcases…</p>;
  const list = events.data?.events ?? [];
  if (list.length === 0) return <p className={styles.muted}>No showcases found right now.</p>;

  const current = list.find((e) => e.id === selected);

  return (
    <section>
      <h2 className={styles.heading}>Showcases</h2>
      <p className={styles.intro}>
        Games revealed at gaming conferences — Summer Game Fest, State of Play, Nintendo Direct and
        more. Updates automatically as new showcases happen. Add the ones you want to your wishlist.
      </p>

      <div className={styles.events}>
        {list.map((e) => (
          <EventChip key={e.id} event={e} active={e.id === selected} onClick={() => setSelected(e.id)} />
        ))}
      </div>

      {current && (
        <div className={styles.lineupHead}>
          <h3 className={styles.lineupTitle}>{current.name}</h3>
          {current.liveStreamUrl && (
            <a className={styles.watch} href={current.liveStreamUrl} target="_blank" rel="noreferrer">
              ▶ watch
            </a>
          )}
        </div>
      )}

      {games.isLoading ? (
        <p className={styles.muted}>▚ loading lineup…</p>
      ) : (
        <div className={styles.grid}>
          {(games.data?.games ?? [])
            .filter((g) => g.igdbId > 0)
            .map((g) => (
              <DiscoverCard
                key={g.igdbId}
                title={g.name}
                coverUrl={g.coverUrl}
                subtitle={subtitle(g)}
                state={stateOf(g.igdbId)}
                onAdd={() => add(g.igdbId)}
                onOpen={() => navigate(`/igdb/${g.igdbId}`)}
              />
            ))}
        </div>
      )}
    </section>
  );
}

function EventChip({
  event,
  active,
  onClick,
}: {
  event: ShowcaseEvent;
  active: boolean;
  onClick: () => void;
}) {
  return (
    <PixelCard
      className={`${styles.chip} ${active ? styles.chipActive : ""}`}
      onClick={onClick}
    >
      <div className={styles.logo}>
        {event.logoUrl ? <img src={event.logoUrl} alt="" loading="lazy" /> : <span>🎤</span>}
      </div>
      <div className={styles.chipBody}>
        <span className={styles.chipName} title={event.name}>
          {event.name}
        </span>
        <span className={styles.chipMeta}>
          {fmtDate(event.startTime)} · {event.gameCount} games
        </span>
      </div>
    </PixelCard>
  );
}
