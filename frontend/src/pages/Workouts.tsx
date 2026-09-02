import { useCallback, useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";
import { api } from "../api/client";
import type { Workout } from "../api/types";
import { Empty, ErrorBox, fmtDate, fmtDuration, msg, useNow } from "../components/ui";

export function Workouts() {
  const [items, setItems] = useState<Workout[]>([]);
  const [note, setNote] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const navigate = useNavigate();
  const now = useNow(items.some((w) => w.ended_at === null));

  const load = useCallback(async () => {
    try {
      setItems(await api.workouts());
      setError(null);
    } catch (e) {
      setError(msg(e));
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => { void load(); }, [load]);

  async function start(e: React.FormEvent) {
    e.preventDefault();
    try {
      await api.createWorkout({ note: note.trim() });
      setNote("");
      await load();
    } catch (err) {
      setError(msg(err));
    }
  }

  async function remove(id: number) {
    if (!confirm("Удалить тренировку вместе с подходами?")) return;
    try {
      await api.deleteWorkout(id);
      await load();
    } catch (err) {
      setError(msg(err));
    }
  }

  return (
    <>
      <div className="page-head">
        <h1>Тренировки</h1>
      </div>
      <ErrorBox error={error} />

      <form className="card row wrap" onSubmit={start}>
        <input placeholder="Заметка к тренировке (необязательно)" maxLength={1000}
               value={note} onChange={(e) => setNote(e.target.value)}
               style={{ flex: 1, minWidth: 220 }} />
        <button className="btn-primary">Начать тренировку</button>
      </form>

      <div style={{ marginTop: 14 }}>
        {loading && <Empty>Загрузка…</Empty>}
        {!loading && items.length === 0 && <Empty>Пока ни одной тренировки. Начните первую.</Empty>}
        {items.map((w) => (
          <div className="list-row clickable" key={w.id} role="link" tabIndex={0}
               onClick={() => navigate(`/workouts/${w.id}`)}
               onKeyDown={(e) => { if (e.key === "Enter") navigate(`/workouts/${w.id}`); }}>
            <div>
              <div className="row" style={{ gap: 8 }}>
                <span style={{ fontWeight: 600 }}>{fmtDate(w.started_at)}</span>
                <span className={`badge ${w.ended_at ? "done" : "live"}`}>
                  {w.ended_at ? "завершена" : "идёт"}
                </span>
                <span className="small muted">{fmtDuration(w.started_at, w.ended_at, now)}</span>
              </div>
              <div className="small muted">{w.note || "без заметки"}</div>
            </div>
                        <button className="btn-danger btn-sm"
                    onClick={(e) => { e.stopPropagation(); void remove(w.id); }}>
              Удалить
            </button>
          </div>
        ))}
      </div>
    </>
  );
}
