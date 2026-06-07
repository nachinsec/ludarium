import { useState } from "react";
import { Link, useParams } from "react-router-dom";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import type { GameStatus, PublicProfile } from "../lib/types";
import { api } from "../lib/api";
import { GameCard } from "../components/GameCard";
import { PixelButton } from "../components/PixelButton";
import styles from "./Profile.module.css";

type Filter = "all" | GameStatus;
const FILTERS: { key: Filter; label: string }[] = [
  { key: "all", label: "All" },
  { key: "playing", label: "Playing" },
  { key: "completed", label: "Cleared" },
  { key: "backlog", label: "Backlog" },
  { key: "wishlist", label: "Wishlist" },
];

export function Profile() {
  const { id } = useParams();
  const userId = Number(id);
  const { data, isLoading } = useQuery({
    queryKey: ["profile", userId],
    queryFn: () => api.profile(userId),
    enabled: Number.isFinite(userId),
  });

  if (isLoading) return <p className={styles.muted}>▚ loading…</p>;
  if (!data) return <p className={styles.muted}>User not found.</p>;
  return <ProfileView profile={data.profile} />;
}

function ProfileView({ profile }: { profile: PublicProfile }) {
  const qc = useQueryClient();
  const [filter, setFilter] = useState<Filter>("all");

  const toggle = useMutation({
    mutationFn: () => (profile.isFollowing ? api.unfollow(profile.id) : api.follow(profile.id)),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["profile", profile.id] }),
  });

  const lib = useQuery({
    queryKey: ["user-library", profile.id],
    queryFn: () => api.userLibrary(profile.id),
    enabled: profile.canView,
  });
  const stats = useQuery({
    queryKey: ["user-stats", profile.id],
    queryFn: () => api.userStats(profile.id),
    enabled: profile.canView,
  });

  const games = lib.data?.games ?? [];
  const visible = filter === "all" ? games : games.filter((g) => g.status === filter);
  const s = stats.data?.stats;

  return (
    <section>
      <Link to="/members" className={styles.back}>
        ← Members
      </Link>

      <header className={styles.header}>
        <div className={styles.avatar}>
          {profile.avatarUrl ? (
            <img src={profile.avatarUrl} alt="" />
          ) : (
            <span>{profile.displayName.charAt(0) || "?"}</span>
          )}
        </div>
        <div className={styles.identity}>
          <h2 className={styles.name}>{profile.displayName}</h2>
          <span className={styles.username}>@{profile.username}</span>
          <div className={styles.counts}>
            <span>
              <b>{profile.gameCount}</b> games
            </span>
            <span>
              <b>{profile.followers}</b> followers
            </span>
            <span>
              <b>{profile.following}</b> following
            </span>
          </div>
        </div>
        {!profile.isSelf && (
          <PixelButton
            variant={profile.isFollowing ? "default" : "primary"}
            disabled={toggle.isPending}
            onClick={() => toggle.mutate()}
          >
            {profile.isFollowing ? "✓ Following" : "+ Follow"}
          </PixelButton>
        )}
      </header>

      {!profile.canView ? (
        <div className={styles.private}>
          <p className={styles.lock}>🔒</p>
          <p>This profile is private.</p>
        </div>
      ) : (
        <>
          {s && (
            <div className={styles.statStrip}>
              <Stat label="Games" value={s.totalGames} />
              <Stat label="Hours" value={Math.round(s.totalHours)} />
              <Stat label="Avg rating" value={s.avgRating ? `${s.avgRating.toFixed(1)}★` : "—"} />
            </div>
          )}

          <div className={styles.filters}>
            {FILTERS.map((f) => (
              <PixelButton
                key={f.key}
                variant={filter === f.key ? "primary" : "default"}
                onClick={() => setFilter(f.key)}
              >
                {f.label}
              </PixelButton>
            ))}
          </div>

          {lib.isLoading ? (
            <p className={styles.muted}>▚ loading library…</p>
          ) : visible.length === 0 ? (
            <p className={styles.muted}>No games here.</p>
          ) : (
            <div className={styles.grid}>
              {visible.map((g) => (
                <GameCard key={g.id} game={g} />
              ))}
            </div>
          )}
        </>
      )}
    </section>
  );
}

function Stat({ label, value }: { label: string; value: number | string }) {
  return (
    <div className={styles.stat}>
      <span className={styles.statValue}>{value}</span>
      <span className={styles.statLabel}>{label}</span>
    </div>
  );
}
