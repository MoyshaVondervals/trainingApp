import { useCallback, useEffect, useState } from "react";
import { api } from "../api/client";
import type { Dashboard, ExerciseWithRole } from "../api/types";
import { Empty, ErrorBox, fmtDate, fmtNum, msg } from "../components/ui";

function isoDay(d: Date): string {
  return d.toISOString().slice(0, 10);
}

export function Stats() {
  const [data, setData] = useState<Dashboard | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const [from, setFrom] = useState(isoDay(new Date(Date.now() - 90 * 864e5)));
  const [to, setTo] = useState(isoDay(new Date()));

  const [openMuscle, setOpenMuscle] = useState<string | null>(null);
  const [byMuscle, setByMuscle] = useState<Record<string, ExerciseWithRole[]>>({});
  const [muscleError, setMuscleError] = useState<string | null>(null);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      setData(await api.stats(`${from}T00:00:00Z`, `${to}T23:59:59Z`));
      setError(null);
    } catch (e) {
      setError(msg(e));
    } finally {
      setLoading(false);
    }
  }, [from, to]);

  useEffect(() => { void load(); }, [load]);

  const maxVolume = data?.muscles.reduce((m, x) => Math.max(m, x.volume), 0) ?? 0;

  async function toggleMuscle(code: string) {
    if (openMuscle === code) { setOpenMuscle(null); return; }
    setOpenMuscle(code);
    setMuscleError(null);
    if (byMuscle[code]) return;
    try {
      const list = await api.exercisesByMuscle(code);
      setByMuscle((prev) => ({ ...prev, [code]: list }));
    } catch (e) {
      setMuscleError(msg(e));
    }
  }

  return (
    <>
      <div className="page-head">
        <h1>Статистика</h1>
        <div className="row wrap">
          <input type="date" value={from} onChange={(e) => setFrom(e.target.value)} style={{ width: 165 }} />
          <span className="muted">—</span>
          <input type="date" value={to} onChange={(e) => setTo(e.target.value)} style={{ width: 165 }} />
          <button className="btn-teal" onClick={load} disabled={loading}>Обновить</button>
        </div>
      </div>

      <ErrorBox error={error} />
      {loading && <Empty>Считаем…</Empty>}

      {data && !loading && (
        <>
          <div className="grid cols-4">
            <div className="card">
              <div className="stat-label">Тренировок</div>
              <div className="stat-value">{data.summary.workouts}</div>
            </div>
            <div className="card">
              <div className="stat-label">Подходов</div>
              <div className="stat-value">{data.summary.sets}</div>
            </div>
            <div className="card">
              <div className="stat-label">Повторов</div>
              <div className="stat-value">{data.summary.reps}</div>
            </div>
            <div className="card">
              <div className="stat-label">Объём, кг</div>
              <div className="stat-value">{fmtNum(data.summary.volume)}</div>
            </div>
          </div>

          <div className="card" style={{ marginTop: 14 }}>
            <h2>Нагрузка по мышечным группам</h2>
            {data.muscles.length === 0 && <div className="small muted">Данных за период нет.</div>}
            {data.muscles.map((m) => (
              <div key={m.code} style={{ marginBottom: 12 }}>
                <div className="muscle-row" role="button" tabIndex={0}
                     onClick={() => void toggleMuscle(m.code)}
                     onKeyDown={(e) => { if (e.key === "Enter") void toggleMuscle(m.code); }}
                     title="Чем качать эту мышцу">
                  <div className="row between small">
                    <span><strong>{m.name}</strong> <span className="muted">· {m.region}</span></span>
                    <span className="muted">
                      {fmtNum(m.volume)} кг · {m.sets} подх. · {m.reps} повт.
                    </span>
                  </div>
                  <div className="bar-track" style={{ marginTop: 5 }}>
                    <div className="bar-fill"
                         style={{ width: maxVolume > 0 ? `${(m.volume / maxVolume) * 100}%` : "0%" }} />
                  </div>
                </div>
                {openMuscle === m.code && (
                  <MuscleExercises list={byMuscle[m.code]} error={muscleError} />
                )}
              </div>
            ))}
          </div>

          <div className="card">
            <h2>Персональные рекорды</h2>
            {data.records.length === 0 && <div className="small muted">Рекордов пока нет.</div>}
            {data.records.length > 0 && (
              <table>
                <thead>
                  <tr><th>Упражнение</th><th>Вес</th><th>Повторы</th><th>Когда</th></tr>
                </thead>
                <tbody>
                  {data.records.map((r) => (
                    <tr key={r.exercise_id}>
                      <td>{r.exercise_name}</td>
                      <td>{r.weight_kg === null ? "своим весом" : `${fmtNum(r.weight_kg)} кг`}</td>
                      <td>{r.reps}</td>
                      <td className="muted">{fmtDate(r.achieved_at)}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            )}
          </div>

          <div className="small muted" style={{ marginTop: 10 }}>
            Период: {fmtDate(data.period.from)} — {fmtDate(data.period.to)}
          </div>
        </>
      )}
    </>
  );
}

function MuscleExercises({ list, error }: { list?: ExerciseWithRole[]; error: string | null }) {
  if (error) return <div className="muscle-drawer"><ErrorBox error={error} /></div>;
  if (!list) return <div className="muscle-drawer small muted">Загрузка упражнений…</div>;
  if (list.length === 0) {
    return <div className="muscle-drawer small muted">Упражнений на эту группу пока нет.</div>;
  }
  return (
    <div className="muscle-drawer">
      {list.map((e) => (
        <div className="row between" key={e.id} style={{ padding: "5px 0" }}>
          <span className="small">
            {e.name}
            {e.user_id !== null && <span className="badge own" style={{ marginLeft: 8 }}>личное</span>}
          </span>
          <span className={`badge ${e.role === "primary" ? "own" : "done"}`}>
            {e.role === "primary" ? "основная" : "вспомогательная"}
          </span>
        </div>
      ))}
    </div>
  );
}
