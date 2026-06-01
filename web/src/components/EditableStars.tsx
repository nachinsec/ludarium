import { useState } from "react";
import styles from "./EditableStars.module.css";

interface Props {
  value: number | null;
  onChange: (v: number | null) => void;
}

export function EditableStars({ value, onChange }: Props) {
  const [hover, setHover] = useState(0);
  const shown = hover || value || 0;

  return (
    <div className={styles.row}>
      {[1, 2, 3, 4, 5].map((n) => (
        <button
          key={n}
          type="button"
          className={n <= shown ? styles.on : styles.off}
          onMouseEnter={() => setHover(n)}
          onMouseLeave={() => setHover(0)}
          // Click the current rating again to clear it.
          onClick={() => onChange(value === n ? null : n)}
        >
          ★
        </button>
      ))}
      {value && (
        <button type="button" className={styles.clear} onClick={() => onChange(null)}>
          clear
        </button>
      )}
    </div>
  );
}
