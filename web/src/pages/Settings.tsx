import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import type { AISettings, Connection, User } from "../lib/types";
import { api, ApiError } from "../lib/api";
import { PixelCard } from "../components/PixelCard";
import { PixelButton } from "../components/PixelButton";
import { PixelInput } from "../components/PixelInput";
import styles from "./Settings.module.css";

interface Props {
  user: User;
  connections: Connection[];
}

export function Settings({ user, connections }: Props) {
  const qc = useQueryClient();
  const [displayName, setDisplayName] = useState(user.displayName);
  const [avatarUrl, setAvatarUrl] = useState(user.avatarUrl);

  const steam = connections.find((c) => c.provider === "steam");
  const psn = connections.find((c) => c.provider === "psn");
  const [npsso, setNpsso] = useState("");

  const saveProfile = useMutation({
    mutationFn: () => api.updateProfile({ displayName, avatarUrl }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["me"] }),
  });

  const disconnect = useMutation({
    mutationFn: () => api.disconnectSteam(),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["me"] }),
  });

  const connectPsn = useMutation({
    mutationFn: () => api.connectPSN(npsso.trim()),
    onSuccess: () => {
      setNpsso("");
      qc.invalidateQueries({ queryKey: ["me"] });
    },
  });

  const disconnectPsn = useMutation({
    mutationFn: () => api.disconnectPSN(),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["me"] }),
  });

  const psnError = connectPsn.error instanceof ApiError ? connectPsn.error.message : null;

  const profileError =
    saveProfile.error instanceof ApiError ? saveProfile.error.message : null;
  const disconnectError =
    disconnect.error instanceof ApiError ? disconnect.error.message : null;

  return (
    <div className={styles.page}>
      <h2 className={styles.heading}>Settings</h2>

      <PixelCard className={styles.section}>
        <h3 className={styles.sectionTitle}>Profile</h3>
        <form
          className={styles.form}
          onSubmit={(e) => {
            e.preventDefault();
            saveProfile.mutate();
          }}
        >
          <div className={styles.readonly}>
            <span className={styles.k}>Username</span>
            <span className={styles.v}>{user.username}</span>
          </div>
          <div className={styles.readonly}>
            <span className={styles.k}>Email</span>
            <span className={styles.v}>{user.email || "—"}</span>
          </div>
          <div className={styles.readonly}>
            <span className={styles.k}>Role</span>
            <span className={styles.v}>{user.role}</span>
          </div>

          <PixelInput
            label="Display name"
            name="displayName"
            value={displayName}
            onChange={(e) => setDisplayName(e.target.value)}
            required
          />
          <PixelInput
            label="Avatar URL"
            name="avatarUrl"
            value={avatarUrl}
            placeholder="https://…"
            onChange={(e) => setAvatarUrl(e.target.value)}
          />

          {profileError && <p className={styles.error}>⚠ {profileError}</p>}
          {saveProfile.isSuccess && <p className={styles.ok}>✓ Saved</p>}

          <PixelButton variant="primary" type="submit" disabled={saveProfile.isPending}>
            {saveProfile.isPending ? "..." : "Save profile"}
          </PixelButton>
        </form>
      </PixelCard>

      <PixelCard className={styles.section}>
        <h3 className={styles.sectionTitle}>Connected accounts</h3>
        <div className={styles.connection}>
          <div>
            <span className={styles.provider}>▶ Steam</span>
            <span className={styles.connStatus}>
              {steam ? `linked · ${steam.externalId}` : "not connected"}
            </span>
          </div>
          {steam ? (
            <PixelButton onClick={() => disconnect.mutate()} disabled={disconnect.isPending}>
              Disconnect
            </PixelButton>
          ) : (
            <a href={api.steamLoginUrl}>
              <PixelButton variant="primary">Connect</PixelButton>
            </a>
          )}
        </div>
        {disconnectError && <p className={styles.error}>⚠ {disconnectError}</p>}
        <p className={styles.hint}>
          Steam is used to import your library. {steam ? "" : "Connect it to sync your games."}
        </p>

        <div className={styles.connection} style={{ marginTop: "var(--s-4)" }}>
          <div>
            <span className={styles.provider}>▶ PlayStation</span>
            <span className={styles.connStatus}>
              {psn ? `linked · ${psn.externalId}` : "not connected"}
            </span>
          </div>
          {psn && (
            <PixelButton onClick={() => disconnectPsn.mutate()} disabled={disconnectPsn.isPending}>
              Disconnect
            </PixelButton>
          )}
        </div>
        {!psn && (
          <form
            className={styles.form}
            onSubmit={(e) => {
              e.preventDefault();
              connectPsn.mutate();
            }}
          >
            <PixelInput
              label="NPSSO token"
              name="npsso"
              value={npsso}
              placeholder="paste your npsso…"
              onChange={(e) => setNpsso(e.target.value)}
            />
            {psnError && <p className={styles.error}>⚠ {psnError}</p>}
            <PixelButton variant="primary" type="submit" disabled={connectPsn.isPending || !npsso.trim()}>
              {connectPsn.isPending ? "Connecting…" : "Connect PlayStation"}
            </PixelButton>
          </form>
        )}
        <p className={styles.hint}>
          PlayStation has no official API. Log in at playstation.com, open{" "}
          <code>ca.account.sony.com/api/v1/ssocookie</code> and paste the <code>npsso</code> value.
          It expires every ~2 months.
        </p>
      </PixelCard>

      <PixelCard className={styles.section}>
        <h3 className={styles.sectionTitle}>AI provider</h3>
        <AISettingsForm />
      </PixelCard>
    </div>
  );
}

function AISettingsForm() {
  const qc = useQueryClient();
  const { data } = useQuery({ queryKey: ["ai-settings"], queryFn: api.getAISettings });
  if (!data) return <p className={styles.hint}>…</p>;
  return <AIForm initial={data} qc={qc} />;
}

function AIForm({ initial, qc }: { initial: AISettings; qc: ReturnType<typeof useQueryClient> }) {
  const [baseUrl, setBaseUrl] = useState(initial.baseUrl);
  const [model, setModel] = useState(initial.model);
  const [apiKey, setApiKey] = useState("");

  const save = useMutation({
    mutationFn: () => api.setAISettings({ baseUrl, model, apiKey }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["ai-settings"] }),
  });
  const error = save.error instanceof ApiError ? save.error.message : null;

  return (
    <form
      className={styles.form}
      onSubmit={(e) => {
        e.preventDefault();
        save.mutate();
      }}
    >
      <PixelInput
        label="Base URL"
        name="aiBaseUrl"
        value={baseUrl}
        placeholder="http://localhost:11434/v1"
        onChange={(e) => setBaseUrl(e.target.value)}
      />
      <PixelInput
        label="Model"
        name="aiModel"
        value={model}
        placeholder="llama3.2:3b"
        onChange={(e) => setModel(e.target.value)}
      />
      <PixelInput
        label="API key"
        name="aiKey"
        type="password"
        value={apiKey}
        placeholder={initial.hasKey ? "•••••• (blank = keep)" : "for Ollama: any value, e.g. ollama"}
        onChange={(e) => setApiKey(e.target.value)}
      />

      {error && <p className={styles.error}>⚠ {error}</p>}
      {save.isSuccess && <p className={styles.ok}>✓ Saved</p>}

      <PixelButton variant="primary" type="submit" disabled={save.isPending}>
        {save.isPending ? "..." : "Save AI"}
      </PixelButton>
      <p className={styles.hint}>
        Leave Base URL blank to use the server’s default AI. Your key is encrypted at rest.
      </p>
    </form>
  );
}
