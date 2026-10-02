"use client";
import Link from "next/link";
import { useParams } from "next/navigation";
import { useResource, Feedback } from "../../../components/resource";
import { RecommendationCard } from "../../../components/recommendation-card";
import { decimal, percent, dateTime, selection } from "../../../lib/format";
import type { Detail } from "../../../lib/types";
export default function MatchDetail() {
  const { id } = useParams<{ id: string }>();
  const {
    data: d,
    error,
    loading,
    reload,
  } = useResource<Detail>(`/api/v1/matches/${id}`);
  const p = d?.recommendation?.prediction,
    l = d?.lineup_snapshots.at(-1);
  return (
    <>
      <Link className="back-link" href="/">
        ← กลับหน้าวันนี้
      </Link>
      <header className="page-header">
        <div>
          <span className="eyebrow">MATCH INTELLIGENCE</span>
          <h1>{d ? `${d.home} vs ${d.away}` : "รายละเอียดการแข่งขัน"}</h1>
          <p>
            {d?.competition} · {d && dateTime(d.kickoff)}
          </p>
        </div>
        <button className="outline" onClick={reload}>
          ↻ โหลดข้อมูลใหม่
        </button>
      </header>
      <Feedback error={error} loading={loading} retry={reload} />
      {d && (
        <>
          <RecommendationCard card={d} onPicked={reload} />
          <section className="panel">
            <h2>ข้อมูลโมเดล</h2>
            <div className="detail-metrics">
              <div>
                <span>ประตูคาดการณ์ (เจ้าบ้าน / ทีมเยือน)</span>
                <strong>
                  {decimal(p?.expected_goals_home)} /{" "}
                  {decimal(p?.expected_goals_away)}
                </strong>
              </div>
              <div>
                <span>ราคายุติธรรม</span>
                <strong>{decimal(p?.fair_odds)}</strong>
              </div>
              <div>
                <span>ราคาขั้นต่ำสำหรับ PLAY</span>
                <strong>
                  {p?.minimum_acceptable_odds != null
                    ? (
                        Math.ceil(p.minimum_acceptable_odds * 100) / 100
                      ).toFixed(2)
                    : "—"}
                </strong>
              </div>
              <div>
                <span>โอกาสจากราคา / หลังหัก margin</span>
                <strong>
                  {percent(p?.market_implied_probability)} /{" "}
                  {percent(p?.margin_removed_probability)}
                </strong>
              </div>
            </div>
            <p className="muted">
              {p?.model_version} · {p && dateTime(p.generated_at)}
            </p>
            {d.recommendation?.reasons.map((reason) => (
              <p key={reason}>• {reason}</p>
            ))}
            {p && (
              <div className="distribution">
                {Object.entries(p.distribution).map(([key, value]) => (
                  <div key={key}>
                    <span>
                      {
                        {
                          p_win: "ชนะเต็ม",
                          p_half_win: "ชนะครึ่ง",
                          p_push: "คืนทุน",
                          p_half_loss: "แพ้ครึ่ง",
                          p_loss: "แพ้เต็ม",
                        }[key]
                      }
                    </span>
                    <strong>{percent(value)}</strong>
                  </div>
                ))}
              </div>
            )}
          </section>
          <section className="panel">
            <h2>รายชื่อผู้เล่น · ข้อมูลจำลอง</h2>
            {l ? (
              <>
                <p className="muted">บันทึกเมื่อ {dateTime(l.captured_at)}</p>
                <div className="lineups">
                  {(["home", "away"] as const).map((side) => (
                    <div key={side}>
                      <h3>{side === "home" ? d.home : d.away}</h3>
                      <ol>
                        {l[side].starting_xi.map((name) => (
                          <li key={name}>{name}</li>
                        ))}
                      </ol>
                      <h4>ตัวสำรอง</h4>
                      <p>{l[side].substitutes.join(" · ")}</p>
                      <h4>บาดเจ็บ</h4>
                      <p>{l[side].injuries.join(" · ") || "ไม่มีข้อมูล"}</p>
                      <h4>ติดโทษแบน</h4>
                      <p>{l[side].suspensions.join(" · ") || "ไม่มี"}</p>
                    </div>
                  ))}
                </div>
              </>
            ) : (
              <p>ยังไม่มีรายชื่อผู้เล่น</p>
            )}
          </section>
          <section className="panel">
            <h2>ประวัติราคา</h2>
            <div className="table-wrap">
              <table>
                <thead>
                  <tr>
                    <th>เวลา</th>
                    <th>ช่วง</th>
                    <th>ตลาด / รายการ</th>
                    <th>ราคา</th>
                    <th>เจ้ามือ</th>
                  </tr>
                </thead>
                <tbody>
                  {d.odds_snapshots.map((o) => (
                    <tr key={o.id}>
                      <td>{dateTime(o.captured_at)}</td>
                      <td>{o.phase}</td>
                      <td>
                        {o.market} · {selection(o, d)}
                      </td>
                      <td>{decimal(o.odds)}</td>
                      <td>{o.bookmaker}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </section>
          <section className="panel">
            <h2>ประวัติ prediction</h2>
            <div className="table-wrap">
              <table>
                <thead>
                  <tr>
                    <th>เวลา</th>
                    <th>รายการ</th>
                    <th>โอกาสได้กำไร</th>
                    <th>EV</th>
                    <th>Fair odds</th>
                    <th>โมเดล</th>
                  </tr>
                </thead>
                <tbody>
                  {d.prediction_snapshots.map((p) => (
                    <tr key={p.id}>
                      <td>{dateTime(p.generated_at)}</td>
                      <td>{selection(p, d)}</td>
                      <td>{percent(p.probability)}</td>
                      <td>{percent(p.ev, true)}</td>
                      <td>{decimal(p.fair_odds)}</td>
                      <td>{p.model_version}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </section>
          <section className="panel">
            <h2>ประวัติคำแนะนำ</h2>
            {d.recommendation_snapshots.map((r) => (
              <p key={r.id}>
                <span className={`badge ${r.status.toLowerCase()}`}>
                  {r.status}
                </span>{" "}
                {dateTime(r.generated_at)} · {r.reasons[0]} · {r.model_version}
              </p>
            ))}
          </section>
          <section className="panel">
            <h2>ประวัติรายชื่อผู้เล่น</h2>
            {d.lineup_snapshots.map((l) => (
              <details key={l.id}>
                <summary>
                  {dateTime(l.captured_at)} · {l.phase}
                </summary>
                <p>
                  {d.home}: {l.home.starting_xi.join(" · ")}
                </p>
                <p>
                  {d.away}: {l.away.starting_xi.join(" · ")}
                </p>
              </details>
            ))}
          </section>
        </>
      )}
    </>
  );
}
