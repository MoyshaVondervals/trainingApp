import { useCallback, useEffect, useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";
import { api } from "../api/client";
import type { Plan, Workout } from "../api/types";
import { Empty, ErrorBox, fmtDate, fmtDuration, msg, useNow } from "../components/ui";
import { Combobox } from "../components/Combobox";

export function Workouts() {
  const [items, setItems] = useState<Workout[]>([]);
  const [note, setNote] = useState("");
  const [plans, setPlans] = useState<Plan[]>([]);
  const [planId, setPlanId] = useState<number | "">("");
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const navigate = useNavigate();
  const now = useNow(items.some((w) => w.ended_at === null));

  const planOptions = useMemo(
    () => [
      { id: 0, label: "Без плана" },
      ...[...plans]
        .sort((a, b) => a.name.localeCompare(b.name, "ru"))
        .map((p) => ({
          id: p.id,
          label: p.name,
          hint: p.note ? p.note.slice(0, 24) : undefined,
        })),
    ],
    [plans],
  );

  const load = useCallback(async () => {
    try {
      const [list, planList] = await Promise.all([api.workouts(), api.plans()]);
      setItems(list);
      setPlans(planList);
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
      await api.createWorkout({
        note: note.trim(),
        plan_id: planId === "" ? undefined : planId,
      });
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

      <form className="card row wrap end" onSubmit={start}>
        <div style={{ flex: 2, minWidth: 220 }}>
          <label htmlFor="w-note">Заметка</label>
          <input id="w-note" placeholder="необязательно" maxLength={1000}
                 value={note} onChange={(e) => setNote(e.target.value)} />
        </div>
        <div style={{ flex: 1, minWidth: 180 }}>
          <label htmlFor="w-plan">План</label>
          <Combobox id="w-plan"
                    options={planOptions}
                    value={planId === "" ? 0 : planId}
                    onChange={(id) => setPlanId(id === 0 ? "" : id)}
                    placeholder="без плана"
                    emptyText="планов пока нет" />
        </div>
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
                {w.plan_id !== null && (
                  <span className="small muted">
                    · {plans.find((p) => p.id === w.plan_id)?.name ?? "по плану"}
                  </span>
                )}
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
