import styles from "./ProgressBlocks.module.css";

interface Props {
  /** 0..100 */
  percent: number;
  blocks?: number;
}

/** Blocky, stepped progress bar (no smooth gradients). */
export function ProgressBlocks({ percent, blocks = 12 }: Props) {
  const clamped = Math.max(0, Math.min(100, percent));
  const filled = Math.round((clamped / 100) * blocks);
  return (
    <span className={styles.bar} aria-label={`${clamped}%`}>
      {Array.from({ length: blocks }, (_, i) => (
        <span key={i} className={i < filled ? styles.on : styles.off} />
      ))}
    </span>
  );
}
