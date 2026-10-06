"use client";
import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { useAuth } from "../../components/auth";
import { request } from "../../lib/api";
export default function LoginPage() {
  const [register, setRegister] = useState(false);
  const [name, setName] = useState(""),
    [email, setEmail] = useState(""),
    [password, setPassword] = useState(""),
    [confirm, setConfirm] = useState("");
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
    if (register && password !== confirm) {
      setError("รหัสผ่านทั้งสองช่องไม่ตรงกัน");
      return;
    }
    setBusy(true);
    try {
      await request(register ? "/api/v1/auth/register" : "/api/v1/auth/login", {
        method: "POST",
        body: JSON.stringify(
          register ? { name, email, password } : { email, password },
        ),
      });
      setPassword("");
      setConfirm("");
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
      <h1>{register ? "สมัครสมาชิก" : "เข้าสู่ระบบ"}</h1>
      <p className="muted">เก็บรายการที่เลือกและประวัติไว้ในบัญชีของคุณ</p>
      <form onSubmit={submit}>
        {register && (
          <>
            <label>
              ชื่อที่แสดง
              <input
                autoComplete="name"
                aria-describedby="register-name-help"
                required
                maxLength={80}
                value={name}
                onChange={(e) => setName(e.target.value)}
              />
            </label>
            <small id="register-name-help" className="muted">
              ชื่อที่แสดง 1–80 ตัวอักษร
            </small>
          </>
        )}
        <label>
          อีเมล
          <input
            type="email"
            autoComplete="email"
            aria-describedby={register ? "register-email-help" : undefined}
            required
            maxLength={254}
            value={email}
            onChange={(e) => setEmail(e.target.value)}
          />
        </label>
        {register && (
          <small id="register-email-help" className="muted">
            ใช้อีเมลรูปแบบถูกต้องและยังไม่เคยสมัคร เช่น name@example.com
          </small>
        )}
        <label>
          รหัสผ่าน
          <input
            type="password"
            autoComplete={register ? "new-password" : "current-password"}
            aria-describedby={register ? "register-password-help" : undefined}
            required
            minLength={register ? 9 : undefined}
            maxLength={register ? 72 : undefined}
            pattern={register ? "[\\x21-\\x7E]{9,72}" : undefined}
            title={
              register
                ? "ใช้ภาษาอังกฤษ ตัวเลข และอักขระพิเศษทั่วไป 9–72 ตัว โดยไม่มีช่องว่าง"
                : undefined
            }
            value={password}
            onChange={(e) => setPassword(e.target.value)}
          />
        </label>
        {register && (
          <>
            <small id="register-password-help" className="muted">
              ใช้ภาษาอังกฤษ A–Z, a–z ตัวเลข 0–9 และอักขระพิเศษทั่วไป เช่น ! @ #
              $ %
              <br />
              ความยาว 9–72 ตัวอักษร ไม่มีช่องว่าง
            </small>
            <label>
              ยืนยันรหัสผ่าน
              <input
                type="password"
                autoComplete="new-password"
                aria-describedby="register-confirm-help"
                required
                minLength={9}
                maxLength={72}
                value={confirm}
                onChange={(e) => setConfirm(e.target.value)}
              />
            </label>
            <small id="register-confirm-help" className="muted">
              กรอกรหัสผ่านให้ตรงกับช่องด้านบน
            </small>
            <p className="muted">
              รุ่นนี้ยังไม่มีระบบกู้รหัสผ่าน กรุณาจดจำรหัสผ่านที่ใช้สมัคร
            </p>
          </>
        )}
        {(error || authError) && (
          <p className="error-text" role="alert">
            {error || authError}
          </p>
        )}
        <button className="primary" type="submit" disabled={busy || loading}>
          {busy ? "กำลังบันทึก…" : register ? "สมัครสมาชิก" : "เข้าสู่ระบบ"}
        </button>
      </form>
      <button
        className="quiet auth-switch"
        disabled={busy}
        onClick={() => {
          setRegister(!register);
          setError("");
          setPassword("");
          setConfirm("");
        }}
      >
        {register ? "มีบัญชีแล้ว? เข้าสู่ระบบ" : "ยังไม่มีบัญชี? สมัครสมาชิก"}
      </button>
    </section>
  );
}
