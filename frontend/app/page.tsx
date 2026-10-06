"use client";
import { useState } from "react";
import { useResource, Feedback, useTimezone } from "../components/resource";
import { RecommendationCard } from "../components/recommendation-card";
import type { Card, Page } from "../lib/types";
export default function Today() {
  const tz = useTimezone();
  const [filter, setFilter] = useState("ALL");
  const { data, error, loading, reload } = useResource<Page<Card>>(
    `/api/v1/matches/today?timezone=${encodeURIComponent(tz)}&limit=200`,
  );
  const cards = data?.items || [],
    shown =
      filter === "ALL"
        ? cards
        : cards.filter((c) => c.recommendation?.status === filter);
  return (
    <>
      <header className="page-header">
        <div>
          <span className="eyebrow">MATCHDAY OVERVIEW</span>
          <h1>
            สนามวันนี้
            <span className="dot" />
          </h1>
          <p>มองเกมให้ชัด ก่อนตัดสินใจเลือก</p>
        </div>
        <button className="outline" onClick={reload}>
          ↻ โหลดข้อมูลใหม่
        </button>
      </header>
      <div className="summary">
        <div>
          <span>คู่แข่งขันวันนี้</span>
          <strong>{cards.length.toString().padStart(2, "0")}</strong>
        </div>
        <div>
          <span>ราคาที่คุ้ม · PLAY</span>
          <strong className="positive">
            {cards
              .filter((c) => c.recommendation?.status === "PLAY")
              .length.toString()
              .padStart(2, "0")}
          </strong>
        </div>
        <div>
          <span>ติดตาม · WATCH</span>
          <strong className="amber">
            {cards
              .filter((c) => c.recommendation?.status === "WATCH")
              .length.toString()
              .padStart(2, "0")}
          </strong>
        </div>
        <div>
          <span>รายการที่เลือกวันนี้</span>
          <strong>
            {cards
              .filter((c) => c.pick)
              .length.toString()
              .padStart(2, "0")}
          </strong>
        </div>
      </div>
      <div className="section-top">
        <div className="tabs">
          {["ALL", "PLAY", "WATCH", "PASS"].map((f) => (
            <button
              key={f}
              className={filter === f ? "active" : ""}
              onClick={() => setFilter(f)}
            >
              {f === "ALL" ? "ทั้งหมด" : f}
            </button>
          ))}
        </div>
        <span className="muted">
          {tz} ·{" "}
          {cards.some((c) => c.provider_name === "api-football")
            ? "API-Football · ข้อมูลจริง"
            : cards.length
              ? "ข้อมูลจำลอง"
              : "ยังไม่มีข้อมูลคู่แข่งขัน"}
        </span>
      </div>
      <Feedback error={error} loading={loading} retry={reload} />
      {!loading && !error && shown.length === 0 && (
        <div className="empty">ไม่มีคู่แข่งขันในรายการนี้</div>
      )}
      <div className="cards-grid">
        {shown.map((card) => (
          <RecommendationCard key={card.id} card={card} onPicked={reload} />
        ))}
      </div>
      <div className="notice">
        PLAY = EV ตั้งแต่ 5% · WATCH = EV เป็นบวกแต่ต่ำกว่า 5% · PASS =
        ยังไม่คุ้มหรือข้อมูลใช้ไม่ได้
        <br />
        <small>
          ราคามีอายุไม่เกิน 15 นาที กรุณารัน worker เพื่อซิงก์ข้อมูล
          ราคาต้นทางอาจยังเก่าเกินเกณฑ์แม้เพิ่งซิงก์
        </small>
      </div>
    </>
  );
}
