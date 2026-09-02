import { createContext, useCallback, useContext, useEffect, useMemo, useState } from "react";
import type { ReactNode } from "react";
import { api, getToken, setToken, setUnauthorizedHandler } from "../api/client";

type Session = { token: string; expiresAt: string; email: string };

type AuthValue = {
  session: Session | null;
  login: (email: string, password: string) => Promise<void>;
  register: (b: {
    username: string; second_name: string; email: string; password: string;
  }) => Promise<void>;
  logout: () => void;
};

const SESSION_KEY = "trainingapp.session";
const REFRESH_MARGIN_MS = 5 * 60 * 1000;
const AuthContext = createContext<AuthValue | null>(null);

function loadSession(): Session | null {
  const raw = localStorage.getItem(SESSION_KEY);
  const token = getToken();
  if (!raw || !token) return null;
  try {
    const s = JSON.parse(raw) as Session;
    if (new Date(s.expiresAt).getTime() <= Date.now()) return null;
    return { ...s, token };
  } catch {
    return null;
  }
}

export function AuthProvider({ children }: { children: ReactNode }) {
  const [session, setSession] = useState<Session | null>(loadSession);

  const logout = useCallback(() => {
    setToken(null);
    localStorage.removeItem(SESSION_KEY);
    setSession(null);
  }, []);

  useEffect(() => {
    setUnauthorizedHandler(logout);
  }, [logout]);

  const refresh = useCallback(async () => {
    try {
      const res = await api.refresh();
      setToken(res.token);
      setSession((prev) => {
        if (!prev) return prev;
        const next: Session = { ...prev, token: res.token, expiresAt: res.expires_at };
        localStorage.setItem(SESSION_KEY, JSON.stringify(next));
        return next;
      });
    } catch {
      logout();
    }
  }, [logout]);

  useEffect(() => {
    if (!session) return;
    const ms = new Date(session.expiresAt).getTime() - Date.now() - REFRESH_MARGIN_MS;
    if (ms <= 0) { void refresh(); return; }
    const t = setTimeout(() => void refresh(), Math.min(ms, 2 ** 31 - 1));
    return () => clearTimeout(t);
  }, [session, refresh]);

  useEffect(() => {
    if (!session) return;
    const onVisible = () => {
      if (document.visibilityState !== "visible") return;
      if (new Date(session.expiresAt).getTime() - Date.now() < REFRESH_MARGIN_MS) void refresh();
    };
    document.addEventListener("visibilitychange", onVisible);
    return () => document.removeEventListener("visibilitychange", onVisible);
  }, [session, refresh]);

  const login = useCallback(async (email: string, password: string) => {
    const res = await api.login({ email, password });
    setToken(res.token);
    const s: Session = { token: res.token, expiresAt: res.expires_at, email };
    localStorage.setItem(SESSION_KEY, JSON.stringify(s));
    setSession(s);
  }, []);

  const register = useCallback(
    async (b: { username: string; second_name: string; email: string; password: string }) => {
      await api.register(b);
      await login(b.email, b.password);
    },
    [login],
  );

  const value = useMemo(
    () => ({ session, login, register, logout }),
    [session, login, register, logout],
  );
  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth(): AuthValue {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error("useAuth вне AuthProvider");
  return ctx;
}
