import { PixelCard } from "./PixelCard";
import { PixelButton } from "./PixelButton";
import styles from "./DiscoverCard.module.css";

export type AddState = "idle" | "adding" | "added";

interface Props {
  title: string;
  coverUrl: string;
  subtitle?: string;
  state: AddState;
  onAdd: () => void;
  onOpen?: () => void; // click cover/title to open a detail view
}

export function DiscoverCard({ title, coverUrl, subtitle, state, onAdd, onOpen }: Props) {
  return (
    <PixelCard className={styles.card}>
      <div
        className={`${styles.cover} ${onOpen ? styles.clickable : ""}`}
        onClick={onOpen}
        role={onOpen ? "button" : undefined}
      >
        {coverUrl && <img className={styles.img} src={coverUrl} alt="" loading="lazy" />}
      </div>
      <div className={styles.body}>
        <h3
          className={`${styles.title} ${onOpen ? styles.clickable : ""}`}
          title={title}
          onClick={onOpen}
        >
          {title}
        </h3>
        {subtitle && <p className={styles.subtitle}>{subtitle}</p>}
        <PixelButton
          className={styles.add}
          variant={state === "added" ? "default" : "primary"}
          disabled={state !== "idle"}
          onClick={onAdd}
        >
          {state === "added" ? "✓ Added" : state === "adding" ? "…" : "+ Add"}
        </PixelButton>
      </div>
    </PixelCard>
  );
}
