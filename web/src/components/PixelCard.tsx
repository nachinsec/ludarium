import type { HTMLAttributes } from "react";
import styles from "./PixelCard.module.css";

/** A surface with a chunky, blur-free drop shadow. */
export function PixelCard({ className, ...rest }: HTMLAttributes<HTMLDivElement>) {
  return <div className={[styles.card, className].filter(Boolean).join(" ")} {...rest} />;
}
