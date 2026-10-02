import type { Match, Selection } from "./types";
export const percent = (n: number | null | undefined, signed = false) =>
  n == null ? "—" : `${signed && n > 0 ? "+" : ""}${(n * 100).toFixed(1)}%`;
export const decimal = (n: number | null | undefined) =>
  n == null ? "—" : n.toFixed(2);
export const dateTime = (date: string) =>
  new Intl.DateTimeFormat("th-TH", {
    day: "numeric",
    month: "short",
    hour: "2-digit",
    minute: "2-digit",
  }).format(new Date(date));
export const kickoff = (date: string) =>
  new Intl.DateTimeFormat("th-TH", {
    hour: "2-digit",
    minute: "2-digit",
  }).format(new Date(date));
export function selection(s: Selection, m?: Match) {
  const side =
    s.selection === "home"
      ? m?.home || "เจ้าบ้าน"
      : s.selection === "away"
        ? m?.away || "ทีมเยือน"
        : s.selection === "draw"
          ? "เสมอ"
          : s.selection === "over"
            ? "สูง"
            : "ต่ำ";
  return `${side}${s.line != null ? ` ${s.market === "AH" && s.line > 0 ? "+" : ""}${s.line}` : ""}`;
}
export const resultLabel = (r: string | null) =>
  ({
    win: "ชนะ",
    "half-win": "ชนะครึ่ง",
    push: "คืนทุน",
    "half-loss": "แพ้ครึ่ง",
    loss: "แพ้",
    void: "ยกเลิกการแข่งขัน",
  })[r || ""] || "รอผล";
