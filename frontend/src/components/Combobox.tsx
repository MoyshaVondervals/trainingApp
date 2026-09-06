import { useEffect, useMemo, useRef, useState } from "react";

export type ComboOption = { id: number; label: string; hint?: string };

type Props = {
  id?: string;
  options: ComboOption[];
  value: number | "";
  onChange: (id: number) => void;
  placeholder?: string;
  emptyText?: string;
  disabled?: boolean;
};

function normalize(s: string): string {
  return s.toLowerCase().replace("ё", "е").trim();
}

export function Combobox({
  id,
  options,
  value,
  onChange,
  placeholder = "начните вводить название",
  emptyText = "ничего не найдено",
  disabled,
}: Props) {
  const [open, setOpen] = useState(false);
  const [query, setQuery] = useState("");
  const [highlight, setHighlight] = useState(0);
  const wrapRef = useRef<HTMLDivElement>(null);
  const listRef = useRef<HTMLUListElement>(null);

  const selected = options.find((o) => o.id === value);

  const filtered = useMemo(() => {
    const q = normalize(query);
    if (q === "") return options;
    return options.filter((o) => normalize(o.label).includes(q));
  }, [options, query]);

  useEffect(() => {
    if (!open) return;
    const onDocDown = (e: MouseEvent) => {
      if (!wrapRef.current?.contains(e.target as Node)) setOpen(false);
    };
    document.addEventListener("mousedown", onDocDown);
    return () => document.removeEventListener("mousedown", onDocDown);
  }, [open]);

  useEffect(() => {
    if (!open) return;
    const el = listRef.current?.children[highlight] as HTMLElement | undefined;
    el?.scrollIntoView({ block: "nearest" });
  }, [highlight, open]);

  function choose(option: ComboOption) {
    onChange(option.id);
    setQuery("");
    setOpen(false);
  }

  function onKeyDown(e: React.KeyboardEvent<HTMLInputElement>) {
    if (e.key === "ArrowDown" || e.key === "ArrowUp") {
      e.preventDefault();
      if (!open) { setOpen(true); return; }
      const delta = e.key === "ArrowDown" ? 1 : -1;
      setHighlight((h) => {
        const next = h + delta;
        if (next < 0) return filtered.length - 1;
        if (next >= filtered.length) return 0;
        return next;
      });
      return;
    }
    if (e.key === "Enter") {
      if (open && filtered[highlight]) {
        e.preventDefault();
        choose(filtered[highlight]);
      }
      return;
    }
    if (e.key === "Escape") {
      setOpen(false);
      setQuery("");
    }
  }

  return (
    <div className="combo" ref={wrapRef}>
      <input
        id={id}
        className="combo-input"
        type="text"
        role="combobox"
        aria-expanded={open}
        aria-controls={id ? `${id}-list` : undefined}
        autoComplete="off"
        disabled={disabled}
        placeholder={selected ? undefined : placeholder}
        value={open ? query : selected?.label ?? ""}
        onFocus={() => { setOpen(true); setHighlight(0); }}
        onChange={(e) => { setQuery(e.target.value); setOpen(true); setHighlight(0); }}
        onKeyDown={onKeyDown}
      />
      <button type="button" className="combo-toggle" tabIndex={-1} disabled={disabled}
              aria-label={open ? "Закрыть список" : "Открыть список"}
              onClick={() => setOpen((v) => !v)}>
        ▾
      </button>

      {open && (
        <ul className="combo-list" id={id ? `${id}-list` : undefined} role="listbox" ref={listRef}>
          {filtered.length === 0 && <li className="combo-empty">{emptyText}</li>}
          {filtered.map((o, i) => (
            <li key={o.id}
                role="option"
                aria-selected={o.id === value}
                className={`combo-option ${i === highlight ? "active" : ""} ${o.id === value ? "chosen" : ""}`}
                onMouseEnter={() => setHighlight(i)}
                onMouseDown={(e) => { e.preventDefault(); choose(o); }}>
              <span>{o.label}</span>
              {o.hint && <span className="combo-hint">{o.hint}</span>}
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
