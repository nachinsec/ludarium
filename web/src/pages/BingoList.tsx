import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useNavigate } from "react-router-dom";
import { api } from "../lib/api";
import { emptyBingoData } from "../lib/bingo";
import { PixelCard } from "../components/PixelCard";
import { PixelButton } from "../components/PixelButton";
import styles from "./BingoList.module.css";

export function BingoList() {
  const qc = useQueryClient();
  const navigate = useNavigate();
  const { data, isLoading } = useQuery({ queryKey: ["bingo"], queryFn: api.bingoBoards });

  const create = useMutation({
    mutationFn: () => api.createBingo({ title: "New bingo", data: emptyBingoData() }),
    onSuccess: ({ board }) => {
      qc.invalidateQueries({ queryKey: ["bingo"] });
      navigate(`/bingo/${board.id}`);
    },
  });

  const del = useMutation({
    mutationFn: (id: number) => api.deleteBingo(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["bingo"] }),
  });

  const boards = data?.boards ?? [];

  return (
    <section>
      <div className={styles.head}>
        <div>
          <h2 className={styles.heading}>Conference bingo</h2>
          <p className={styles.intro}>
            Make a 5×5 card of predictions, then mark them live during the next showcase.
          </p>
        </div>
        <PixelButton variant="primary" onClick={() => create.mutate()} disabled={create.isPending}>
          {create.isPending ? "…" : "+ New board"}
        </PixelButton>
      </div>

      {isLoading ? (
        <p className={styles.muted}>▚ loading…</p>
      ) : boards.length === 0 ? (
        <p className={styles.muted}>No boards yet. Create one for the next Direct.</p>
      ) : (
        <div className={styles.grid}>
          {boards.map((b) => (
            <PixelCard key={b.id} className={styles.card}>
              <button className={styles.open} onClick={() => navigate(`/bingo/${b.id}`)}>
                <span className={styles.title}>{b.title}</span>
                <span className={styles.meta}>edited {b.updatedAt.split(" ")[0]}</span>
              </button>
              <PixelButton onClick={() => del.mutate(b.id)} disabled={del.isPending}>
                ✕
              </PixelButton>
            </PixelCard>
          ))}
        </div>
      )}
    </section>
  );
}
