import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Link } from "react-router-dom";
import type { FeedItem, UserCard } from "../lib/types";
import { api } from "../lib/api";
import { PixelCard } from "../components/PixelCard";
import { PixelButton } from "../components/PixelButton";
import { StatusBadge } from "../components/StatusBadge";
import styles from "./Members.module.css";

type Tab = "activity" | "everyone";

// SQLite stores 'YYYY-MM-DD HH:MM:SS' in UTC; render a short relative label.
function ago(ts: string): string {
  const then = new Date(ts.replace(" ", "T") + "Z").getTime();
  const s = Math.max(0, Math.floor((Date.now() - then) / 1000));
  if (s < 60) return "just now";
  const m = Math.floor(s / 60);
  if (m < 60) return `${m}m ago`;
  const h = Math.floor(m / 60);
  if (h < 24) return `${h}h ago`;
  const d = Math.floor(h / 24);
  if (d < 30) return `${d}d ago`;
  return `${Math.floor(d / 30)}mo ago`;
}

export function Members() {
  const [tab, setTab] = useState<Tab>("activity");

  return (
    <section>
      <h2 className={styles.heading}>Members</h2>
      <div className={styles.tabs}>
        <PixelButton
          variant={tab === "activity" ? "primary" : "default"}
          onClick={() => setTab("activity")}
        >
          Activity
        </PixelButton>
        <PixelButton
          variant={tab === "everyone" ? "primary" : "default"}
          onClick={() => setTab("everyone")}
        >
          Everyone
        </PixelButton>
      </div>

      {tab === "activity" ? <Feed /> : <Directory />}
    </section>
  );
}

function Feed() {
  const { data, isLoading } = useQuery({ queryKey: ["feed"], queryFn: api.feed });
  if (isLoading) return <p className={styles.muted}>▚ loading activity…</p>;
  const items = data?.items ?? [];

  if (items.length === 0) {
    return (
      <p className={styles.muted}>
        No activity yet. Follow people in <b>Everyone</b> to see what they play.
      </p>
    );
  }

  return (
    <div className={styles.feed}>
      {items.map((it, i) => (
        <FeedRow key={i} item={it} />
      ))}
    </div>
  );
}

function FeedRow({ item }: { item: FeedItem }) {
  return (
    <PixelCard className={styles.feedRow}>
      <div className={styles.cover}>
        {item.coverUrl && <img src={item.coverUrl} alt="" loading="lazy" />}
      </div>
      <div className={styles.feedBody}>
        <p className={styles.feedText}>
          <Link to={`/u/${item.userId}`} className={styles.actor}>
            {item.displayName}
          </Link>{" "}
          added <b>{item.title}</b>
        </p>
        <div className={styles.feedMeta}>
          <StatusBadge status={item.status} />
          <span className={styles.time}>{ago(item.at)}</span>
        </div>
      </div>
    </PixelCard>
  );
}

function Directory() {
  const { data, isLoading } = useQuery({ queryKey: ["users"], queryFn: api.users });
  if (isLoading) return <p className={styles.muted}>▚ loading members…</p>;
  const users = data?.users ?? [];

  if (users.length === 0) {
    return <p className={styles.muted}>You're the only member here so far.</p>;
  }

  return (
    <div className={styles.grid}>
      {users.map((u) => (
        <MemberCard key={u.id} user={u} />
      ))}
    </div>
  );
}

function MemberCard({ user }: { user: UserCard }) {
  const qc = useQueryClient();
  const toggle = useMutation({
    mutationFn: () => (user.isFollowing ? api.unfollow(user.id) : api.follow(user.id)),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["users"] });
      qc.invalidateQueries({ queryKey: ["feed"] });
    },
  });

  return (
    <PixelCard className={styles.card}>
      <Link to={`/u/${user.id}`} className={styles.link}>
        <div className={styles.avatar}>
          {user.avatarUrl ? <img src={user.avatarUrl} alt="" /> : <span>{user.displayName.charAt(0) || "?"}</span>}
        </div>
        <div className={styles.who}>
          <span className={styles.name}>{user.displayName}</span>
          <span className={styles.sub}>
            {user.visibility === "private" ? "🔒 private" : `${user.gameCount} games`}
          </span>
        </div>
      </Link>
      <PixelButton
        variant={user.isFollowing ? "default" : "primary"}
        disabled={toggle.isPending}
        onClick={() => toggle.mutate()}
      >
        {user.isFollowing ? "✓ Following" : "+ Follow"}
      </PixelButton>
    </PixelCard>
  );
}
