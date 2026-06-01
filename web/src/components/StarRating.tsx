import styles from "./StarRating.module.css";

interface Props {
  /** 1..5, or null when unrated. */
  value: number | null;
  max?: number;
}

/** Pixel star rating. Renders filled/empty blocks; dims when unrated. */
export function StarRating({ value, max = 5 }: Props) {
  const filled = value ?? 0;
  return (
    <span className={styles.row} aria-label={value ? `${value} of ${max}` : "unrated"}>
      {Array.from({ length: max }, (_, i) => (
        <span key={i} className={i < filled ? styles.on : styles.off}>
          ★
        </span>
      ))}
    </span>
  );
}
