"use client";
import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { useAuth } from "../../components/auth";
import { request } from "../../lib/api";
export default function LoginPage() {
  const [email, setEmail] = useState(""),
    [password, setPassword] = useState("");
  const [busy, setBusy] = useState(false),
    [error, setError] = useState("");
  const { user, loading, refresh, error: authError } = useAuth();
  const router = useRouter();
  useEffect(() => {
    if (user) router.replace("/");
  }, [user, router]);
  async function submit(e: React.FormEvent<HTMLFormElement>) {
    e.preventDefault();
    setError("");
    setBusy(true);
    try {
      await request("/api/v1/auth/login", {
        method: "POST",
        body: JSON.stringify({ email, password }),
      });
      setPassword("");
      await refresh();
    } catch (e) {
      setError((e as Error).message);
    } finally {
      setBusy(false);
    }
  }
  return (
    <section className="panel auth-panel">
      <span className="eyebrow">YOUR FOOTBALL JOURNAL</span>
      <h1>เข้าสู่ระบบ</h1>
      <p className="muted">ระบบส่วนตัวสำหรับเจ้าของบัญชีเท่านั้น ไม่เปิดรับสมัครสมาชิก</p>
      <form onSubmit={submit}>
        <label>
          อีเมล
          <input
            type="email"
            autoComplete="email"
            required
            maxLength={254}
            value={email}
            onChange={(e) => setEmail(e.target.value)}
          />
        </label>
        <label>
          รหัสผ่าน
          <input
            type="password"
            autoComplete="current-password"
            required
            value={password}
            onChange={(e) => setPassword(e.target.value)}
          />
        </label>
        {(error || authError) && (
          <p className="error-text" role="alert">
            {error || authError}
          </p>
        )}
        <button className="primary" type="submit" disabled={busy || loading}>
          {busy ? "กำลังเข้าสู่ระบบ…" : "เข้าสู่ระบบ"}
        </button>
      </form>
    </section>
  );
}
