import { useMemo, useState } from "react";
import type { Muscle, MuscleGroup } from "../api/types";

type Region = { code: string; name: string; groups: MuscleGroup[] };

/** Группирует плоский справочник по регионам, сохраняя порядок каталога. */
export function useRegions(catalog: MuscleGroup[]): Region[] {
  return useMemo(() => {
    const byRegion = new Map<string, Region>();
    for (const g of catalog) {
      let r = byRegion.get(g.region_code);
      if (!r) {
        r = { code: g.region_code, name: g.region_name, groups: [] };
        byRegion.set(g.region_code, r);
      }
      r.groups.push(g);
    }
    return [...byRegion.values()];
  }, [catalog]);
}

type PickerProps = {
  catalog: MuscleGroup[];
  value: Muscle[];
  onChange: (next: Muscle[]) => void;
};

/**
 * Выбор мышц в два шага: сначала разделы, затем группы внутри выбранных.
 * Ровно одна группа обязана быть primary — это требование API.
 */
export function MusclePicker({ catalog, value, onChange }: PickerProps) {
  const regions = useRegions(catalog);
  const chosen = useMemo(() => new Map(value.map((m) => [m.muscle_group_id, m.role])), [value]);

  // Разделы уже выбранных групп раскрыты изначально — иначе правка вслепую.
  const [openRegions, setOpenRegions] = useState<string[]>(() => {
    const ids = new Set(value.map((m) => m.muscle_group_id));
    return catalog.filter((g) => ids.has(g.id)).map((g) => g.region_code);
  });

  function toggleRegion(code: string) {
    setOpenRegions((prev) =>
      prev.includes(code) ? prev.filter((c) => c !== code) : [...prev, code],
    );
  }

  function toggleGroup(id: number) {
    if (chosen.has(id)) {
      onChange(value.filter((m) => m.muscle_group_id !== id));
      return;
    }
    // Первая выбранная группа становится primary, остальные — secondary.
    const role: Muscle["role"] = value.some((m) => m.role === "primary") ? "secondary" : "primary";
    onChange([...value, { muscle_group_id: id, role }]);
  }

  function makePrimary(id: number) {
    onChange(
      value.map((m) => ({
        muscle_group_id: m.muscle_group_id,
        role: m.muscle_group_id === id ? "primary" : "secondary",
      })),
    );
  }

  const primaryCount = value.filter((m) => m.role === "primary").length;

  return (
    <div>
      <div className="small muted" style={{ marginBottom: 6 }}>Шаг 1 — разделы</div>
      <div className="chips">
        {regions.map((r) => {
          const count = r.groups.filter((g) => chosen.has(g.id)).length;
          return (
            <button type="button" key={r.code}
                    className={`chip ${openRegions.includes(r.code) ? "chip-on" : ""}`}
                    onClick={() => toggleRegion(r.code)}>
              {r.name}{count > 0 && <span className="chip-count">{count}</span>}
            </button>
          );
        })}
      </div>

      {openRegions.length > 0 && (
        <div className="small muted" style={{ margin: "14px 0 6px" }}>Шаг 2 — мышцы</div>
      )}
      {regions
        .filter((r) => openRegions.includes(r.code))
        .map((r) => (
          <div key={r.code} className="region-block">
            <div className="small" style={{ fontWeight: 700, marginBottom: 6 }}>{r.name}</div>
            <div className="chips">
              {r.groups.map((g) => {
                const role = chosen.get(g.id);
                return (
                  <span key={g.id} className={`chip ${role ? `chip-${role}` : ""}`}>
                    <button type="button" className="chip-main" onClick={() => toggleGroup(g.id)}>
                      {g.name}
                    </button>
                    {role === "secondary" && (
                      <button type="button" className="chip-star" title="Сделать основной"
                              onClick={() => makePrimary(g.id)}>★</button>
                    )}
                    {role === "primary" && <span className="chip-star on" title="Основная">★</span>}
                  </span>
                );
              })}
            </div>
          </div>
        ))}

      {value.length > 0 && (
        <div className="small muted" style={{ marginTop: 10 }}>
          Выбрано: {value.length}.{" "}
          {primaryCount === 1
            ? "★ — основная мышца."
            : <span style={{ color: "var(--danger)" }}>Нужна ровно одна основная (★).</span>}
        </div>
      )}
    </div>
  );
}

/** Только чтение: мышцы упражнения, сгруппированные по разделам. */
export function MuscleSummary({ catalog, muscles }: { catalog: MuscleGroup[]; muscles: Muscle[] }) {
  const byId = useMemo(() => new Map(catalog.map((g) => [g.id, g])), [catalog]);

  const grouped = useMemo(() => {
    const map = new Map<string, { name: string; items: { name: string; role: Muscle["role"] }[] }>();
    for (const m of muscles) {
      const g = byId.get(m.muscle_group_id);
      const key = g?.region_code ?? "unknown";
      const entry = map.get(key) ?? { name: g?.region_name ?? "Прочее", items: [] };
      entry.items.push({ name: g?.name ?? `группа #${m.muscle_group_id}`, role: m.role });
      map.set(key, entry);
    }
    // primary первыми — их и хочется видеть сразу.
    for (const e of map.values()) {
      e.items.sort((a, b) => (a.role === b.role ? 0 : a.role === "primary" ? -1 : 1));
    }
    return [...map.values()];
  }, [muscles, byId]);

  if (muscles.length === 0) return <div className="small muted">Мышцы не назначены.</div>;

  return (
    <div>
      {grouped.map((r) => (
        <div key={r.name} className="region-block">
          <div className="small muted" style={{ marginBottom: 5 }}>{r.name}</div>
          <div className="chips">
            {r.items.map((i) => (
              <span key={i.name} className={`chip chip-${i.role} static`}>
                {i.name}{i.role === "primary" && <span className="chip-star on">★</span>}
              </span>
            ))}
          </div>
        </div>
      ))}
    </div>
  );
}
