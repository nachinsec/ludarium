import { useState } from "react";
import { useNavigate } from "react-router-dom";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import type { IGDBGame } from "../lib/types";
import { api } from "../lib/api";
import { DiscoverCard, type AddState } from "../components/DiscoverCard";
import { PixelButton } from "../components/PixelButton";
import styles from "./QuickAdd.module.css";

const PLATFORMS = ["PC", "PlayStation", "Xbox", "Switch", "Nintendo", "Retro", "Mobile", "Other"];
const STATUSES = ["backlog", "playing", "ongoing", "completed", "dropped", "wishlist"];

const meta = (g: IGDBGame) =>
  [g.releaseYear, (g.genres ?? []).slice(0, 2).join(" · ")].filter(Boolean).join("  ·  ");

export function QuickAdd() {
  const qc = useQueryClient();
  const navigate = useNavigate();
  const [platform, setPlatform] = useState("PC");
  const [status, setStatus] = useState("backlog");
  const [term, setTerm] = useState("");
  const [submitted, setSubmitted] = useState("");
  const [added, setAdded] = useState<Set<number>>(new Set());
  const [addingId, setAddingId] = useState<number | null>(null);

  const [manualOpen, setManualOpen] = useState(false);
  const [manualTitle, setManualTitle] = useState("");
  const [manualCover, setManualCover] = useState("");

  const manual = useMutation({
    mutationFn: () =>
      api.addManual({ title: manualTitle.trim(), status, platform, coverUrl: manualCover.trim() }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["library"] });
      setManualTitle("");
      setManualCover("");
    },
  });

  function openManual() {
    setManualTitle(submitted || term);
    setManualOpen(true);
  }

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

  const stateOf = (id: number): AddState =>
    added.has(id) ? "added" : addingId === id ? "adding" : "idle";

  const games = submitted ? search.data?.games ?? [] : [];

  return (
    <section>
      <h2 className={styles.heading}>Add games</h2>
      <p className={styles.intro}>
        Set the platform once, then search and add as many as you like — any platform, no API needed.
      </p>

      <div className={styles.bar}>
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
        <form
          className={styles.searchForm}
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
      </div>

      {games.length > 0 ? (
        <div className={styles.grid}>
          {games.map((g) => (
            <DiscoverCard
              key={g.igdbId}
              title={g.name}
              coverUrl={g.coverUrl}
              subtitle={meta(g)}
              state={stateOf(g.igdbId)}
              onAdd={() => add(g)}
              onOpen={() => navigate(`/igdb/${g.igdbId}`)}
            />
          ))}
        </div>
      ) : (
        submitted && search.data && <p className={styles.muted}>No matches on IGDB.</p>
      )}

      <div className={styles.manual}>
        {!manualOpen ? (
          <button type="button" className={styles.manualToggle} onClick={openManual}>
            Can't find your game? <span>Add it manually →</span>
          </button>
        ) : (
          <form
            className={styles.manualForm}
            onSubmit={(e) => {
              e.preventDefault();
              if (manualTitle.trim()) manual.mutate();
            }}
          >
            <p className={styles.manualHint}>
              No IGDB match needed — it goes in as <b>{platform}</b> · <b>{status}</b>.
            </p>
            <input
              className={styles.search}
              placeholder="Game title"
              value={manualTitle}
              autoFocus
              onChange={(e) => setManualTitle(e.target.value)}
            />
            <input
              className={styles.search}
              placeholder="Cover image URL (optional)"
              value={manualCover}
              onChange={(e) => setManualCover(e.target.value)}
            />
            <div className={styles.manualActions}>
              <PixelButton type="submit" variant="primary" disabled={manual.isPending || !manualTitle.trim()}>
                {manual.isPending ? "Adding…" : "Add manually"}
              </PixelButton>
              <PixelButton type="button" onClick={() => setManualOpen(false)}>
                Cancel
              </PixelButton>
              {manual.isSuccess && <span className={styles.ok}>✓ Added</span>}
            </div>
          </form>
        )}
      </div>
    </section>
  );
}
