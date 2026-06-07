import { useRef, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import type { AISettings, Connection, User } from "../lib/types";
import { api, ApiError } from "../lib/api";
import { PixelCard } from "../components/PixelCard";
import { PixelButton } from "../components/PixelButton";
import { PixelInput } from "../components/PixelInput";
import { SteamIcon, PlayStationIcon } from "../components/icons";
import styles from "./Settings.module.css";

interface Props {
  user: User;
  connections: Connection[];
}

// Resize an uploaded image to a square 128px data URL — keeps the avatar tiny
// enough to store inline in the DB, no upload endpoint needed.
function fileToAvatar(file: File): Promise<string> {
  return new Promise((resolve, reject) => {
    const reader = new FileReader();
    reader.onerror = () => reject(reader.error);
    reader.onload = () => {
      const img = new Image();
      img.onerror = reject;
      img.onload = () => {
        const size = 128;
        const canvas = document.createElement("canvas");
        canvas.width = canvas.height = size;
        const ctx = canvas.getContext("2d")!;
        const scale = Math.max(size / img.width, size / img.height);
        const w = img.width * scale;
        const h = img.height * scale;
        ctx.drawImage(img, (size - w) / 2, (size - h) / 2, w, h);
        resolve(canvas.toDataURL("image/jpeg", 0.85));
      };
      img.src = reader.result as string;
    };
    reader.readAsDataURL(file);
  });
}

export function Settings({ user, connections }: Props) {
  const qc = useQueryClient();
  const [displayName, setDisplayName] = useState(user.displayName);
  const [avatarUrl, setAvatarUrl] = useState(user.avatarUrl);
  const fileRef = useRef<HTMLInputElement>(null);

  async function onPickFile(e: React.ChangeEvent<HTMLInputElement>) {
    const file = e.target.files?.[0];
    e.target.value = ""; // allow re-picking the same file
    if (file) setAvatarUrl(await fileToAvatar(file));
  }

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
          <div className={styles.avatarRow}>
            <button
              type="button"
              className={styles.avatarPreview}
              onClick={() => fileRef.current?.click()}
              title="Upload an image"
            >
              {avatarUrl ? <img src={avatarUrl} alt="" /> : <span>{displayName.charAt(0) || "?"}</span>}
              <span className={styles.avatarEdit}>change</span>
            </button>
            <input ref={fileRef} type="file" accept="image/*" hidden onChange={onPickFile} />
            <div className={styles.avatarFields}>
              <PixelButton type="button" onClick={() => fileRef.current?.click()}>
                Upload image
              </PixelButton>
              <PixelInput
                label="…or paste an image URL"
                name="avatarUrl"
                value={avatarUrl.startsWith("data:") ? "" : avatarUrl}
                placeholder="https://…"
                onChange={(e) => setAvatarUrl(e.target.value)}
              />
            </div>
          </div>

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
            <span className={styles.provider}>
              <SteamIcon /> Steam
            </span>
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
            <span className={styles.provider}>
              <PlayStationIcon /> PlayStation
            </span>
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
