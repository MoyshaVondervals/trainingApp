import { Fragment, useCallback, useEffect, useMemo, useState } from "react";
import { Link, useNavigate, useParams } from "react-router-dom";
import { api } from "../api/client";
import type { Exercise, Muscle, MuscleGroup, Set, Workout } from "../api/types";
import { MuscleSummary } from "../components/MusclePicker";
import { Empty, ErrorBox, fmtDate, fmtDuration, fmtNum, msg, useNow } from "../components/ui";

export function WorkoutDetail() {
  const { id } = useParams();
  const workoutId = Number(id);
  const navigate = useNavigate();

  const [workout, setWorkout] = useState<Workout | null>(null);
  const [sets, setSets] = useState<Set[]>([]);
  const [exercises, setExercises] = useState<Exercise[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);

  const [exerciseId, setExerciseId] = useState<number | "">("");
  const [reps, setReps] = useState("10");
  const [weight, setWeight] = useState("0");
  const [noteDraft, setNoteDraft] = useState("");

  const [catalog, setCatalog] = useState<MuscleGroup[]>([]);
  const [openSet, setOpenSet] = useState<number | null>(null);
  const [musclesByExercise, setMusclesByExercise] = useState<Record<number, Muscle[]>>({});
  const [musclesError, setMusclesError] = useState<string | null>(null);

  const now = useNow(workout !== null && workout.ended_at === null);

  const load = useCallback(async () => {
    try {
      const [w, s, ex, groups] = await Promise.all([
        api.workout(workoutId),
        api.setsByWorkout(workoutId),
        api.exercises(),
        api.muscleGroups(),
      ]);
      setCatalog(groups);
      setWorkout(w);
      setNoteDraft(w.note ?? "");
      setSets(s);
      setExercises(ex);
      if (ex.length > 0) setExerciseId((prev) => (prev === "" ? ex[0].id : prev));
      setError(null);
    } catch (e) {
      setError(msg(e));
    } finally {
      setLoading(false);
    }
  }, [workoutId]);

  useEffect(() => { void load(); }, [load]);

  const exerciseName = useCallback(
    (eid: number) => exercises.find((e) => e.id === eid)?.name ?? `#${eid}`,
    [exercises],
  );

  const groups = useMemo(() => {
    const byExercise = new Map<number, Set[]>();
    for (const s of sets) {
      const list = byExercise.get(s.exercise_id) ?? [];
      list.push(s);
      byExercise.set(s.exercise_id, list);
    }
    return [...byExercise.entries()].map(([eid, list]) => ({
      exerciseId: eid,
      sets: [...list].sort((a, b) => a.set_number - b.set_number),
    }));
  }, [sets]);

  const volume = useMemo(
    () => sets.reduce((acc, s) => acc + s.reps * s.weight, 0),
    [sets],
  );

  async function addSet(e: React.FormEvent) {
    e.preventDefault();
    if (exerciseId === "") return;
    const used = sets.filter((s) => s.exercise_id === exerciseId);
    const nextNumber = used.length === 0
      ? 1
      : Math.max(...used.map((s) => s.set_number)) + 1;
    try {
      await api.createSet({
        exercise_id: exerciseId,
        workout_id: workoutId,
        set_number: nextNumber,
        reps: Number(reps),
        weight: weight === "" ? 0 : Number(weight),
      });
      setSets(await api.setsByWorkout(workoutId));
      setError(null);
    } catch (err) {
      setError(msg(err));
    }
  }

  async function toggleSet(s: Set) {
    if (openSet === s.id) { setOpenSet(null); return; }
    setOpenSet(s.id);
    if (musclesByExercise[s.exercise_id]) return;
    try {
      const list = await api.muscles(s.exercise_id);
      setMusclesByExercise((prev) => ({ ...prev, [s.exercise_id]: list }));
      setMusclesError(null);
    } catch (err) {
      setMusclesError(msg(err));
    }
  }

  async function removeSet(setId: number) {
    try {
      await api.deleteSet(setId);
      setSets(await api.setsByWorkout(workoutId));
    } catch (err) {
      setError(msg(err));
    }
  }

  async function finish() {
    try {
      setWorkout(await api.finishWorkout(workoutId));
    } catch (err) {
      setError(msg(err));
    }
  }

  async function saveNote() {
    try {
      setWorkout(await api.updateWorkout(workoutId, { note: noteDraft }));
      setError(null);
    } catch (err) {
      setError(msg(err));
    }
  }

  async function removeWorkout() {
    if (!confirm("Удалить тренировку вместе с подходами?")) return;
    try {
      await api.deleteWorkout(workoutId);
      navigate("/workouts", { replace: true });
    } catch (err) {
      setError(msg(err));
    }
  }

  if (loading) return <Empty>Загрузка…</Empty>;
  if (!workout) return <><ErrorBox error={error} /><Link to="/workouts">← к тренировкам</Link></>;

  return (
    <>
      <div className="page-head">
        <div>
          <Link to="/workouts" className="small">← к тренировкам</Link>
          <h1 style={{ marginTop: 6 }}>Тренировка {fmtDate(workout.started_at)}</h1>
          <div className="row small muted">
            <span className={`badge ${workout.ended_at ? "done" : "live"}`}>
              {workout.ended_at ? "завершена" : "идёт"}
            </span>
            {workout.ended_at && <span>окончание {fmtDate(workout.ended_at)}</span>}
            <span>длительность {fmtDuration(workout.started_at, workout.ended_at, now)}</span>
          </div>
        </div>
        <div className="row">
          {!workout.ended_at && <button className="btn-teal" onClick={finish}>Завершить</button>}
          <button className="btn-danger" onClick={removeWorkout}>Удалить</button>
        </div>
      </div>

      <ErrorBox error={error} />

      <div className="grid cols-4">
        <div className="card">
          <div className="stat-label">Подходов</div>
          <div className="stat-value">{sets.length}</div>
        </div>
        <div className="card">
          <div className="stat-label">Повторов</div>
          <div className="stat-value">{sets.reduce((a, s) => a + s.reps, 0)}</div>
        </div>
        <div className="card">
          <div className="stat-label">Объём, кг</div>
          <div className="stat-value">{fmtNum(volume)}</div>
        </div>
        <div className="card">
          <div className="stat-label">Упражнений</div>
          <div className="stat-value">{groups.length}</div>
        </div>
        <div className="card">
          <div className="stat-label">Длительность</div>
          <div className="stat-value" style={{ fontSize: 24 }}>
            {fmtDuration(workout.started_at, workout.ended_at, now)}
          </div>
        </div>
      </div>

      <div className="card" style={{ marginTop: 14 }}>
        <h2>Добавить подход</h2>
        <form className="row wrap end" onSubmit={addSet}>
          <div style={{ flex: 2, minWidth: 200 }}>
            <label htmlFor="set-exercise">Упражнение</label>
            <select id="set-exercise" value={exerciseId}
                    onChange={(e) => setExerciseId(Number(e.target.value))}>
              {exercises.map((e) => <option key={e.id} value={e.id}>{e.name}</option>)}
            </select>
          </div>
          <div style={{ flex: 1, minWidth: 110 }}>
            <label htmlFor="set-reps">Повторения</label>
            <input id="set-reps" type="number" min={1} max={5000} value={reps} required
                   onChange={(e) => setReps(e.target.value)} />
          </div>
          <div style={{ flex: 1, minWidth: 140 }}>
            <label htmlFor="set-weight">Вес, кг — 0 свой вес</label>
            <input id="set-weight" type="number" min={0} max={350} step="0.5" value={weight}
                   onChange={(e) => setWeight(e.target.value)} />
          </div>
          <button className="btn-primary" disabled={exercises.length === 0}>Добавить</button>
        </form>
        {exercises.length === 0 && (
          <div className="small muted" style={{ marginTop: 8 }}>
            Сначала создайте упражнение на вкладке «Упражнения».
          </div>
        )}
      </div>

      <div className="card">
        <h2>Заметка</h2>
        <div className="row wrap">
          <input value={noteDraft} maxLength={1000} onChange={(e) => setNoteDraft(e.target.value)}
                 placeholder="Как прошло" style={{ flex: 1, minWidth: 220 }} />
          <button className="btn-ghost" onClick={saveNote}>Сохранить</button>
        </div>
      </div>

      {groups.length === 0 && <Empty>Подходов ещё нет.</Empty>}
      {groups.map((g) => (
        <div className="card" key={g.exerciseId}>
          <h2>{exerciseName(g.exerciseId)}</h2>
          <table>
            <thead>
              <tr><th>№</th><th>Повторы</th><th>Вес, кг</th><th>Объём</th><th></th></tr>
            </thead>
            <tbody>
              {g.sets.map((s) => (
                <Fragment key={s.id}>
                  <tr className={`clickable ${openSet === s.id ? "open" : ""}`}
                      onClick={() => void toggleSet(s)}
                      title="Показать задействованные мышцы">
                    <td>{s.set_number}</td>
                    <td>{s.reps}</td>
                    <td>{s.weight === 0 ? <span className="muted">свой вес</span> : fmtNum(s.weight)}</td>
                    <td className="muted">{fmtNum(s.reps * s.weight)}</td>
                    <td style={{ textAlign: "right" }}>
                      <button className="btn-danger btn-sm"
                              onClick={(e) => { e.stopPropagation(); void removeSet(s.id); }}>×</button>
                    </td>
                  </tr>
                  {openSet === s.id && (
                    <tr className="set-details">
                      <td colSpan={5}>
                        <ErrorBox error={musclesError} />
                        {musclesByExercise[s.exercise_id]
                          ? <MuscleSummary catalog={catalog}
                                           muscles={musclesByExercise[s.exercise_id]} />
                          : <div className="small muted">Загрузка мышц…</div>}
                      </td>
                    </tr>
                  )}
                </Fragment>
              ))}
            </tbody>
          </table>
        </div>
      ))}
    </>
  );
}
