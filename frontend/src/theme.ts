export type Theme = "dark" | "light";

const THEME_KEY = "trainingapp.theme";

export function readTheme(): Theme {
  try {
    const stored = localStorage.getItem(THEME_KEY);
    if (stored === "dark" || stored === "light") return stored;
  } catch {
  }
  return "dark";
}

export function applyTheme(theme: Theme) {
  document.documentElement.dataset.theme = theme;
  const meta = document.querySelector('meta[name="theme-color"]');
  meta?.setAttribute("content", theme === "light" ? "#f4f5f7" : "#101215");
  try {
    localStorage.setItem(THEME_KEY, theme);
  } catch {
  }
}
