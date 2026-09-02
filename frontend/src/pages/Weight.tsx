import { useCallback, useEffect, useMemo, useState } from "react";
import { api } from "../api/client";
import type { BodyWeight } from "../api/types";
import { WeightChart } from "../components/WeightChart";
import { Empty, ErrorBox, fmtNum, msg } from "../components/ui";

function isoDay(d: Date): string {
  return d.toISOString().slice(0, 10);
}

export function Weight() {
  const [items, setItems] = useState<BodyWeight[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const [value, setValue] = useState("");
  const [day, setDay] = useState(isoDay(new Date()));
  const [note, setNote] = useState("");
  const [saving, setSaving] = useState(false);

  const load = useCallback(async () => {
    try {
      setItems(await api.weights());
      setError(null);
    } catch (e) {
      setError(msg(e));
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => { void load(); }, [load]);

  const summary = useMemo(() => {
    if (items.length === 0) return null;
    const sorted = [...items].sort(
      (a, b) => new Date(a.measured_on).getTime() - new Date(b.measured_on).getTime(),
    );
    const first = sorted[0].weight_kg;
    const last = sorted[sorted.length - 1].weight_kg;
    const values = sorted.map((w) => w.weight_kg);
    return {
      current: last,
      delta: last - first,
      min: Math.min(...values),
      max: Math.max(...values),
      count: sorted.length,
    };
  }, [items]);

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    setSaving(true);
    try {
      await api.createWeight({
        weight_kg: Number(value),
        measured_on: day,
        note: note.trim(),
      });
      setValue("");
      setNote("");
      setError(null);
      await load();
    } catch (err) {
      setError(msg(err));
    } finally {
      setSaving(false);
    }
  }

  async function remove(id: number) {
    try {
      await api.deleteWeight(id);
      await load();
    } catch (err) {
      setError(msg(err));
    }
  }

  return (
    <>
      <div className="page-head"><h1>Вес</h1></div>
      <ErrorBox error={error} />

      {summary && (
        <div className="grid cols-4">
          <div className="card">
            <div className="stat-label">Сейчас</div>
            <div className="stat-value">{fmtNum(summary.current)} <span className="unit">кг</span></div>
          </div>
          <div className="card">
            <div className="stat-label">Изменение</div>
            <div className={`stat-value ${summary.delta > 0 ? "up" : "down"}`}>
              {summary.delta > 0 ? "+" : ""}{fmtNum(summary.delta)} <span className="unit">кг</span>
            </div>
          </div>
          <div className="card">
            <div className="stat-label">Минимум</div>
            <div className="stat-value" style={{ fontSize: 24 }}>{fmtNum(summary.min)}</div>
          </div>
          <div className="card">
            <div className="stat-label">Максимум</div>
            <div className="stat-value" style={{ fontSize: 24 }}>{fmtNum(summary.max)}</div>
          </div>
        </div>
      )}

      <div className="card" style={{ marginTop: 14 }}>
        <h2>Динамика</h2>
        {loading ? <Empty>Загрузка…</Empty> : <WeightChart items={items} />}
      </div>

      <form className="card row wrap end" onSubmit={submit}>
        <div style={{ flex: 1, minWidth: 120 }}>
          <label htmlFor="w-value">Вес, кг</label>
          <input id="w-value" type="number" min={20} max={500} step="0.1" required
                 value={value} onChange={(e) => setValue(e.target.value)} />
        </div>
        <div style={{ flex: 1, minWidth: 150 }}>
          <label htmlFor="w-day">Дата</label>
          <input id="w-day" type="date" value={day} max={isoDay(new Date())}
                 onChange={(e) => setDay(e.target.value)} />
        </div>
        <div style={{ flex: 2, minWidth: 180 }}>
          <label htmlFor="w-note">Заметка</label>
          <input id="w-note" maxLength={200} value={note}
                 onChange={(e) => setNote(e.target.value)} />
        </div>
        <button className="btn-primary" disabled={saving}>
          {saving ? "Сохраняем…" : "Записать"}
        </button>
      </form>

      {items.length > 0 && (
        <div className="card">
          <h2>Замеры</h2>
          <table>
            <thead>
              <tr><th>Дата</th><th>Вес, кг</th><th>Заметка</th><th></th></tr>
            </thead>
            <tbody>
              {items.map((w) => (
                <tr key={w.id}>
                  <td>{new Date(w.measured_on).toLocaleDateString("ru-RU")}</td>
                  <td>{fmtNum(w.weight_kg)}</td>
                  <td className="muted">{w.note || "—"}</td>
                  <td style={{ textAlign: "right" }}>
                    <button className="btn-danger btn-sm" onClick={() => void remove(w.id)}>×</button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </>
  );
}
