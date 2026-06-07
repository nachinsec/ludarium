import { useEffect } from "react";
import styles from "./Lightbox.module.css";

interface Props {
  images: string[];
  index: number;
  onIndex: (i: number) => void;
  onClose: () => void;
}

export function Lightbox({ images, index, onIndex, onClose }: Props) {
  const prev = () => onIndex((index - 1 + images.length) % images.length);
  const next = () => onIndex((index + 1) % images.length);

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") onClose();
      else if (e.key === "ArrowLeft") prev();
      else if (e.key === "ArrowRight") next();
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [index]);

  return (
    <div className={styles.overlay} onClick={onClose}>
      <button className={styles.close} onClick={onClose} aria-label="Close">
        ✕
      </button>
      {images.length > 1 && (
        <button
          className={`${styles.arrow} ${styles.left}`}
          onClick={(e) => {
            e.stopPropagation();
            prev();
          }}
          aria-label="Previous"
        >
          ‹
        </button>
      )}
      <figure className={styles.stage} onClick={(e) => e.stopPropagation()}>
        <img className={styles.img} src={images[index]} alt="" />
        {images.length > 1 && (
          <figcaption className={styles.counter}>
            {index + 1} / {images.length}
          </figcaption>
        )}
      </figure>
      {images.length > 1 && (
        <button
          className={`${styles.arrow} ${styles.right}`}
          onClick={(e) => {
            e.stopPropagation();
            next();
          }}
          aria-label="Next"
        >
          ›
        </button>
      )}
    </div>
  );
}
