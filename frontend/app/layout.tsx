import type { Metadata } from "next";
import Link from "next/link";
import "./globals.css";
export const metadata: Metadata = {
  title: "สนาม · Football Analyzer",
  description: "แดชบอร์ดวิเคราะห์ฟุตบอลส่วนตัว",
};
export default function Layout({
  children,
}: Readonly<{ children: React.ReactNode }>) {
  return (
    <html lang="th">
      <body>
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
            <div className="sidebar-foot">
              <span className="dot" /> โหมดข้อมูลจำลอง
              <small>
                Poisson · v1-mock-poisson
                <br />
                เดิมพันจำลอง 1 unit ต่อรายการ
              </small>
            </div>
          </aside>
          <main>
            {children}
            <footer>
              ข้อมูลทั้งหมดเป็นข้อมูลจำลองสำหรับทดสอบระบบ ·
              คุณเป็นผู้ตัดสินใจเลือกเอง
            </footer>
          </main>
        </div>
      </body>
    </html>
  );
}
