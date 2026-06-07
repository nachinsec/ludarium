import { useState } from "react";
import { Link, useParams } from "react-router-dom";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import type { GameStatus, LibraryGame } from "../lib/types";
import { api, ApiError } from "../lib/api";
import { PixelButton } from "../components/PixelButton";
import { EditableStars } from "../components/EditableStars";
import { StatusBadge } from "../components/StatusBadge";
import { Lightbox } from "../components/Lightbox";
import styles from "./GameDetail.module.css";

const STATUSES: { key: GameStatus; label: string }[] = [
  { key: "playing", label: "Playing" },
  { key: "ongoing", label: "Ongoing" },
  { key: "completed", label: "Cleared" },
  { key: "backlog", label: "Backlog" },
  { key: "dropped", label: "Dropped" },
  { key: "wishlist", label: "Wishlist" },
];

const shortPlatform = (p: string) => (p === "PC / Steam" ? "Steam" : p);
const date = (s?: string | null) => (s ? s.split(" ")[0] : null);

export function GameDetail() {
  const { id } = useParams();
  const gameId = Number(id);
  const { data, isLoading } = useQuery({
    queryKey: ["game", gameId],
    queryFn: () => api.game(gameId),
    enabled: Number.isFinite(gameId),
  });

  if (isLoading) return <p className={styles.muted}>▚ loading…</p>;
  if (!data) return <p className={styles.muted}>Game not found.</p>;
  return <Detail game={data.game} />;
}

function Detail({ game }: { game: LibraryGame }) {
  const qc = useQueryClient();
  const [status, setStatus] = useState<GameStatus>(game.status);
  const [rating, setRating] = useState<number | null>(game.rating);
  const [notes, setNotes] = useState(game.notes);
  const [shot, setShot] = useState<number | null>(null);
  const shots = game.screenshots ?? [];

  const save = useMutation({
    mutationFn: () => api.updateGame(game.id, { status, rating, notes }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["library"] });
      qc.invalidateQueries({ queryKey: ["game", game.id] });
    },
  });
  const error = save.error instanceof ApiError ? save.error.message : null;
  const genres = (game.genres ?? []).join(" · ");

  return (
    <section>
      <Link to="/" className={styles.back}>
        ← Library
      </Link>

      <div className={styles.detail}>
        <div className={styles.coverCol}>
          <div className={styles.cover}>
            {game.coverUrl && <img src={game.coverUrl} alt="" />}
            <div className={styles.badge}>
              <StatusBadge status={status} />
            </div>
          </div>
          {game.platform && <span className={styles.platform}>{shortPlatform(game.platform)}</span>}
        </div>

        <div className={styles.info}>
          <h2 className={styles.title}>{game.title}</h2>
          <div className={styles.meta}>
            {[game.developer, game.releaseYear].filter(Boolean).join(" · ")}
          </div>
          {genres && <div className={styles.genres}>{genres}</div>}
          <div className={styles.statRow}>
            <span className={styles.stat}>⏱ {game.hours.toFixed(1)}h played</span>
            {!!game.score && <span className={styles.score}>★ {game.score} / 100</span>}
          </div>

          {game.summary && <p className={styles.summary}>{game.summary}</p>}

          {shots.length > 0 && (
            <div className={styles.shots}>
              {shots.map((s, i) => (
                <button key={s} className={styles.shotBtn} onClick={() => setShot(i)} aria-label="View screenshot">
                  <img src={s} alt="" loading="lazy" className={styles.shot} />
                </button>
              ))}
            </div>
          )}

          <label className={styles.label}>Status</label>
          <div className={styles.statuses}>
            {STATUSES.map((s) => (
              <PixelButton
                key={s.key}
                variant={status === s.key ? "primary" : "default"}
                onClick={() => setStatus(s.key)}
              >
                {s.label}
              </PixelButton>
            ))}
          </div>

          <label className={styles.label}>Rating</label>
          <EditableStars value={rating} onChange={setRating} />

          <label className={styles.label}>Notes</label>
          <textarea
            className={styles.notes}
            value={notes}
            rows={5}
            placeholder="Your thoughts…"
            onChange={(e) => setNotes(e.target.value)}
          />

          {(date(game.startedAt) || date(game.finishedAt)) && (
            <div className={styles.dates}>
              {date(game.startedAt) && <span>Started {date(game.startedAt)}</span>}
              {date(game.finishedAt) && <span>· Finished {date(game.finishedAt)}</span>}
            </div>
          )}

          {error && <p className={styles.error}>⚠ {error}</p>}
          {save.isSuccess && <p className={styles.ok}>✓ Saved</p>}

          <PixelButton variant="primary" onClick={() => save.mutate()} disabled={save.isPending}>
            {save.isPending ? "Saving…" : "Save"}
          </PixelButton>
        </div>
      </div>

      {shot !== null && (
        <Lightbox images={shots} index={shot} onIndex={setShot} onClose={() => setShot(null)} />
      )}
    </section>
  );
}
