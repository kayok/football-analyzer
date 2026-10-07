"use client";
import { useEffect, useRef, useState } from "react";
import { request } from "../lib/api";
import { dateTime } from "../lib/format";

interface SyncStatus {
  state: string;
  message: string;
  last_success_at: string | null;
  retry_at: string | null;
  automatic: boolean;
  next_scheduled_at: string | null;
}
export function SyncControl({ onSynced }: { onSynced: () => void }) {
  const [status, setStatus] = useState<SyncStatus | null>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [now, setNow] = useState(0);
  const success = useRef<string | null | undefined>(undefined);
  useEffect(() => {
    const controller = new AbortController();
    async function poll() {
      try {
        const next = await request<SyncStatus>("/api/v1/sync", {
          signal: controller.signal,
        });
        if (controller.signal.aborted) return;
        if (
          success.current !== undefined &&
          next.last_success_at &&
          success.current !== next.last_success_at
        )
          onSynced();
        success.current = next.last_success_at;
        setStatus(next);
        setError("");
        setNow(Date.now());
      } catch (e) {
        if (!controller.signal.aborted) setError((e as Error).message);
      }
    }
    void poll();
    const timer = setInterval(() => void poll(), 5000);
    return () => {
      controller.abort();
      clearInterval(timer);
    };
  }, [onSynced]);
  async function sync() {
    setBusy(true);
    setError("");
    try {
      setStatus(await request<SyncStatus>("/api/v1/sync", { method: "POST" }));
      setNow(Date.now());
    } catch (e) {
      setError((e as Error).message);
    } finally {
      setBusy(false);
    }
  }
  const running = status?.state === "running";
  const cooling =
    !!status?.retry_at && new Date(status.retry_at).getTime() > now;
  return (
    <section className="notice sync-control" aria-label="ซิงก์ข้อมูลฟุตบอล">
      <div>
        <strong>{status?.message || "กำลังตรวจสถานะการซิงก์…"}</strong>
        <p>ระหว่างซิงก์จะแสดงข้อมูลเดิม ข้อมูลใหม่จะแสดงเมื่อบันทึกสำเร็จ</p>
        {status?.last_success_at && (
          <small>สำเร็จล่าสุด: {dateTime(status.last_success_at, true)}</small>
        )}
        <br />
        {status && (
          <small>
            {status.automatic && status.next_scheduled_at
              ? `ซิงก์อัตโนมัติรอบถัดไป: ${dateTime(status.next_scheduled_at, true)}`
              : status.automatic
                ? "เปิดซิงก์อัตโนมัติแล้ว"
                : "ซิงก์อัตโนมัติยังปิดอยู่"}
          </small>
        )}
        {!running && cooling && status?.retry_at && (
          <p>ซิงก์อีกครั้งได้หลัง {dateTime(status.retry_at)}</p>
        )}
        {error && (
          <p role="alert" className="error-text">
            {error}
          </p>
        )}
      </div>
      <button
        className="outline"
        disabled={!status || busy || running || cooling}
        onClick={() => void sync()}
      >
        {busy || running ? "กำลังซิงก์…" : "ซิงก์ข้อมูล"}
      </button>
    </section>
  );
}
