import { useEffect, useState } from "react";
import type { ReactNode } from "react";

export function ErrorBox({ error }: { error: string | null }) {
  if (!error) return null;
  return <div className="error">{error}</div>;
}

export function Empty({ children }: { children: ReactNode }) {
  return <div className="empty">{children}</div>;
}

export function fmtDate(iso: string | null): string {
  if (!iso) return "—";
  const d = new Date(iso);
  return d.toLocaleString("ru-RU", {
    day: "2-digit", month: "2-digit", year: "numeric", hour: "2-digit", minute: "2-digit",
  });
}

export function fmtNum(n: number): string {
  return new Intl.NumberFormat("ru-RU", { maximumFractionDigits: 1 }).format(n);
}

/** Приводит ошибку любого происхождения к строке для ErrorBox. */
export function msg(e: unknown): string {
  return e instanceof Error ? e.message : "Неизвестная ошибка";
}

/**
 * Длительность тренировки. Для незавершённой отсчёт идёт до `now`,
 * который приходит извне — иначе значение не обновлялось бы на экране.
 */
export function fmtDuration(startIso: string, endIso: string | null, now = Date.now()): string {
  const start = new Date(startIso).getTime();
  const end = endIso ? new Date(endIso).getTime() : now;
  const ms = end - start;
  if (!Number.isFinite(ms) || ms < 0) return "—";

  const totalMinutes = Math.floor(ms / 60000);
  if (totalMinutes < 1) return "меньше минуты";
  const hours = Math.floor(totalMinutes / 60);
  const minutes = totalMinutes % 60;
  if (hours === 0) return `${minutes} мин`;
  return `${hours} ч ${String(minutes).padStart(2, "0")} мин`;
}

/** Тикающее «сейчас». Нужен только там, где на экране есть идущая тренировка. */
export function useNow(active: boolean, intervalMs = 30_000): number {
  const [now, setNow] = useState(() => Date.now());
  useEffect(() => {
    if (!active) return;
    const id = setInterval(() => setNow(Date.now()), intervalMs);
    return () => clearInterval(id);
  }, [active, intervalMs]);
  return now;
}
