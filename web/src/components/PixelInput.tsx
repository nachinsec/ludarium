import type { InputHTMLAttributes } from "react";
import styles from "./PixelInput.module.css";

interface Props extends InputHTMLAttributes<HTMLInputElement> {
  label?: string;
}

/** Labeled text input styled to match the pixel design system. */
export function PixelInput({ label, className, id, ...rest }: Props) {
  const inputId = id ?? rest.name;
  return (
    <label className={styles.field} htmlFor={inputId}>
      {label && <span className={styles.label}>{label}</span>}
      <input
        id={inputId}
        className={[styles.input, className].filter(Boolean).join(" ")}
        {...rest}
      />
    </label>
  );
}
