"use client";
import Link from "next/link";
import { useState } from "react";
import type { Card, Pick } from "../lib/types";
import { cancelPick, selectPick } from "../lib/api";
import { decimal, kickoff, percent, selection } from "../lib/format";
export function RecommendationCard({
  card,
  onPicked,
}: {
  card: Card;
  onPicked: () => void;
}) {
  const [busy, setBusy] = useState(false),
    [error, setError] = useState(""),
    [dismissed, setDismissed] = useState(false),
    [selected, setSelected] = useState<Pick | null>(null),
    [cancelled, setCancelled] = useState<string[]>([]);
  const r = card.recommendation,
    p = r?.prediction,
    o = r?.odds,
    status = r?.status || "PASS";
  async function choose() {
    if (!r) return;
    setBusy(true);
    setError("");
    try {
      const pick = await selectPick(r.id);
      setSelected(pick);
      onPicked();
    } catch (e) {
      setError((e as Error).message);
    } finally {
      setBusy(false);
    }
  }
  const pick =
    selected && selected.recommendation_id === r?.id
      ? card.pick?.id === selected.id
        ? card.pick
        : selected
      : card.pick;
  const activePick = pick && !cancelled.includes(pick.id) ? pick : null;
  async function cancel() {
    if (!activePick) return;
    setBusy(true);
    setError("");
    try {
      await cancelPick(activePick.id);
      setCancelled((ids) => [...ids, activePick.id]);
      setSelected(null);
      onPicked();
    } catch (e) {
      setError((e as Error).message);
    } finally {
      setBusy(false);
    }
  }
  return (
    <article
      className={`match-card ${status.toLowerCase()} ${dismissed ? "dismissed" : ""}`}
    >
      <div className="card-top">
        <span>{card.competition}</span>
        <span>{kickoff(card.kickoff)} น.</span>
      </div>
      <div className="fixture">
        <h2>
          {card.home}
          <span>vs</span>
          {card.away}
        </h2>
        <span className={`badge ${status.toLowerCase()}`}>{status}</span>
      </div>
      {card.result && (
        <p className="score">
          ผลการแข่งขัน {card.result.home} – {card.result.away}
        </p>
      )}
      <div className="recommendation">
        <span className="eyebrow">เล่นอะไร</span>
        <strong>{o ? selection(o, card) : "ไม่มีรายการแนะนำ"}</strong>
        <small>
          {o?.market || "—"} · {o?.bookmaker || "ข้อมูลไม่ครบ"}
        </small>
      </div>
      <div className="metrics">
        <div>
          <span>ราคา</span>
          <strong>{decimal(o?.odds)}</strong>
        </div>
        <div>
          <span>{o?.market === "1X2" ? "โอกาสชนะ" : "โอกาสได้กำไร"}</span>
          <strong>{percent(p?.probability)}</strong>
        </div>
        <div>
          <span>ความคุ้มค่า EV</span>
          <strong className={p && p.ev > 0 ? "positive" : "negative"}>
            {percent(p?.ev, true)}
          </strong>
        </div>
      </div>
      <p className="reason">
        {r?.reasons[0] || "ยังไม่มีข้อมูลสำหรับวิเคราะห์"}
      </p>
      {error && (
        <p className="error-text" role="alert">
          {error}
        </p>
      )}
      <div className="card-actions">
        {activePick ? (
          <>
            <button className="primary" disabled>
              เลือกแล้ว ✓
            </button>
            {!activePick.settled_at && (
              <button className="quiet" onClick={cancel} disabled={busy}>
                {busy ? "กำลังยกเลิก…" : "ยกเลิกรายการ"}
              </button>
            )}
          </>
        ) : status !== "PASS" && !dismissed ? (
          <>
            <button className="primary" onClick={choose} disabled={busy}>
              {busy ? "กำลังบันทึก…" : "เลือกเล่น"}
            </button>
            <button
              className="quiet"
              onClick={() => setDismissed(true)}
              disabled={busy}
            >
              ไม่เล่น
            </button>
          </>
        ) : null}
        {dismissed && (
          <button className="quiet" onClick={() => setDismissed(false)}>
            แสดงอีกครั้ง
          </button>
        )}
        <Link href={`/matches/${card.id}`} className="detail-link">
          รายละเอียด ↗
        </Link>
      </div>
    </article>
  );
}
