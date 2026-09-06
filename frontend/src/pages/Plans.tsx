import { useCallback, useEffect, useState } from "react";
import { api } from "../api/client";
import type { Exercise, Plan, PlanItem } from "../api/types";
import { Empty, ErrorBox, msg } from "../components/ui";
import { Combobox } from "../components/Combobox";

type Draft = {
  id: number | null;
  name: string;
  note: string;
  items: PlanItem[];
};

const emptyDraft: Draft = { id: null, name: "", note: "", items: [] };

export function Plans() {
  const [plans, setPlans] = useState<Plan[]>([]);
  const [exercises, setExercises] = useState<Exercise[]>([]);
  const [expanded, setExpanded] = useState<Record<number, Plan>>({});
  const [openPlan, setOpenPlan] = useState<number | null>(null);
  const [draft, setDraft] = useState<Draft>(emptyDraft);
  const [editing, setEditing] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);

  const load = useCallback(async () => {
    try {
      const [list, ex] = await Promise.all([api.plans(), api.exercises()]);
      setPlans(list);
      setExercises(ex);
      setError(null);
    } catch (e) {
      setError(msg(e));
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => { void load(); }, [load]);

  function addItem(exerciseId: number) {
    if (draft.items.some((i) => i.exercise_id === exerciseId)) return;
    setDraft({
      ...draft,
      items: [...draft.items, {
        exercise_id: exerciseId,
        position: draft.items.length + 1,
        target_sets: 4,
        target_reps: 10,
      }],
    });
  }

  function renumber(items: PlanItem[]): PlanItem[] {
    return items.map((item, i) => ({ ...item, position: i + 1 }));
  }

  function removeItem(index: number) {
    setDraft({ ...draft, items: renumber(draft.items.filter((_, i) => i !== index)) });
  }

  function move(index: number, delta: number) {
    const next = [...draft.items];
    const target = index + delta;
    if (target < 0 || target >= next.length) return;
    [next[index], next[target]] = [next[target], next[index]];
    setDraft({ ...draft, items: renumber(next) });
  }

  function updateItem(index: number, patch: Partial<PlanItem>) {
    setDraft({
      ...draft,
      items: draft.items.map((item, i) => (i === index ? { ...item, ...patch } : item)),
    });
  }

  async function save(e: React.FormEvent) {
    e.preventDefault();
    if (draft.items.length === 0) {
      setError("В плане должно быть хотя бы одно упражнение");
      return;
    }
    setSaving(true);
    try {
      const body = { name: draft.name, note: draft.note, exercises: renumber(draft.items) };
      if (draft.id === null) await api.createPlan(body);
      else await api.updatePlan(draft.id, body);
      setDraft(emptyDraft);
      setEditing(false);
      setExpanded({});
      setError(null);
      await load();
    } catch (err) {
      setError(msg(err));
    } finally {
      setSaving(false);
    }
  }

  async function startEdit(p: Plan) {
    try {
      const full = await api.plan(p.id);
      setDraft({
        id: full.id,
        name: full.name,
        note: full.note ?? "",
        items: full.exercises ?? [],
      });
      setEditing(true);
      setError(null);
    } catch (err) {
      setError(msg(err));
    }
  }

  async function remove(id: number) {
    if (!confirm("Удалить план?")) return;
    try {
      await api.deletePlan(id);
      if (draft.id === id) { setDraft(emptyDraft); setEditing(false); }
      await load();
    } catch (err) {
      setError(msg(err));
    }
  }

  async function toggle(p: Plan) {
    if (openPlan === p.id) { setOpenPlan(null); return; }
    setOpenPlan(p.id);
    if (expanded[p.id]) return;
    try {
      const full = await api.plan(p.id);
      setExpanded((prev) => ({ ...prev, [p.id]: full }));
    } catch (err) {
      setError(msg(err));
    }
  }

  const nameById = (id: number) => exercises.find((e) => e.id === id)?.name ?? `#${id}`;

  return (
    <>
      <div className="page-head">
        <h1>Планы</h1>
        {!editing && (
          <button className="btn-primary" onClick={() => { setDraft(emptyDraft); setEditing(true); }}>
            Новый план
          </button>
        )}
      </div>
      <ErrorBox error={error} />

      {editing && (
        <form className="card" onSubmit={save}>
          <h2>{draft.id === null ? "Новый план" : `Правка «${draft.name}»`}</h2>
          <div className="field">
            <label htmlFor="plan-name">Название</label>
            <input id="plan-name" required maxLength={100} value={draft.name}
                   onChange={(e) => setDraft({ ...draft, name: e.target.value })} />
          </div>
          <div className="field">
            <label htmlFor="plan-note">Заметка</label>
            <input id="plan-note" maxLength={1000} value={draft.note}
                   onChange={(e) => setDraft({ ...draft, note: e.target.value })} />
          </div>

          <div className="field">
            <label htmlFor="plan-add">Добавить упражнение</label>
            <Combobox id="plan-add"
                      options={exercises
                        .filter((e) => !draft.items.some((i) => i.exercise_id === e.id))
                        .sort((a, b) => a.name.localeCompare(b.name, "ru"))
                        .map((e) => ({
                          id: e.id,
                          label: e.name,
                          hint: e.user_id === null ? undefined : "личное",
                        }))}
                      value=""
                      onChange={addItem}
                      placeholder="название или часть слова" />
          </div>

          {draft.items.length === 0 && (
            <div className="small muted">Упражнений пока нет.</div>
          )}
          {draft.items.map((item, i) => (
            <div className="plan-item" key={item.exercise_id}>
              <span className="plan-pos">{i + 1}</span>
              <span className="plan-name">{nameById(item.exercise_id)}</span>
              <div className="plan-targets">
                <input type="number" min={1} max={30} value={item.target_sets} aria-label="подходы"
                       onChange={(e) => updateItem(i, { target_sets: Number(e.target.value) })} />
                <span className="muted">×</span>
                <input type="number" min={1} max={5000} value={item.target_reps} aria-label="повторы"
                       onChange={(e) => updateItem(i, { target_reps: Number(e.target.value) })} />
              </div>
              <div className="row">
                <button type="button" className="btn-ghost btn-sm" onClick={() => move(i, -1)}
                        disabled={i === 0}>↑</button>
                <button type="button" className="btn-ghost btn-sm" onClick={() => move(i, 1)}
                        disabled={i === draft.items.length - 1}>↓</button>
                <button type="button" className="btn-danger btn-sm" onClick={() => removeItem(i)}>×</button>
              </div>
            </div>
          ))}

          <div className="row" style={{ marginTop: 14 }}>
            <button className="btn-primary" disabled={saving}>
              {saving ? "Сохраняем…" : draft.id === null ? "Создать" : "Сохранить"}
            </button>
            <button type="button" className="btn-ghost"
                    onClick={() => { setDraft(emptyDraft); setEditing(false); }}>
              Отмена
            </button>
          </div>
        </form>
      )}

      <div style={{ marginTop: 14 }}>
        {loading && <Empty>Загрузка…</Empty>}
        {!loading && plans.length === 0 && !editing && (
          <Empty>Планов пока нет. Создайте первый, чтобы не держать порядок упражнений в голове.</Empty>
        )}
        {plans.map((p) => (
          <div key={p.id}>
            <div className="list-row clickable" onClick={() => void toggle(p)}>
              <div>
                <strong>{p.name}</strong>
                {p.note && <div className="small muted">{p.note}</div>}
              </div>
              <div className="row" onClick={(e) => e.stopPropagation()}>
                <button className="btn-ghost btn-sm" onClick={() => void startEdit(p)}>Править</button>
                <button className="btn-danger btn-sm" onClick={() => void remove(p.id)}>Удалить</button>
              </div>
            </div>
            {openPlan === p.id && (
              <div className="card" style={{ marginTop: 8 }}>
                {expanded[p.id]
                  ? (expanded[p.id].exercises ?? []).map((i) => (
                      <div className="row between small" key={i.exercise_id} style={{ padding: "4px 0" }}>
                        <span>{i.position}. {i.exercise_name ?? nameById(i.exercise_id)}</span>
                        <span className="muted">{i.target_sets}×{i.target_reps}</span>
                      </div>
                    ))
                  : <div className="small muted">Загрузка…</div>}
              </div>
            )}
          </div>
        ))}
      </div>
    </>
  );
}
