import { useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import type { IGDBGame } from "../lib/types";
import { api } from "../lib/api";
import { DiscoverCard, type AddState } from "../components/DiscoverCard";
import { PixelButton } from "../components/PixelButton";
import styles from "./Calendar.module.css";

const PLATFORMS = ["All", "PC", "PS5", "Xbox Series", "Switch 2", "Switch"];
const WEEKDAYS = ["Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"];

const monthLabel = (unix: number) =>
  new Date(unix * 1000).toLocaleDateString("en-GB", { month: "long", year: "numeric", timeZone: "UTC" });
const dayLabel = (unix: number) =>
  new Date(unix * 1000).toLocaleDateString("en-GB", { day: "numeric", month: "short", timeZone: "UTC" });
const ymKey = (unix: number) => {
  const d = new Date(unix * 1000);
  return `${d.getUTCFullYear()}-${String(d.getUTCMonth() + 1).padStart(2, "0")}`;
};

const subtitle = (g: IGDBGame) => {
  const date = g.releaseDate ? dayLabel(g.releaseDate) : "TBA";
  const plats = (g.platforms ?? []).slice(0, 4).join(" · ") || "Platform TBA";
  return `${date}  ·  ${plats}`;
};

type Fam = "multi" | "playstation" | "xbox" | "nintendo" | "pc" | "other";

// family classifies a game by platform for colour coding; multi = several families.
function family(platforms: string[] | null | undefined): Fam {
  const fams = new Set<Fam>();
  for (const p of platforms ?? []) {
    if (/PS\d|PlayStation/.test(p)) fams.add("playstation");
    else if (/Xbox/.test(p)) fams.add("xbox");
    else if (/Switch/.test(p)) fams.add("nintendo");
    else if (p === "PC") fams.add("pc");
  }
  if (fams.size > 1) return "multi";
  if (fams.size === 1) return [...fams][0];
  return "other";
}

const LEGEND: { fam: Fam; label: string }[] = [
  { fam: "multi", label: "Multi" },
  { fam: "playstation", label: "PlayStation" },
  { fam: "xbox", label: "Xbox" },
  { fam: "nintendo", label: "Nintendo" },
  { fam: "pc", label: "PC" },
];

export function Calendar() {
  const qc = useQueryClient();
  const navigate = useNavigate();
  const [plat, setPlat] = useState("All");
  const [view, setView] = useState<"list" | "grid">("list");
  const [cursor, setCursor] = useState(0);
  const [added, setAdded] = useState<Set<number>>(new Set());
  const [addingId, setAddingId] = useState<number | null>(null);

  const { data, isLoading } = useQuery({
    queryKey: ["calendar"],
    queryFn: api.calendar,
    staleTime: 3.6e6,
  });

  async function add(igdbId: number) {
    setAddingId(igdbId);
    try {
      await api.addGame(igdbId, "wishlist");
      setAdded((s) => new Set(s).add(igdbId));
      qc.invalidateQueries({ queryKey: ["library"] });
    } finally {
      setAddingId(null);
    }
  }
  const stateOf = (id: number): AddState =>
    added.has(id) ? "added" : addingId === id ? "adding" : "idle";

  const groups = useMemo(() => {
    const games = (data?.games ?? []).filter(
      (g) => plat === "All" || (g.platforms ?? []).includes(plat),
    );
    const map = new Map<string, IGDBGame[]>();
    for (const g of games) {
      if (!g.releaseDate) continue;
      const key = monthLabel(g.releaseDate);
      if (!map.has(key)) map.set(key, []);
      map.get(key)!.push(g);
    }
    return [...map.entries()];
  }, [data, plat]);

  // One page per month, navigable with arrows or a month input.
  const months = useMemo(
    () => groups.map(([key, games]) => ({ key, games, ym: ymKey(games[0].releaseDate!) })),
    [groups],
  );
  const idx = Math.min(cursor, Math.max(0, months.length - 1));
  const current = months[idx];

  function jumpTo(v: string) {
    if (!v) return;
    let i = months.findIndex((m) => m.ym === v);
    if (i < 0) i = months.findIndex((m) => m.ym >= v); // snap to next available month
    setCursor(i < 0 ? months.length - 1 : i);
  }

  return (
    <section>
      <h2 className={styles.heading}>Release calendar</h2>
      <p className={styles.intro}>
        Anticipated games with a confirmed release date, soonest first. Add the ones you want to your
        wishlist.
      </p>

      <div className={styles.toolbar}>
        <div className={styles.filters}>
          {PLATFORMS.map((p) => (
            <PixelButton
              key={p}
              variant={plat === p ? "primary" : "default"}
              onClick={() => {
                setPlat(p);
                setCursor(0);
              }}
            >
              {p}
            </PixelButton>
          ))}
        </div>
        <div className={styles.views}>
          <PixelButton variant={view === "list" ? "primary" : "default"} onClick={() => setView("list")}>
            ▤ List
          </PixelButton>
          <PixelButton variant={view === "grid" ? "primary" : "default"} onClick={() => setView("grid")}>
            ▦ Calendar
          </PixelButton>
        </div>
      </div>

      {view === "grid" && (
        <div className={styles.legend}>
          {LEGEND.map((l) => (
            <span key={l.fam} className={styles.legendItem}>
              <span className={`${styles.dot} ${styles[l.fam]}`} />
              {l.label}
            </span>
          ))}
          <span className={styles.legendHint}>· click a game to see details</span>
        </div>
      )}

      {isLoading ? (
        <p className={styles.muted}>▚ loading calendar…</p>
      ) : !current ? (
        <p className={styles.muted}>No upcoming games for {plat}.</p>
      ) : (
        <>
          <div className={styles.pager}>
            <PixelButton disabled={idx <= 0} onClick={() => setCursor(idx - 1)}>
              ‹
            </PixelButton>
            <input
              type="month"
              className={styles.monthInput}
              value={current.ym}
              min={months[0].ym}
              max={months[months.length - 1].ym}
              onChange={(e) => jumpTo(e.target.value)}
            />
            <PixelButton disabled={idx >= months.length - 1} onClick={() => setCursor(idx + 1)}>
              ›
            </PixelButton>
          </div>

          <h3 className={styles.monthTitle}>
            {current.key} <span className={styles.count}>· {current.games.length}</span>
          </h3>

          {view === "list" ? (
            <div className={styles.grid}>
              {current.games.map((g) => (
                <DiscoverCard
                  key={g.igdbId}
                  title={g.name}
                  coverUrl={g.coverUrl}
                  subtitle={subtitle(g)}
                  state={stateOf(g.igdbId)}
                  onAdd={() => add(g.igdbId)}
                  onOpen={() => navigate(`/igdb/${g.igdbId}`)}
                />
              ))}
            </div>
          ) : (
            <MonthCalendar games={current.games} onOpen={(id) => navigate(`/igdb/${id}`)} />
          )}
        </>
      )}
    </section>
  );
}

function MonthCalendar({
  games,
  onOpen,
}: {
  games: IGDBGame[];
  onOpen: (id: number) => void;
}) {
  const first = new Date(games[0].releaseDate! * 1000);
  const year = first.getUTCFullYear();
  const month = first.getUTCMonth();
  const daysInMonth = new Date(year, month + 1, 0).getDate();
  // Monday-first offset for the 1st of the month.
  const offset = (new Date(year, month, 1).getDay() + 6) % 7;

  const byDay = new Map<number, IGDBGame[]>();
  for (const g of games) {
    const day = new Date(g.releaseDate! * 1000).getUTCDate();
    if (!byDay.has(day)) byDay.set(day, []);
    byDay.get(day)!.push(g);
  }

  const cells: (number | null)[] = [];
  for (let i = 0; i < offset; i++) cells.push(null);
  for (let d = 1; d <= daysInMonth; d++) cells.push(d);

  return (
    <div className={styles.month}>
      <div className={styles.calWrap}>
        <div className={styles.cal}>
          {WEEKDAYS.map((w) => (
            <div key={w} className={styles.weekday}>
              {w}
            </div>
          ))}
          {cells.map((d, i) => (
            <div key={i} className={`${styles.cell} ${d === null ? styles.blank : ""}`}>
              {d !== null && (
                <>
                  <span className={styles.dayNum}>{d}</span>
                  {(byDay.get(d) ?? []).map((g) => (
                    <button
                      key={g.igdbId}
                      className={`${styles.event} ${styles[family(g.platforms)]}`}
                      title={`${g.name} — ${(g.platforms ?? []).join(", ") || "Platform TBA"}`}
                      onClick={() => onOpen(g.igdbId)}
                    >
                      {g.name}
                    </button>
                  ))}
                </>
              )}
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}
