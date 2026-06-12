import { useState } from "react";
import { Link, useNavigate, useParams } from "react-router-dom";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import type { GameStatus, IGDBGame } from "../lib/types";
import { api } from "../lib/api";
import { PixelButton } from "../components/PixelButton";
import { Lightbox } from "../components/Lightbox";
import { IgdbTrailer, IgdbLinks } from "../components/IgdbExtras";
import styles from "./IgdbDetail.module.css";

const STATUSES: { key: GameStatus; label: string }[] = [
  { key: "wishlist", label: "Wishlist" },
  { key: "backlog", label: "Backlog" },
  { key: "playing", label: "Playing" },
  { key: "completed", label: "Cleared" },
];

export function IgdbDetail() {
  const { id } = useParams();
  const igdbId = Number(id);
  const { data, isLoading } = useQuery({
    queryKey: ["igdb-game", igdbId],
    queryFn: () => api.igdbGame(igdbId),
    enabled: Number.isFinite(igdbId),
  });

  if (isLoading) return <p className={styles.muted}>▚ loading…</p>;
  if (!data) return <p className={styles.muted}>Game not found.</p>;
  return <Detail game={data.game} />;
}

function Detail({ game }: { game: IGDBGame }) {
  const qc = useQueryClient();
  const navigate = useNavigate();
  const [status, setStatus] = useState<GameStatus>("wishlist");
  const [platform, setPlatform] = useState((game.platforms ?? [])[0] ?? "");
  const [shot, setShot] = useState<number | null>(null);
  const shots = game.screenshots ?? [];
  const platforms = game.platforms ?? [];

  const add = useMutation({
    mutationFn: () => api.addGame(game.igdbId, status, platform),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["library"] }),
  });

  const meta = [game.developer, game.releaseYear].filter(Boolean).join(" · ");
  const genres = (game.genres ?? []).join(" · ");

  return (
    <section>
      <button className={styles.back} onClick={() => navigate(-1)}>
        ← Back
      </button>

      <div className={styles.detail}>
        <div className={styles.coverCol}>
          <div className={styles.cover}>{game.coverUrl && <img src={game.coverUrl} alt="" />}</div>
        </div>

        <div className={styles.info}>
          <h2 className={styles.title}>{game.name}</h2>
          {meta && <div className={styles.meta}>{meta}</div>}
          {genres && <div className={styles.genres}>{genres}</div>}
          <div className={styles.statRow}>
            {platforms.length > 0 && <span className={styles.stat}>{platforms.join(" · ")}</span>}
            {!!game.score && <span className={styles.score}>★ {game.score} / 100</span>}
          </div>

          {game.summary && <p className={styles.summary}>{game.summary}</p>}

          <IgdbTrailer game={game} />

          {shots.length > 0 && (
            <div className={styles.shots}>
              {shots.map((s, i) => (
                <button key={s} className={styles.shotBtn} onClick={() => setShot(i)} aria-label="View screenshot">
                  <img src={s} alt="" loading="lazy" className={styles.shot} />
                </button>
              ))}
            </div>
          )}

          <label className={styles.label}>Add to library as</label>
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

          {platforms.length > 0 && (
            <>
              <label className={styles.label}>Platform</label>
              <select
                className={styles.select}
                value={platform}
                onChange={(e) => setPlatform(e.target.value)}
              >
                {platforms.map((p) => (
                  <option key={p}>{p}</option>
                ))}
                <option value="">Other</option>
              </select>
            </>
          )}

          {add.isSuccess ? (
            <p className={styles.ok}>
              ✓ Added to your library · <Link to="/">go to library</Link>
            </p>
          ) : (
            <PixelButton variant="primary" onClick={() => add.mutate()} disabled={add.isPending}>
              {add.isPending ? "Adding…" : "+ Add to library"}
            </PixelButton>
          )}

          <IgdbLinks game={game} />
        </div>
      </div>

      {shot !== null && (
        <Lightbox images={shots} index={shot} onIndex={setShot} onClose={() => setShot(null)} />
      )}
    </section>
  );
}
