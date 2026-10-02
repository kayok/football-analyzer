"use client";
import { useCallback, useEffect, useState } from "react";
import { requestAll, timezone } from "../lib/api";
export function useResource<T>(path: string) {
  const [data, setData] = useState<T | null>(null);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(true);
  const [revision, setRevision] = useState(0);
  const reload = useCallback(() => {
    setLoading(true);
    setError("");
    setRevision((n) => n + 1);
  }, []);
  useEffect(() => {
    const controller = new AbortController();
    requestAll<T>(path, { signal: controller.signal })
      .then((value) => {
        setData(value);
        setError("");
        setLoading(false);
      })
      .catch((e: Error) => {
        if (!controller.signal.aborted) {
          setError(e.message);
          setLoading(false);
        }
      });
    return () => controller.abort();
  }, [path, revision]);
  return { data, error, loading, reload };
}
export function Feedback({
  error,
  loading,
  retry,
}: {
  error: string;
  loading: boolean;
  retry: () => void;
}) {
  if (error)
    return (
      <div className="notice error" role="alert">
        {error}
        <button onClick={retry}>ลองอีกครั้ง</button>
      </div>
    );
  if (loading)
    return (
      <div className="skeleton" role="status">
        กำลังโหลดข้อมูลการแข่งขัน…
      </div>
    );
  return null;
}

export function useTimezone() {
  const [tz, setTZ] = useState("Asia/Bangkok");
  useEffect(() => {
    const value = timezone();
    if (value !== "Asia/Bangkok") queueMicrotask(() => setTZ(value));
  }, []);
  return tz;
}
