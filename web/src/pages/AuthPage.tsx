import { useState } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { api, ApiError } from "../lib/api";
import { PixelButton } from "../components/PixelButton";
import { PixelInput } from "../components/PixelInput";
import { PixelCard } from "../components/PixelCard";
import styles from "./AuthPage.module.css";

type Mode = "login" | "register";

export function AuthPage() {
  const [mode, setMode] = useState<Mode>("login");
  const qc = useQueryClient();

  const [login, setLogin] = useState("");
  const [username, setUsername] = useState("");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");

  const mutation = useMutation({
    mutationFn: async () => {
      if (mode === "login") return api.login({ login, password });
      return api.register({ username, email, password });
    },
    onSuccess: () => qc.invalidateQueries({ queryKey: ["me"] }),
  });

  const error = mutation.error instanceof ApiError ? mutation.error.message : null;

  return (
    <div className={styles.wrap}>
      <PixelCard className={styles.card}>
        <div className={styles.brand}>
          <span className={styles.star}>★</span>
          <h1 className={styles.logo}>LUDARIUM</h1>
        </div>
        <p className={styles.tagline}>Your self-hosted game library.</p>

        <div className={styles.tabs}>
          <PixelButton
            variant={mode === "login" ? "primary" : "default"}
            onClick={() => setMode("login")}
          >
            Login
          </PixelButton>
          <PixelButton
            variant={mode === "register" ? "primary" : "default"}
            onClick={() => setMode("register")}
          >
            Register
          </PixelButton>
        </div>

        <form
          className={styles.form}
          onSubmit={(e) => {
            e.preventDefault();
            mutation.mutate();
          }}
        >
          {mode === "login" ? (
            <PixelInput
              label="Username or email"
              name="login"
              value={login}
              onChange={(e) => setLogin(e.target.value)}
              autoComplete="username"
              required
            />
          ) : (
            <>
              <PixelInput
                label="Username"
                name="username"
                value={username}
                onChange={(e) => setUsername(e.target.value)}
                autoComplete="username"
                required
              />
              <PixelInput
                label="Email (optional)"
                name="email"
                type="email"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                autoComplete="email"
              />
            </>
          )}

          <PixelInput
            label="Password"
            name="password"
            type="password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            autoComplete={mode === "login" ? "current-password" : "new-password"}
            required
          />

          {error && <p className={styles.error}>⚠ {error}</p>}

          <PixelButton variant="primary" type="submit" disabled={mutation.isPending}>
            {mutation.isPending ? "..." : mode === "login" ? "▶ Enter" : "▶ Create account"}
          </PixelButton>
        </form>

        <div className={styles.divider}>
          <span>or</span>
        </div>

        <a href={api.steamLoginUrl} className={styles.steamLink}>
          <PixelButton className={styles.steamBtn}>▶ Continue with Steam</PixelButton>
        </a>
      </PixelCard>
    </div>
  );
}
