import { useMemo, useState } from "react";
import type { BodyWeight } from "../api/types";

type Point = { x: number; y: number; value: number; date: Date; note: string };

const W = 720;
const H = 260;
const PAD = { top: 18, right: 18, bottom: 30, left: 44 };

function niceBounds(min: number, max: number): [number, number] {
  if (min === max) return [min - 1, max + 1];
  const pad = (max - min) * 0.15;
  return [min - pad, max + pad];
}

function fmtDay(d: Date): string {
  return d.toLocaleDateString("ru-RU", { day: "2-digit", month: "short" });
}

export function WeightChart({ items }: { items: BodyWeight[] }) {
  const [hover, setHover] = useState<number | null>(null);

  const points = useMemo<Point[]>(() => {
    const sorted = [...items].sort(
      (a, b) => new Date(a.measured_on).getTime() - new Date(b.measured_on).getTime(),
    );
    if (sorted.length === 0) return [];

    const times = sorted.map((w) => new Date(w.measured_on).getTime());
    const values = sorted.map((w) => w.weight_kg);
    const [lo, hi] = niceBounds(Math.min(...values), Math.max(...values));
    const tMin = times[0];
    const tMax = times[times.length - 1];
    const spanT = tMax - tMin || 1;
    const spanV = hi - lo || 1;
    const plotW = W - PAD.left - PAD.right;
    const plotH = H - PAD.top - PAD.bottom;

    return sorted.map((w, i) => ({
      x: PAD.left + ((times[i] - tMin) / spanT) * plotW,
      y: PAD.top + (1 - (values[i] - lo) / spanV) * plotH,
      value: values[i],
      date: new Date(w.measured_on),
      note: w.note,
    }));
  }, [items]);

  const ticks = useMemo(() => {
    if (points.length === 0) return [];
    const values = points.map((p) => p.value);
    const [lo, hi] = niceBounds(Math.min(...values), Math.max(...values));
    const plotH = H - PAD.top - PAD.bottom;
    return [0, 0.25, 0.5, 0.75, 1].map((f) => ({
      y: PAD.top + (1 - f) * plotH,
      label: (lo + f * (hi - lo)).toFixed(1),
    }));
  }, [points]);

  if (points.length === 0) {
    return <div className="empty">Замеров пока нет — добавьте первый.</div>;
  }

  const line = points.map((p, i) => `${i === 0 ? "M" : "L"}${p.x.toFixed(1)},${p.y.toFixed(1)}`).join(" ");
  const area = `${line} L${points[points.length - 1].x.toFixed(1)},${H - PAD.bottom} L${points[0].x.toFixed(1)},${H - PAD.bottom} Z`;
  const active = hover !== null ? points[hover] : null;

  const xLabels = points.length <= 4
    ? points.map((p, i) => ({ p, i }))
    : [0, Math.floor((points.length - 1) / 2), points.length - 1].map((i) => ({ p: points[i], i }));

  function onMove(e: React.MouseEvent<SVGSVGElement>) {
    const rect = e.currentTarget.getBoundingClientRect();
    const x = ((e.clientX - rect.left) / rect.width) * W;
    let best = 0;
    for (let i = 1; i < points.length; i++) {
      if (Math.abs(points[i].x - x) < Math.abs(points[best].x - x)) best = i;
    }
    setHover(best);
  }

  return (
    <div className="chart-wrap">
      <svg viewBox={`0 0 ${W} ${H}`} className="chart" role="img"
           aria-label="Динамика веса тела"
           onMouseMove={onMove} onMouseLeave={() => setHover(null)}>
        <defs>
          <linearGradient id="weight-fill" x1="0" y1="0" x2="0" y2="1">
            <stop offset="0%" stopColor="var(--orange)" stopOpacity="0.28" />
            <stop offset="100%" stopColor="var(--orange)" stopOpacity="0" />
          </linearGradient>
        </defs>

        {ticks.map((t) => (
          <g key={t.label}>
            <line x1={PAD.left} y1={t.y} x2={W - PAD.right} y2={t.y}
                  stroke="var(--graphite-600)" strokeWidth="1" />
            <text x={PAD.left - 8} y={t.y + 4} textAnchor="end"
                  fill="var(--text-dim)" fontSize="11">{t.label}</text>
          </g>
        ))}

        {xLabels.map(({ p, i }) => (
          <text key={i} x={p.x} y={H - 10} textAnchor="middle"
                fill="var(--text-dim)" fontSize="11">{fmtDay(p.date)}</text>
        ))}

        <path d={area} fill="url(#weight-fill)" />
        <path d={line} fill="none" stroke="var(--orange)" strokeWidth="2"
              strokeLinejoin="round" strokeLinecap="round" />

        {points.map((p, i) => (
          <circle key={i} cx={p.x} cy={p.y} r={active && hover === i ? 5.5 : 4}
                  fill="var(--orange)" stroke="var(--graphite-800)" strokeWidth="2" />
        ))}

        {active && (
          <line x1={active.x} y1={PAD.top} x2={active.x} y2={H - PAD.bottom}
                stroke="var(--teal)" strokeWidth="1" strokeDasharray="3 3" />
        )}
      </svg>

      {active && (
        <div className="chart-tip" style={{ left: `${(active.x / W) * 100}%` }}>
          <strong>{active.value.toFixed(1)} кг</strong>
          <span className="muted"> · {fmtDay(active.date)}</span>
          {active.note && <div className="small muted">{active.note}</div>}
        </div>
      )}
    </div>
  );
}
