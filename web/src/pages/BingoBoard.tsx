import { useEffect, useRef, useState } from "react";
import { Link, useParams } from "react-router-dom";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import type { BingoBoard as Board, BingoData } from "../lib/types";
import { api } from "../lib/api";
import { CENTER, normalizeBingo, shuffleFill, winningCells } from "../lib/bingo";
import { PixelButton } from "../components/PixelButton";
import styles from "./BingoBoard.module.css";

export function BingoBoard() {
  const { id } = useParams();
  const boardId = Number(id);
  const { data, isLoading } = useQuery({
    queryKey: ["bingo", boardId],
    queryFn: () => api.bingoBoard(boardId),
    enabled: Number.isFinite(boardId),
  });

  if (isLoading) return <p className={styles.muted}>▚ loading…</p>;
  if (!data) return <p className={styles.muted}>Board not found.</p>;
  return <Editor board={data.board} />;
}

function Editor({ board }: { board: Board }) {
  const qc = useQueryClient();
  const [title, setTitle] = useState(board.title);
  const [data, setData] = useState<BingoData>(normalizeBingo(board.data));
  const [editing, setEditing] = useState(() => data.squares.every((s) => !s.text || s.text === "FREE"));

  const save = useMutation({
    mutationFn: () => api.updateBingo(board.id, { title, data, visibility: board.visibility }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["bingo"] }),
  });

  // Debounced autosave — skip the initial mount.
  const first = useRef(true);
  useEffect(() => {
    if (first.current) {
      first.current = false;
      return;
    }
    const t = setTimeout(() => save.mutate(), 700);
    return () => clearTimeout(t);
  }, [title, data]); // eslint-disable-line react-hooks/exhaustive-deps

  const setSquare = (i: number, patch: Partial<{ text: string; marked: boolean }>) =>
    setData((d) => ({ ...d, squares: d.squares.map((s, j) => (j === i ? { ...s, ...patch } : s)) }));

  const toggleMark = (i: number) => {
    if (data.freeCenter && i === CENTER) return;
    setSquare(i, { marked: !data.squares[i].marked });
  };

  const shuffle = () => setData((d) => ({ ...d, squares: shuffleFill(d.freeCenter) }));

  const toggleFree = () =>
    setData((d) => {
      const freeCenter = !d.freeCenter;
      const squares = d.squares.map((s, j) =>
        j === CENTER ? (freeCenter ? { text: "FREE", marked: true } : { text: "", marked: false }) : s,
      );
      return { freeCenter, squares };
    });

  const resetMarks = () =>
    setData((d) => ({
      ...d,
      squares: d.squares.map((s, j) => ({ ...s, marked: d.freeCenter && j === CENTER })),
    }));

  const wins = winningCells(data.squares);
  const hasBingo = wins.size > 0;

  return (
    <section>
      <Link to="/bingo" className={styles.back}>
        ← Boards
      </Link>

      <div className={styles.toolbar}>
        <input
          className={styles.titleInput}
          value={title}
          onChange={(e) => setTitle(e.target.value)}
          placeholder="Board title"
        />
        <div className={styles.actions}>
          <PixelButton variant={editing ? "primary" : "default"} onClick={() => setEditing((e) => !e)}>
            {editing ? "✓ Done" : "✏️ Edit"}
          </PixelButton>
          <PixelButton onClick={shuffle}>🎲 Shuffle</PixelButton>
          <PixelButton variant={data.freeCenter ? "primary" : "default"} onClick={toggleFree}>
            Free center
          </PixelButton>
          <PixelButton onClick={resetMarks}>↺ Reset</PixelButton>
          <span className={styles.saved}>{save.isPending ? "saving…" : "saved"}</span>
        </div>
      </div>

      {hasBingo && !editing && <div className={styles.banner}>★ BINGO! ★</div>}

      <div className={styles.board}>
        {data.squares.map((sq, i) => {
          const isFree = data.freeCenter && i === CENTER;
          const win = wins.has(i);
          if (editing) {
            return (
              <textarea
                key={i}
                className={styles.cellEdit}
                value={sq.text}
                disabled={isFree}
                placeholder="…"
                onChange={(e) => setSquare(i, { text: e.target.value })}
              />
            );
          }
          return (
            <button
              key={i}
              className={[styles.cell, sq.marked && styles.marked, win && styles.win, isFree && styles.free]
                .filter(Boolean)
                .join(" ")}
              onClick={() => toggleMark(i)}
            >
              <span>{sq.text}</span>
            </button>
          );
        })}
      </div>
    </section>
  );
}
