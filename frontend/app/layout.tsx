import type { Metadata } from "next";
import Link from "next/link";
import "./globals.css";
import { AuthProvider, AuthGate, MemberMenu } from "../components/auth";
export const metadata: Metadata = {
  title: "สนาม · Football Analyzer",
  description: "แดชบอร์ดวิเคราะห์ฟุตบอลส่วนตัว",
};
export default function Layout({
  children,
}: Readonly<{ children: React.ReactNode }>) {
  const realData = process.env.FOOTBALL_PROVIDER === "api-football";
  return (
    <html lang="th">
      <body>
        <AuthProvider>
          <div className="app-shell">
            <aside className="sidebar">
              <Link href="/" className="brand">
                <span className="brand-mark">◈</span>
                <span>
                  สนาม<small>FOOTBALL ANALYZER</small>
                </span>
              </Link>
              <div className="nav-label">แดชบอร์ดส่วนตัว</div>
              <nav>
                <Link href="/">
                  ◷ <span>วันนี้</span>
                </Link>
                <Link href="/picks">
                  ▤ <span>รายการที่เลือก</span>
                </Link>
                <Link href="/history">
                  ↺ <span>ประวัติและผลลัพธ์</span>
                </Link>
              </nav>
              <MemberMenu />
              <div className="sidebar-foot">
                <span className="dot" />{" "}
                {realData ? "API-Football · ข้อมูลจริง" : "โหมดข้อมูลจำลอง"}
                <small>
                  {realData
                    ? "Poisson · ผลย้อนหลัง"
                    : "Poisson · v1-mock-poisson"}
                  <br />
                  เดิมพันจำลอง 1 unit ต่อรายการ
                </small>
              </div>
            </aside>
            <main>
              <AuthGate>{children}</AuthGate>
              <footer>
                {realData
                  ? "ข้อมูลจาก API-Football · โมเดลพื้นฐานจากผลย้อนหลัง"
                  : "ข้อมูลทั้งหมดเป็นข้อมูลจำลองสำหรับทดสอบระบบ"}{" "}
                · คุณเป็นผู้ตัดสินใจเลือกเอง
              </footer>
            </main>
          </div>
        </AuthProvider>
      </body>
    </html>
  );
}
