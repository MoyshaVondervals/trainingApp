import { Fragment, useCallback, useEffect, useMemo, useRef, useState } from "react";
import { Link, useNavigate, useParams } from "react-router-dom";
import { api } from "../api/client";
import type {
  Exercise, LastPerformance, Muscle, MuscleGroup, Plan, Set, Workout,
} from "../api/types";
import { MuscleSummary } from "../components/MusclePicker";
import { Combobox } from "../components/Combobox";
import {
  Empty, ErrorBox, fmtDate, fmtDuration, fmtNum, fmtRest, msg, parseDecimal, useNow,
} from "../components/ui";

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
  const [lastByExercise, setLastByExercise] = useState<Record<number, LastPerformance | null>>({});
  const [plan, setPlan] = useState<Plan | null>(null);
  const [step, setStep] = useState<number | null>(null);
  const [showPlan, setShowPlan] = useState(false);
  const [showNote, setShowNote] = useState(false);
  const [openSet, setOpenSet] = useState<number | null>(null);
  const [musclesByExercise, setMusclesByExercise] = useState<Record<number, Muscle[]>>({});
  const [musclesError, setMusclesError] = useState<string | null>(null);

  const now = useNow(workout !== null && workout.ended_at === null);
  // Таймер отдыха тикает ежесекундно, но только пока тренировка идёт и в ней есть подходы.
  const restNow = useNow(workout !== null && workout.ended_at === null && sets.length > 0, 1000);

  const load = useCallback(async () => {
    try {
      const [w, s, ex, groups] = await Promise.all([
        api.workout(workoutId),
        api.setsByWorkout(workoutId),
        api.exercises(),
        api.muscleGroups(),
      ]);
      setCatalog(groups);
      setPlan(w.plan_id === null ? null : await api.plan(w.plan_id));
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

  useEffect(() => {
    if (exerciseId === "") return;
    if (exerciseId in lastByExercise) return;
    let cancelled = false;
    api.lastSets(exerciseId, workoutId, workout?.plan_id ?? null)
      .then((res) => { if (!cancelled) setLastByExercise((p) => ({ ...p, [exerciseId]: res })); })
      .catch(() => { if (!cancelled) setLastByExercise((p) => ({ ...p, [exerciseId]: null })); });
    return () => { cancelled = true; };
  }, [exerciseId, workoutId, lastByExercise, workout?.plan_id]);

  useEffect(() => {
    if (exerciseId === "") return;
    const done = sets
      .filter((s) => s.exercise_id === exerciseId)
      .sort((a, b) => a.set_number - b.set_number);
    const previous = lastByExercise[exerciseId]?.sets ?? [];
    const source = done.length > 0 ? done[done.length - 1] : previous[previous.length - 1];
    if (!source) return;
    setReps(String(source.reps));
    setWeight(String(source.weight));
  }, [exerciseId, sets, lastByExercise]);

  const exerciseOptions = useMemo(
    () => [...exercises]
      .sort((a, b) => a.name.localeCompare(b.name, "ru"))
      .map((e) => ({
        id: e.id,
        label: e.name,
        hint: e.user_id === null ? undefined : "личное",
      })),
    [exercises],
  );

  const planItems = useMemo(() => plan?.exercises ?? [], [plan]);

  const doneByExercise = useMemo(() => {
    const counts = new Map<number, number>();
    for (const s of sets) {
      counts.set(s.exercise_id, (counts.get(s.exercise_id) ?? 0) + 1);
    }
    return counts;
  }, [sets]);

  const firstUnfinished = useMemo(() => {
    const index = planItems.findIndex(
      (item) => (doneByExercise.get(item.exercise_id) ?? 0) < item.target_sets,
    );
    return index === -1 ? Math.max(planItems.length - 1, 0) : index;
  }, [planItems, doneByExercise]);

  const stepKey = `trainingapp.plan-step.${workoutId}`;

  useEffect(() => {
    if (planItems.length === 0) return;
    if (step !== null) return;
    let stored: number | null = null;
    try {
      const raw = localStorage.getItem(stepKey);
      if (raw !== null) {
        const parsed = Number(raw);
        if (Number.isInteger(parsed) && parsed >= 0 && parsed < planItems.length) stored = parsed;
      }
    } catch {
      stored = null;
    }
    setStep(stored ?? firstUnfinished);
  }, [planItems, step, firstUnfinished, stepKey]);

  const currentItem = step === null ? undefined : planItems[step];

  useEffect(() => {
    if (!currentItem) return;
    setExerciseId(currentItem.exercise_id);
    setReps(String(currentItem.target_reps));
  }, [currentItem]);

  const touchStartX = useRef<number | null>(null);

  function onTouchStart(e: React.TouchEvent) {
    touchStartX.current = e.touches[0].clientX;
  }

  function onTouchEnd(e: React.TouchEvent) {
    if (touchStartX.current === null || step === null) return;
    const delta = e.changedTouches[0].clientX - touchStartX.current;
    touchStartX.current = null;
    if (Math.abs(delta) < 50) return;
    goToStep(delta < 0 ? step + 1 : step - 1);
  }

  function goToStep(next: number) {
    if (next < 0 || next >= planItems.length) return;
    setStep(next);
    try {
      localStorage.setItem(stepKey, String(next));
    } catch {
      // приватный режим браузера — состояние всё равно восстановится из подходов
    }
  }

  const exerciseLink = useCallback(
    (eid: number) => exercises.find((e) => e.id === eid)?.link ?? "",
    [exercises],
  );

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
    return [...byExercise.entries()]
      .map(([eid, list]) => ({
        exerciseId: eid,
        sets: [...list].sort((a, b) => a.set_number - b.set_number),
        lastAt: Math.max(...list.map((s) => new Date(s.created_at).getTime())),
      }))
      .sort((a, b) => b.lastAt - a.lastAt);
  }, [sets]);

  const lastSetAt = useMemo(() => {
    if (sets.length === 0) return null;
    return sets.reduce(
      (latest, s) => (s.created_at > latest ? s.created_at : latest),
      sets[0].created_at,
    );
  }, [sets]);

  const volume = useMemo(
    () => sets.reduce((acc, s) => acc + s.reps * s.weight, 0),
    [sets],
  );

  async function addSet(e: React.FormEvent) {
    e.preventDefault();
    if (exerciseId === "") return;
    const kg = weight.trim() === "" ? 0 : parseDecimal(weight);
    if (!Number.isFinite(kg) || kg < 0 || kg > 350) {
      setError("Вес должен быть числом от 0 до 350");
      return;
    }
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
        weight: kg,
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

      <div className="stat-strip">
        <div className="stat-tile">
          <span className="stat-label">Подходов</span>
          <span className="stat-num">{sets.length}</span>
        </div>
        <div className="stat-tile">
          <span className="stat-label">Повторов</span>
          <span className="stat-num">{sets.reduce((a, s) => a + s.reps, 0)}</span>
        </div>
        <div className="stat-tile">
          <span className="stat-label">Объём</span>
          <span className="stat-num">{fmtNum(volume)}</span>
        </div>
        <div className="stat-tile">
          <span className="stat-label">Упражнений</span>
          <span className="stat-num">{groups.length}</span>
        </div>
        <div className="stat-tile">
          <span className="stat-label">Длительность</span>
          <span className="stat-num">{fmtDuration(workout.started_at, workout.ended_at, now)}</span>
        </div>
        {lastSetAt && workout.ended_at === null && (
          <div className="stat-tile rest-tile">
            <span className="stat-label">Отдых</span>
            <span className="stat-num rest-value">{fmtRest(lastSetAt, restNow)}</span>
          </div>
        )}
      </div>

      {currentItem && (
        <div className="card plan-banner" style={{ marginTop: 14 }}
             onTouchStart={onTouchStart} onTouchEnd={onTouchEnd}>
          <div style={{ flex: 1, minWidth: 220 }}>
            <div className="row between">
              <span className="stat-label">{plan?.name} · шаг {step! + 1} из {planItems.length}</span>
              <span className="row-actions">
                <button className="btn-ghost btn-sm icon-btn" title="Предыдущее упражнение"
                        onClick={() => goToStep(step! - 1)} disabled={step === 0}>‹</button>
                <button className="btn-teal btn-sm icon-btn" title="Следующее упражнение"
                        onClick={() => goToStep(step! + 1)}
                        disabled={step! >= planItems.length - 1}>›</button>
              </span>
            </div>
            <div className="row" style={{ gap: 10 }}>
              <div className="plan-step">{exerciseName(currentItem.exercise_id)}</div>
              {exerciseLink(currentItem.exercise_id) && (
                <a className="link-btn" href={exerciseLink(currentItem.exercise_id)}
                   target="_blank" rel="noopener noreferrer" title="Открыть описание техники">
                  Техника
                </a>
              )}
            </div>
            <div className="small muted">
              сделано {doneByExercise.get(currentItem.exercise_id) ?? 0} из {currentItem.target_sets}
              {" "}по {currentItem.target_reps} повторений
            </div>
            <div className="plan-progress">
              {Array.from({ length: currentItem.target_sets }, (_, i) => (
                <span key={i}
                      className={`plan-dot ${i < (doneByExercise.get(currentItem.exercise_id) ?? 0) ? "done" : ""}`} />
              ))}
            </div>
          </div>
          <div className="row wrap">
            <button className="btn-ghost btn-sm" onClick={() => setShowPlan((v) => !v)}>
              {showPlan ? "Скрыть план" : "Посмотреть план"}
            </button>
            <span className="small muted swipe-hint">свайп влево и вправо листает упражнения</span>
          </div>
          {showPlan && (
            <div className="plan-queue" style={{ width: "100%" }}>
              {planItems.map((item, i) => {
                const done = doneByExercise.get(item.exercise_id) ?? 0;
                const state = i === step ? "current" : done >= item.target_sets ? "finished" : "";
                return (
                  <div className={`plan-queue-row ${state}`} key={item.exercise_id}
                       onClick={() => goToStep(i)} style={{ cursor: "pointer" }}>
                    <span>{i + 1}. {exerciseName(item.exercise_id)}</span>
                    <span>{done}/{item.target_sets} × {item.target_reps}</span>
                  </div>
                );
              })}
            </div>
          )}
        </div>
      )}

      <div className="card" style={{ marginTop: 14 }}>
        <h2>Добавить подход</h2>
        <form className="row wrap end" onSubmit={addSet}>
          <div style={{ flex: 2, minWidth: 200 }}>
            <label htmlFor="set-exercise">Упражнение</label>
            <Combobox id="set-exercise"
                      options={exerciseOptions}
                      value={exerciseId}
                      onChange={setExerciseId}
                      placeholder="название или часть слова"
                      disabled={exercises.length === 0} />
          </div>
          <div style={{ flex: 1, minWidth: 110 }}>
            <label htmlFor="set-reps">Повторения</label>
            <input id="set-reps" type="number" min={1} max={5000} value={reps} required
                   onChange={(e) => setReps(e.target.value)} />
          </div>
          <div style={{ flex: 1, minWidth: 140 }}>
            <label htmlFor="set-weight">Вес, кг — 0 свой вес</label>
            <input id="set-weight" type="text" inputMode="decimal" value={weight}
                   onChange={(e) => setWeight(e.target.value)} />
          </div>
          <button className="btn-primary" disabled={exercises.length === 0}>Добавить</button>
        </form>
        {exerciseId !== "" && lastByExercise[exerciseId] && (
          <div className="small muted" style={{ marginTop: 10 }}>
            Прошлый раз {fmtDate(lastByExercise[exerciseId]!.performed_at)}:{" "}
            {lastByExercise[exerciseId]!.sets
              .map((s) => `${s.reps}×${s.weight === 0 ? "свой вес" : `${fmtNum(s.weight)} кг`}`)
              .join(", ")}
          </div>
        )}
        {exerciseId !== "" && lastByExercise[exerciseId] === null && (
          <div className="small muted" style={{ marginTop: 10 }}>
            Это упражнение раньше не выполнялось.
          </div>
        )}
        {exercises.length === 0 && (
          <div className="small muted" style={{ marginTop: 8 }}>
            Сначала создайте упражнение на вкладке «Упражнения».
          </div>
        )}
      </div>

      {groups.length === 0 && <Empty>Подходов ещё нет.</Empty>}
      {groups.map((g) => (
        <div className="card" key={g.exerciseId}>
          <div className="row between wrap" style={{ marginBottom: 12 }}>
            <h2 style={{ margin: 0 }}>{exerciseName(g.exerciseId)}</h2>
            {exerciseLink(g.exerciseId) && (
              <a className="link-btn" href={exerciseLink(g.exerciseId)}
                 target="_blank" rel="noopener noreferrer">
                Техника
              </a>
            )}
          </div>
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

      <div className="card note-card">
        <button className="note-toggle" onClick={() => setShowNote((v) => !v)}>
          <span>Заметка</span>
          <span className="small muted ellipsis">
            {workout.note ? workout.note : "нет"}
          </span>
          <span className="note-chevron">{showNote ? "▴" : "▾"}</span>
        </button>
        {showNote && (
          <div className="row wrap" style={{ marginTop: 10 }}>
            <input value={noteDraft} maxLength={1000} onChange={(e) => setNoteDraft(e.target.value)}
                   placeholder="Как прошло" style={{ flex: 1, minWidth: 180 }} />
            <button className="btn-ghost" onClick={saveNote}>Сохранить</button>
          </div>
        )}
      </div>
    </>
  );
}
