"use client";
import Link from "next/link";
import { useState } from "react";
import { useResource, Feedback, useTimezone } from "../../components/resource";
import {
  decimal,
  percent,
  dateTime,
  resultLabel,
  selection,
} from "../../lib/format";
import {
  groupPicksByDate,
  groupRecommendationsByDate,
} from "../../lib/history";
import type { History } from "../../lib/types";
export default function HistoryPage() {
  const [date, setDate] = useState("");
  const tz = useTimezone();
  const { data, error, loading, reload } = useResource<History>(
    `/api/v1/history?timezone=${encodeURIComponent(tz)}&limit=200${date ? `&date=${date}` : ""}`,
  );
  const names = new Map(data?.matches.items.map((m) => [m.id, m]) || []);
  const pickGroups = groupPicksByDate(data?.picks.items || [], tz);
  return (
    <>
      <header className="page-header">
        <div>
          <span className="eyebrow">PERFORMANCE JOURNAL</span>
          <h1>ประวัติและผลลัพธ์</h1>
          <p>คำแนะนำของระบบ และรายการที่คุณเลือก แสดงแยกกัน</p>
        </div>
        <div className="date-filter">
          <label>
            วันที่แข่งขัน{" "}
            <input
              type="date"
              value={date}
              onChange={(e) => {
                setDate(e.target.value);
              }}
            />
          </label>
          <button
            className="outline"
            onClick={() => {
              setDate("");
              reload();
            }}
          >
            ทั้งหมด
          </button>
        </div>
      </header>
      <Feedback error={error} loading={loading} retry={reload} />
      <div className="summary">
        <div>
          <span>กำไรสุทธิ</span>
          <strong>
            {decimal(data?.stats.profit_units)}
            <small> units</small>
          </strong>
        </div>
        <div>
          <span>ROI · รายการสรุปผล</span>
          <strong
            className={(data?.stats.roi || 0) >= 0 ? "positive" : "negative"}
          >
            {percent(data?.stats.roi, true)}
          </strong>
        </div>
        <div>
          <span>เงินเดิมพันที่สรุปผล</span>
          <strong>{decimal(data?.stats.settled_stake_units)}</strong>
        </div>
        <div>
          <span>จำนวนที่สรุปผล</span>
          <strong>{data?.stats.settled_picks || 0}</strong>
        </div>
      </div>
      <section className="panel">
        <h2>รายการที่คุณเลือก</h2>
        <p className="muted">แยกตามวันที่เลือก · {tz}</p>
        <div className="table-wrap">
          <table>
            <thead>
              <tr>
                <th>คู่แข่งขัน / รายการ</th>
                <th>เลือกเมื่อ</th>
                <th>ราคา ณ เลือก</th>
                <th>EV ณ เลือก</th>
                <th>ผล</th>
                <th>กำไร (units)</th>
              </tr>
            </thead>
            {pickGroups.map((group) => (
              <tbody key={group.date}>
                <tr className="history-date-heading">
                  <th scope="rowgroup" colSpan={6}>
                    วันที่เลือก <time dateTime={group.date}>{group.label}</time>
                    <span className="muted">
                      {" "}
                      · {group.items.length} รายการ
                    </span>
                  </th>
                </tr>
                {group.items.map((p) => (
                  <tr key={p.id}>
                    <td>
                      <Link href={`/matches/${p.match_id}`}>
                        {p.match.home} vs {p.match.away}
                      </Link>
                      <small>
                        {selection(p, p.match)} {p.demo && "· ตัวอย่าง"}
                      </small>
                      <small>
                        แข่งขัน{" "}
                        <time dateTime={p.match.kickoff}>
                          {dateTime(p.match.kickoff, true)}
                        </time>
                      </small>
                    </td>
                    <td>
                      <time dateTime={p.picked_at}>
                        {dateTime(p.picked_at, true)}
                      </time>
                    </td>
                    <td>{decimal(p.odds_at_pick)}</td>
                    <td>{percent(p.ev_at_pick, true)}</td>
                    <td>
                      {p.cancelled_at ? "ยกเลิกรายการ" : resultLabel(p.result)}
                    </td>
                    <td>{decimal(p.net_profit_units)}</td>
                  </tr>
                ))}
              </tbody>
            ))}
          </table>
        </div>
        {data?.picks.items.length === 0 && (
          <p className="empty">ไม่มีรายการในช่วงนี้</p>
        )}
      </section>
      {["PLAY", "WATCH", "PASS"].map((status) => (
        <section className="panel" key={status}>
          <h2>
            คำแนะนำระบบ{" "}
            <span className={`badge ${status.toLowerCase()}`}>{status}</span>
          </h2>
          <div className="table-wrap">
            <table>
              <thead>
                <tr>
                  <th>คู่แข่งขัน</th>
                  <th>รายการ</th>
                  <th>EV</th>
                  <th>สร้างเมื่อ</th>
                  <th>เหตุผล</th>
                </tr>
              </thead>
              {groupRecommendationsByDate(
                data?.recommendations.items.filter(
                  (r) => r.status === status,
                ) || [],
                tz,
              ).map((group) => (
                <tbody key={group.date}>
                  <tr className="history-date-heading">
                    <th scope="rowgroup" colSpan={5}>
                      วันที่สร้าง{" "}
                      <time dateTime={group.date}>{group.label}</time>
                      <span className="muted">
                        {" "}
                        · {group.items.length} รายการ
                      </span>
                    </th>
                  </tr>
                  {group.items.map((r) => (
                    <tr key={r.id}>
                      <td>
                        <Link href={`/matches/${r.match_id}`}>
                          {names.get(r.match_id)?.home || "คู่แข่งขัน"} vs{" "}
                          {names.get(r.match_id)?.away || ""}
                        </Link>
                      </td>
                      <td>
                        {r.odds
                          ? selection(r.odds, names.get(r.match_id))
                          : "—"}
                      </td>
                      <td>{percent(r.prediction?.ev, true)}</td>
                      <td>
                        <time dateTime={r.generated_at}>
                          {dateTime(r.generated_at, true)}
                        </time>
                      </td>
                      <td>{r.reasons[0]}</td>
                    </tr>
                  ))}
                </tbody>
              ))}
            </table>
          </div>
        </section>
      ))}
      <section className="panel">
        <h2>ผลการแข่งขัน</h2>
        {data?.results.items.map((r) => (
          <p key={r.match_id}>
            {names.get(r.match_id)?.home}{" "}
            <strong>
              {r.home} – {r.away}
            </strong>{" "}
            {names.get(r.match_id)?.away}
          </p>
        ))}
      </section>
      <section className="panel">
        <h2>ความแม่นยำโมเดล · Brier score</h2>
        <p className="muted">
          ใช้ผล 1X2 กับ prediction ล่าสุดก่อนเริ่มแข่งขันของแต่ละโมเดล ·
          ค่ายิ่งต่ำยิ่งดี
        </p>
        {Object.entries(data?.stats.brier_by_model || {}).map(
          ([model, value]) => (
            <p key={model}>
              {model} <strong>{value.toFixed(4)}</strong>
            </p>
          ),
        )}
        {Object.keys(data?.stats.brier_by_model || {}).length === 0 && (
          <p>ยังไม่มีผลสำหรับประเมิน</p>
        )}
      </section>
    </>
  );
}
