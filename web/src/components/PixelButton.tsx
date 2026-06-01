import type { ButtonHTMLAttributes } from "react";
import styles from "./PixelButton.module.css";

type Variant = "default" | "primary" | "ghost";

interface Props extends ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: Variant;
}

/** Chunky pixel button that physically "presses" on click. */
export function PixelButton({ variant = "default", className, ...rest }: Props) {
  return (
    <button
      className={[styles.btn, styles[variant], className].filter(Boolean).join(" ")}
      {...rest}
    />
  );
}
