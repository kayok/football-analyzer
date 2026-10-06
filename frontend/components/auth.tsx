"use client";
import Link from "next/link";
import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useState,
} from "react";
import { usePathname, useRouter } from "next/navigation";
import { APIError, request } from "../lib/api";
export interface Member {
  id: string;
  name: string;
  email: string;
  created_at: string;
}
interface Auth {
  user: Member | null;
  loading: boolean;
  error: string;
  refresh: () => Promise<void>;
  logout: () => Promise<void>;
}
const AuthContext = createContext<Auth | null>(null);
export function useAuth() {
  const auth = useContext(AuthContext);
  if (!auth) throw new Error("AuthProvider is required");
  return auth;
}
export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [user, setUser] = useState<Member | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const router = useRouter();
  const refresh = useCallback(async () => {
    try {
      setUser(await request<Member>("/api/v1/auth/me"));
      setError("");
    } catch (e) {
      setUser(null);
      setError(
        e instanceof APIError && e.status === 401 ? "" : (e as Error).message,
      );
    } finally {
      setLoading(false);
    }
  }, []);
  useEffect(() => {
    const controller = new AbortController();
    request<Member>("/api/v1/auth/me", { signal: controller.signal })
      .then((member) => {
        if (!controller.signal.aborted) setUser(member);
      })
      .catch((e: Error) => {
        if (
          !controller.signal.aborted &&
          !(e instanceof APIError && e.status === 401)
        )
          setError(e.message);
      })
      .finally(() => {
        if (!controller.signal.aborted) setLoading(false);
      });
    return () => controller.abort();
  }, []);
  // Expiry during any API operation hides cached personal data.
  useEffect(() => {
    const expired = () => {
      setUser(null);
      router.replace("/login");
    };
    window.addEventListener("auth-expired", expired);
    return () => window.removeEventListener("auth-expired", expired);
  }, [router]);
  async function logout() {
    await request<void>("/api/v1/auth/logout", { method: "POST" });
    setUser(null);
    router.replace("/login");
  }
  return (
    <AuthContext.Provider value={{ user, loading, error, refresh, logout }}>
      {children}
    </AuthContext.Provider>
  );
}
export function AuthGate({ children }: { children: React.ReactNode }) {
  const { user, loading, error, refresh } = useAuth();
  const pathname = usePathname();
  const router = useRouter();
  useEffect(() => {
    if (!loading && !error && !user && pathname !== "/login")
      router.replace("/login");
  }, [user, loading, error, pathname, router]);
  if (pathname === "/login") return children;
  if (loading)
    return (
      <div className="skeleton" role="status">
        กำลังตรวจสอบบัญชี…
      </div>
    );
  if (error)
    return (
      <div className="notice error" role="alert">
        {error}
        <button onClick={() => void refresh()}>ลองอีกครั้ง</button>
      </div>
    );
  return user ? <div key={user.id}>{children}</div> : null;
}
export function MemberMenu() {
  const { user, loading, logout } = useAuth();
  const [busy, setBusy] = useState(false),
    [error, setError] = useState("");
  async function leave() {
    setBusy(true);
    setError("");
    try {
      await logout();
    } catch (e) {
      setError((e as Error).message);
    } finally {
      setBusy(false);
    }
  }
  return (
    <div className="member-menu">
      {user ? (
        <>
          <strong>{user.name}</strong>
          <small>{user.email}</small>
          <button className="outline" disabled={busy} onClick={leave}>
            {busy ? "กำลังออกจากระบบ…" : "ออกจากระบบ"}
          </button>
        </>
      ) : (
        !loading && <Link href="/login">เข้าสู่ระบบ / สมัครสมาชิก</Link>
      )}
      {error && (
        <p role="alert" className="error-text">
          {error}
        </p>
      )}
    </div>
  );
}
