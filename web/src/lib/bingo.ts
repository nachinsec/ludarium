import type { BingoData, BingoSquare } from "./types";
import { BINGO_TEMPLATES } from "./bingoTemplates";

export const SIZE = 5;
export const CELLS = SIZE * SIZE;
export const CENTER = 12;

// The 12 winning lines (5 rows, 5 cols, 2 diagonals) as cell indices.
export const LINES: number[][] = (() => {
  const lines: number[][] = [];
  for (let r = 0; r < SIZE; r++) lines.push([...Array(SIZE)].map((_, c) => r * SIZE + c));
  for (let c = 0; c < SIZE; c++) lines.push([...Array(SIZE)].map((_, r) => r * SIZE + c));
  lines.push([...Array(SIZE)].map((_, i) => i * SIZE + i));
  lines.push([...Array(SIZE)].map((_, i) => i * SIZE + (SIZE - 1 - i)));
  return lines;
})();

export function emptyBingoData(): BingoData {
  return {
    freeCenter: true,
    squares: Array.from({ length: CELLS }, (_, i) =>
      i === CENTER ? { text: "FREE", marked: true } : { text: "", marked: false },
    ),
  };
}

// Normalize whatever the server returns into a valid 25-cell board.
export function normalizeBingo(data: BingoData | undefined): BingoData {
  if (!data?.squares || data.squares.length !== CELLS) return emptyBingoData();
  return data;
}

// shuffleFill picks fresh squares from the cliché pool, keeping the free centre.
export function shuffleFill(freeCenter: boolean): BingoSquare[] {
  const pool = [...BINGO_TEMPLATES].sort(() => Math.random() - 0.5);
  let p = 0;
  return Array.from({ length: CELLS }, (_, i) => {
    if (freeCenter && i === CENTER) return { text: "FREE", marked: true };
    return { text: pool[p++] ?? "", marked: false };
  });
}

// winningCells returns the set of cell indices that form any completed line.
export function winningCells(squares: BingoSquare[]): Set<number> {
  const cells = new Set<number>();
  for (const line of LINES) {
    if (line.every((i) => squares[i]?.marked)) line.forEach((i) => cells.add(i));
  }
  return cells;
}
