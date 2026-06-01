import { useEffect, useMemo, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import type { IGDBGame, Recommendation } from "../lib/types";
import { api, ApiError } from "../lib/api";
import { PixelButton } from "../components/PixelButton";
import styles from "./Oracle.module.css";

const ROUNDS = 5;
const STAGES = ["READING YOUR PICKS", "MINING THE ARCHIVE", "BUILDING MATCHES", "SUMMONING"];

const shuffle = (g: IGDBGame[]) => [...g].sort(() => Math.random() - 0.5);
const meta = (year: number | null, genres: string[] | null) =>
  [year, (genres ?? []).slice(0, 3).join(" · ")].filter(Boolean).join("  ·  ");

function StagedLoader() {
  const [i, setI] = useState(0);
  useEffect(() => {
    const t = setInterval(() => setI((x) => (x + 1) % STAGES.length), 1400);
    return () => clearInterval(t);
  }, []);
  return <div className={styles.loader}>▚ {STAGES[i]}…</div>;
}

export function Oracle() {
  const qc = useQueryClient();
  const [picks, setPicks] = useState<string[]>([]);
  const [idx, setIdx] = useState(0);
  const [seed, setSeed] = useState(0);
  const [added, setAdded] = useState<Set<number>>(new Set());
  const [addingId, setAddingId] = useState<number | null>(null);

  const candidates = useQuery({
    queryKey: ["oracle-candidates", seed],
    queryFn: api.oracleCandidates,
    staleTime: 0,
  });
  const pool = useMemo(
    () => shuffle((candidates.data?.games ?? []).filter((g) => g.coverUrl)),
    [candidates.data],
  );
  useEffect(() => setIdx(0), [candidates.data]);

  const rec = useMutation({ mutationFn: (preferred: string[]) => api.recommend(preferred) });

  function nextPair() {
    const ni = idx + 2;
    if (ni + 1 >= pool.length) setSeed((s) => s + 1);
    else setIdx(ni);
  }

  function pick(name: string) {
    const next = [...picks, name];
    setPicks(next);
    if (next.length >= ROUNDS) rec.mutate(next);
    else nextPair();
  }

  function restart() {
    setPicks([]);
    setIdx(0);
    setSeed((s) => s + 1);
    rec.reset();
  }

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

  // --- results (big horizontal cards, stacked + scrollable)
  if (rec.data) {
    return (
      <section>
        <h2 className={styles.heading}>The Oracle has spoken</h2>
        <p className={styles.intro}>Based on what you were drawn to, you might love:</p>
        <div className={styles.results}>
          {rec.data.recommendations
            .filter((r) => r.igdbId > 0)
            .map((r) => (
              <ResultRow
                key={r.igdbId}
                rec={r}
                state={added.has(r.igdbId) ? "added" : addingId === r.igdbId ? "adding" : "idle"}
                onAdd={() => handleAdd(r.igdbId)}
              />
            ))}
        </div>
        <div className={styles.againRow}>
          <PixelButton variant="primary" onClick={restart}>
            ↻ Play again
          </PixelButton>
        </div>
      </section>
    );
  }

  // --- loading
  if (rec.isPending) {
    return (
      <section className={styles.center}>
        <h2 className={styles.heading}>Consulting the Oracle</h2>
        <StagedLoader />
      </section>
    );
  }

  // --- error
  if (rec.error) {
    const msg = rec.error instanceof ApiError ? rec.error.message : "something went wrong";
    return (
      <section className={styles.center}>
        <p className={styles.error}>⚠ {msg}</p>
        <PixelButton variant="primary" onClick={restart}>
          Try again
        </PixelButton>
      </section>
    );
  }

  // --- quiz (side by side)
  const pair = pool.slice(idx, idx + 2);
  if (pair.length < 2) {
    return <p className={styles.muted}>▚ summoning the Oracle…</p>;
  }

  return (
    <section className={styles.center}>
      <h2 className={styles.heading}>Which one calls to you?</h2>
      <p className={styles.progress}>Pick {picks.length + 1} / {ROUNDS}</p>

      <div className={styles.versus}>
        <PickCard game={pair[0]} onPick={() => pick(pair[0].name)} />
        <span className={styles.vs}>VS</span>
        <PickCard game={pair[1]} onPick={() => pick(pair[1].name)} />
      </div>

      <div className={styles.skipRow}>
        <PixelButton variant="ghost" onClick={nextPair}>
          I don’t know either → new pair
        </PixelButton>
      </div>
    </section>
  );
}

function PickCard({ game, onPick }: { game: IGDBGame; onPick: () => void }) {
  return (
    <button className={styles.pick} onClick={onPick} type="button">
      <img className={styles.pickImg} src={game.coverUrl} alt="" />
      <span className={styles.pickName}>{game.name}</span>
      <span className={styles.pickMeta}>{meta(game.releaseYear, game.genres)}</span>
    </button>
  );
}

function ResultRow({
  rec,
  state,
  onAdd,
}: {
  rec: Recommendation;
  state: "idle" | "adding" | "added";
  onAdd: () => void;
}) {
  return (
    <article className={styles.resultRow}>
      {rec.coverUrl && <img className={styles.resultCover} src={rec.coverUrl} alt="" />}
      <div className={styles.resultInfo}>
        <h3 className={styles.resultTitle}>{rec.title}</h3>
        <span className={styles.resultMeta}>{meta(rec.releaseYear, rec.genres)}</span>
        <p className={styles.resultReason}>{rec.reason}</p>
        <PixelButton
          className={styles.resultAdd}
          variant={state === "added" ? "default" : "primary"}
          disabled={state !== "idle"}
          onClick={onAdd}
        >
          {state === "added" ? "✓ Added" : state === "adding" ? "…" : "+ Add"}
        </PixelButton>
      </div>
    </article>
  );
}
