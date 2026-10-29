import { useState } from "react";
import { applyTheme, readTheme } from "../theme";
import type { Theme } from "../theme";

export function ThemeToggle() {
  const [theme, setTheme] = useState<Theme>(readTheme);

  function toggle() {
    const next: Theme = theme === "dark" ? "light" : "dark";
    applyTheme(next);
    setTheme(next);
  }

  return (
    <button className="theme-toggle" onClick={toggle}
            title={theme === "dark" ? "Светлая тема" : "Тёмная тема"}
            aria-label={theme === "dark" ? "Включить светлую тему" : "Включить тёмную тему"}>
      {theme === "dark" ? "☀" : "☾"}
    </button>
  );
}
