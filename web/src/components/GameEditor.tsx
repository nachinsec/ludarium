import { useState } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import type { GameStatus, LibraryGame } from "../lib/types";
import { api, ApiError } from "../lib/api";
import { PixelModal } from "./PixelModal";
import { PixelButton } from "./PixelButton";
import { EditableStars } from "./EditableStars";
import styles from "./GameEditor.module.css";

const STATUSES: { key: GameStatus; label: string }[] = [
  { key: "playing", label: "Playing" },
  { key: "ongoing", label: "Ongoing" },
  { key: "completed", label: "Cleared" },
  { key: "backlog", label: "Backlog" },
  { key: "dropped", label: "Dropped" },
  { key: "wishlist", label: "Wishlist" },
];

export function GameEditor({ game, onClose }: { game: LibraryGame; onClose: () => void }) {
  const qc = useQueryClient();
  const [status, setStatus] = useState<GameStatus>(game.status);
  const [rating, setRating] = useState<number | null>(game.rating);
  const [notes, setNotes] = useState(game.notes);

  const save = useMutation({
    mutationFn: () => api.updateGame(game.id, { status, rating, notes }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["library"] });
      onClose();
    },
  });

  const error = save.error instanceof ApiError ? save.error.message : null;

  return (
    <PixelModal onClose={onClose}>
      <div className={styles.header}>
        <h3 className={styles.title}>{game.title}</h3>
        <span className={styles.hours}>⏱ {game.hours.toFixed(1)}h</span>
      </div>

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
        rows={4}
        placeholder="Your thoughts…"
        onChange={(e) => setNotes(e.target.value)}
      />

      {error && <p className={styles.error}>⚠ {error}</p>}

      <div className={styles.actions}>
        <PixelButton variant="ghost" onClick={onClose}>
          Cancel
        </PixelButton>
        <PixelButton variant="primary" onClick={() => save.mutate()} disabled={save.isPending}>
          {save.isPending ? "Saving…" : "Save"}
        </PixelButton>
      </div>
    </PixelModal>
  );
}
