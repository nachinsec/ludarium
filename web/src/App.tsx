import { useQuery, useQueryClient } from "@tanstack/react-query";
import { NavLink, Navigate, Route, Routes } from "react-router-dom";
import { api } from "./lib/api";
import { Library } from "./pages/Library";
import { Discover } from "./pages/Discover";
import { Oracle } from "./pages/Oracle";
import { Stats } from "./pages/Stats";
import { Settings } from "./pages/Settings";
import { AuthPage } from "./pages/AuthPage";
import { PixelButton } from "./components/PixelButton";
import styles from "./App.module.css";

export function App() {
  const qc = useQueryClient();
  const { data, isLoading } = useQuery({ queryKey: ["me"], queryFn: api.me });

  if (isLoading) {
    return <div className={styles.loading}>▚ loading…</div>;
  }

  const user = data?.user ?? null;
  if (!user) {
    return <AuthPage />;
  }

  async function handleLogout() {
    await api.logout();
    qc.invalidateQueries({ queryKey: ["me"] });
  }

  const navLink = ({ isActive }: { isActive: boolean }) =>
    isActive ? `${styles.nav} ${styles.navActive}` : styles.nav;

  return (
    <div className={styles.shell}>
      <header className={styles.header}>
        <div className={styles.brand}>
          <span className={styles.star}>★</span>
          <h1 className={styles.logo}>LUDARIUM</h1>
        </div>

        <nav className={styles.navbar}>
          <NavLink to="/" className={navLink} end>
            Library
          </NavLink>
          <NavLink to="/discover" className={navLink}>
            Discover
          </NavLink>
          <NavLink to="/oracle" className={navLink}>
            Oracle
          </NavLink>
          <NavLink to="/stats" className={navLink}>
            Stats
          </NavLink>
          <NavLink to="/settings" className={navLink}>
            Settings
          </NavLink>
        </nav>

        <div className={styles.account}>
          {user.avatarUrl && <img className={styles.avatar} src={user.avatarUrl} alt="" />}
          <span className={styles.name}>{user.displayName}</span>
          <PixelButton variant="ghost" onClick={handleLogout}>
            Logout
          </PixelButton>
        </div>
      </header>

      <main className={styles.main}>
        <Routes>
          <Route path="/" element={<Library />} />
          <Route path="/discover" element={<Discover />} />
          <Route path="/oracle" element={<Oracle />} />
          <Route path="/stats" element={<Stats />} />
          <Route
            path="/settings"
            element={<Settings user={user} connections={data?.connections ?? []} />}
          />
          <Route path="*" element={<Navigate to="/" replace />} />
        </Routes>
      </main>

      <footer className={styles.footer}>
        ▶ self-hosted · bring your own keys · {new Date().getFullYear()}
      </footer>
    </div>
  );
}
