"use client";
import Link from "next/link";
import { useState } from "react";
import { useResource, Feedback } from "../../components/resource";
import { cancelPick } from "../../lib/api";
import {
  decimal,
  percent,
  selection,
  resultLabel,
  dateTime,
} from "../../lib/format";
import type { Page, Pick } from "../../lib/types";
export default function Picks() {
  const { data, error, loading, reload } = useResource<Page<Pick>>(
      "/api/v1/user-picks?limit=200",
    ),
    [actionError, setError] = useState(""),
    [busy, setBusy] = useState("");
  async function cancel(id: string) {
    setBusy(id);
    setError("");
    try {
      await cancelPick(id);
      reload();
    } catch (e) {
      setError((e as Error).message);
    } finally {
      setBusy("");
    }
  }
  return (
    <>
      <header className="page-header">
        <div>
          <span className="eyebrow">YOUR DECISIONS</span>
          <h1>รายการที่เลือก</h1>
          <p>ราคาและความน่าจะเป็นถูกเก็บไว้ ณ เวลาที่คุณเลือก</p>
        </div>
        <button className="outline" onClick={reload}>
          ↻ โหลดข้อมูลใหม่
        </button>
      </header>
      <Feedback error={error} loading={loading} retry={reload} />
      {actionError && (
        <p className="notice error" role="alert">
          {actionError}
        </p>
      )}
      {data?.items.length === 0 && (
        <div className="empty">
          ยังไม่มีรายการที่เลือก
          <br />
          <Link href="/">ดูคำแนะนำวันนี้ →</Link>
        </div>
      )}
      <div className="pick-list">
        {data?.items.map((p) => (
          <article className="panel pick-row" key={p.id}>
            <div>
              <span className="eyebrow">
                {p.match.competition} {p.demo && "· ตัวอย่าง"}
              </span>
              <h2>
                <Link href={`/matches/${p.match_id}`}>
                  {p.match.home} vs {p.match.away}
                </Link>
              </h2>
              <strong>{selection(p, p.match)}</strong>
              <p className="muted">
                เลือกเมื่อ {dateTime(p.picked_at)} · {p.stake_units} unit
              </p>
            </div>
            <div className="pick-values">
              <div>
                <small>ราคา ณ เลือก</small>
                <b>{decimal(p.odds_at_pick)}</b>
              </div>
              <div>
                <small>โอกาส ณ เลือก</small>
                <b>{percent(p.probability_at_pick)}</b>
              </div>
              <div>
                <small>EV ณ เลือก</small>
                <b className="positive">{percent(p.ev_at_pick, true)}</b>
              </div>
              <div>
                <small>ผล</small>
                <b>{resultLabel(p.result)}</b>
              </div>
            </div>
            {p.settled_at ? (
              <strong
                className={
                  (p.net_profit_units || 0) >= 0 ? "positive" : "negative"
                }
              >
                {decimal(p.net_profit_units)} units
              </strong>
            ) : (
              <button
                className="outline"
                disabled={busy === p.id}
                onClick={() => cancel(p.id)}
              >
                {busy === p.id ? "กำลังยกเลิก…" : "ยกเลิกรายการ"}
              </button>
            )}
          </article>
        ))}
      </div>
    </>
  );
}
