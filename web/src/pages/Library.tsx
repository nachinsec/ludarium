import { useMemo, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useNavigate } from "react-router-dom";
import type { GameStatus } from "../lib/types";
import { api, ApiError } from "../lib/api";
import { GameCard } from "../components/GameCard";
import { PixelButton } from "../components/PixelButton";
import { PixelCard } from "../components/PixelCard";
import styles from "./Library.module.css";

type Filter = "all" | GameStatus;

const FILTERS: { key: Filter; label: string }[] = [
  { key: "all", label: "All" },
  { key: "playing", label: "Playing" },
  { key: "ongoing", label: "Ongoing" },
  { key: "backlog", label: "Backlog" },
  { key: "completed", label: "Cleared" },
  { key: "wishlist", label: "Wishlist" },
];

export function Library() {
  const qc = useQueryClient();
  const navigate = useNavigate();
  const [filter, setFilter] = useState<Filter>("all");
  const [query, setQuery] = useState("");

  const { data, isLoading } = useQuery({
    queryKey: ["library"],
    queryFn: api.library,
  });
  const games = data?.games ?? [];

  const me = useQuery({ queryKey: ["me"], queryFn: api.me });
  const psnLinked = (me.data?.connections ?? []).some((c) => c.provider === "psn");

  const sync = useMutation({
    mutationFn: api.syncSteam,
    onSuccess: () => qc.invalidateQueries({ queryKey: ["library"] }),
  });

  const syncPsn = useMutation({
    mutationFn: api.syncPSN,
    onSuccess: () => qc.invalidateQueries({ queryKey: ["library"] }),
  });

  const enrich = useMutation({
    mutationFn: api.enrich,
    onSuccess: () => qc.invalidateQueries({ queryKey: ["library"] }),
  });

  const visible = useMemo(() => {
    return games.filter((g) => {
      const matchesFilter = filter === "all" || g.status === filter;
      const matchesQuery = g.title.toLowerCase().includes(query.toLowerCase());
      return matchesFilter && matchesQuery;
    });
  }, [games, filter, query]);

  const syncError = sync.error instanceof ApiError ? sync.error.message : null;

  const syncButton = (
    <PixelButton variant="primary" onClick={() => sync.mutate()} disabled={sync.isPending}>
      {sync.isPending ? "Syncing…" : "↻ Sync Steam"}
    </PixelButton>
  );

  if (isLoading) {
    return <p className={styles.empty}>▚ loading library…</p>;
  }

  if (games.length === 0) {
    return (
      <PixelCard className={styles.emptyCard}>
        <h2 className={styles.emptyTitle}>Your library is empty</h2>
        <p className={styles.emptyText}>
          Connect Steam in Settings, then sync to import your games.
        </p>
        {syncButton}
        {sync.isSuccess && (
          <p className={styles.ok}>✓ Imported {sync.data.total} games</p>
        )}
        {syncError && <p className={styles.error}>⚠ {syncError}</p>}
      </PixelCard>
    );
  }

  return (
    <section>
      <div className={styles.toolbar}>
        <div className={styles.filters}>
          {FILTERS.map((f) => (
            <PixelButton
              key={f.key}
              variant={filter === f.key ? "primary" : "default"}
              onClick={() => setFilter(f.key)}
            >
              {f.label}
            </PixelButton>
          ))}
        </div>
        <input
          className={styles.search}
          placeholder="/ search…"
          value={query}
          onChange={(e) => setQuery(e.target.value)}
        />
      </div>

      <div className={styles.syncRow}>
        {syncButton}
        {psnLinked && (
          <PixelButton onClick={() => syncPsn.mutate()} disabled={syncPsn.isPending}>
            {syncPsn.isPending ? "Syncing…" : "↻ Sync PSN"}
          </PixelButton>
        )}
        <PixelButton onClick={() => enrich.mutate()} disabled={enrich.isPending}>
          {enrich.isPending ? "Enriching…" : "✦ Enrich"}
        </PixelButton>
        {sync.isSuccess && (
          <span className={styles.ok}>
            ✓ {sync.data.total} games · {sync.data.added} new
          </span>
        )}
        {syncPsn.isSuccess && (
          <span className={styles.ok}>✓ PSN: {syncPsn.data.total} games · {syncPsn.data.added} new</span>
        )}
        {enrich.isSuccess && (
          <span className={styles.ok}>✓ enriched {enrich.data.matched}/{enrich.data.checked}</span>
        )}
        {syncError && <span className={styles.error}>⚠ {syncError}</span>}
      </div>

      {visible.length === 0 ? (
        <p className={styles.empty}>No games match.</p>
      ) : (
        <div className={styles.grid}>
          {visible.map((g) => (
            <GameCard key={g.id} game={g} onClick={() => navigate(`/game/${g.id}`)} />
          ))}
        </div>
      )}
    </section>
  );
}
