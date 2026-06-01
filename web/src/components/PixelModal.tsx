import { useEffect, type ReactNode } from "react";
import { PixelCard } from "./PixelCard";
import styles from "./PixelModal.module.css";

interface Props {
  onClose: () => void;
  children: ReactNode;
}

export function PixelModal({ onClose, children }: Props) {
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => e.key === "Escape" && onClose();
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [onClose]);

  return (
    <div className={styles.overlay} onClick={onClose}>
      <PixelCard className={styles.card} onClick={(e) => e.stopPropagation()}>
        {children}
      </PixelCard>
    </div>
  );
}
