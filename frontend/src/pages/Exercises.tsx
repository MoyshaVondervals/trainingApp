import { useCallback, useEffect, useState } from "react";
import { api } from "../api/client";
import type { Exercise, Muscle, MuscleGroup } from "../api/types";
import { MusclePicker, MuscleSummary } from "../components/MusclePicker";
import { Empty, ErrorBox, msg } from "../components/ui";

export function Exercises() {
  const [items, setItems] = useState<Exercise[]>([]);
  const [catalog, setCatalog] = useState<MuscleGroup[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);

  const [form, setForm] = useState({ name: "", description: "" });
  const [muscles, setMuscles] = useState<Muscle[]>([]);
  const [editing, setEditing] = useState<Exercise | null>(null);
  const [saving, setSaving] = useState(false);

  const [openMuscles, setOpenMuscles] = useState<number | null>(null);
  const [musclesByExercise, setMusclesByExercise] = useState<Record<number, Muscle[]>>({});

  const load = useCallback(async () => {
    try {
      const [ex, groups] = await Promise.all([api.exercises(), api.muscleGroups()]);
      setItems(ex);
      setCatalog(groups);
      setError(null);
    } catch (e) {
      setError(msg(e));
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => { void load(); }, [load]);

  function resetForm() {
    setEditing(null);
    setForm({ name: "", description: "" });
    setMuscles([]);
  }

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    if (muscles.length > 0 && muscles.filter((m) => m.role === "primary").length !== 1) {
      setError("Ровно одна мышца должна быть основной (★)");
      return;
    }
    setSaving(true);
    try {
      const saved = editing
        ? await api.updateExercise(editing.id, form)
        : await api.createExercise(form);
      if (muscles.length > 0) {
        const fresh = await api.setMuscles(saved.id, muscles);
        setMusclesByExercise((prev) => ({ ...prev, [saved.id]: fresh }));
      }
      resetForm();
      setError(null);
      await load();
    } catch (err) {
      setError(msg(err));
    } finally {
      setSaving(false);
    }
  }

  async function startEdit(ex: Exercise) {
    setEditing(ex);
    setForm({ name: ex.name, description: ex.description ?? "" });
    try {
      const current = musclesByExercise[ex.id] ?? (await api.muscles(ex.id));
      setMusclesByExercise((prev) => ({ ...prev, [ex.id]: current }));
      setMuscles(current);
    } catch (err) {
      setError(msg(err));
      setMuscles([]);
    }
  }

  async function remove(id: number) {
    if (!confirm("Удалить упражнение?")) return;
    try {
      await api.deleteExercise(id);
      if (editing?.id === id) resetForm();
      await load();
    } catch (err) {
      setError(msg(err));
    }
  }

  async function toggleMuscles(ex: Exercise) {
    if (openMuscles === ex.id) { setOpenMuscles(null); return; }
    setOpenMuscles(ex.id);
    if (musclesByExercise[ex.id]) return;
    try {
      const list = await api.muscles(ex.id);
      setMusclesByExercise((prev) => ({ ...prev, [ex.id]: list }));
    } catch (err) {
      setError(msg(err));
    }
  }

  return (
    <>
      <div className="page-head"><h1>Упражнения</h1></div>
      <ErrorBox error={error} />

      <form className="card" onSubmit={submit}>
        <h2>{editing ? `Правка «${editing.name}»` : "Новое личное упражнение"}</h2>
        <div className="field">
          <label htmlFor="name">Название</label>
          <input id="name" required maxLength={100} value={form.name}
                 onChange={(e) => setForm({ ...form, name: e.target.value })} />
        </div>
        <div className="field">
          <label htmlFor="descr">Описание</label>
          <textarea id="descr" rows={2} maxLength={1000} value={form.description}
                    onChange={(e) => setForm({ ...form, description: e.target.value })} />
        </div>
        <div className="field">
          <label>Задействованные мышцы</label>
          <MusclePicker catalog={catalog} value={muscles} onChange={setMuscles} />
        </div>
        <div className="row" style={{ marginTop: 14 }}>
          <button className="btn-primary" disabled={saving}>
            {saving ? "Сохраняем…" : editing ? "Сохранить" : "Создать"}
          </button>
          {editing && (
            <button type="button" className="btn-ghost" onClick={resetForm}>Отмена</button>
          )}
        </div>
      </form>

      <div style={{ marginTop: 14 }}>
        {loading && <Empty>Загрузка…</Empty>}
        {!loading && items.length === 0 && <Empty>Каталог пуст.</Empty>}
        {items.map((ex) => (
          <div key={ex.id}>
            <div className="list-row clickable" onClick={() => void toggleMuscles(ex)}>
              <div>
                <div className="row" style={{ gap: 8 }}>
                  <strong>{ex.name}</strong>
                  <span className={`badge ${ex.user_id === null ? "sys" : "own"}`}>
                    {ex.user_id === null ? "системное" : "личное"}
                  </span>
                </div>
                {ex.description && <div className="small muted">{ex.description}</div>}
              </div>
              {ex.user_id !== null && (
                <div className="row" onClick={(e) => e.stopPropagation()}>
                  <button className="btn-ghost btn-sm" onClick={() => void startEdit(ex)}>Править</button>
                  <button className="btn-danger btn-sm" onClick={() => void remove(ex.id)}>Удалить</button>
                </div>
              )}
            </div>
            {openMuscles === ex.id && (
              <div className="card" style={{ marginTop: 8 }}>
                {musclesByExercise[ex.id]
                  ? <MuscleSummary catalog={catalog} muscles={musclesByExercise[ex.id]} />
                  : <div className="small muted">Загрузка мышц…</div>}
              </div>
            )}
          </div>
        ))}
      </div>
    </>
  );
}
