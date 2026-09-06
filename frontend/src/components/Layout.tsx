import { Link, NavLink, Outlet } from "react-router-dom";
import { useAuth } from "../auth/AuthContext";

export function Layout() {
  const { session, logout } = useAuth();
  return (
    <div className="app">
      <header className="topbar">
        <Link to="/workouts" className="brand-link" title="К тренировкам">
          <div className="brand">TRAINING<span>APP</span></div>
        </Link>
        <nav className="nav">
          <NavLink to="/workouts">Тренировки</NavLink>
          <NavLink to="/exercises">Упражнения</NavLink>
          <NavLink to="/plans">Планы</NavLink>
          <NavLink to="/weight">Вес</NavLink>
          <NavLink to="/stats">Статистика</NavLink>
        </nav>
        <div className="row">
          <span className="user-chip">{session?.email}</span>
          <button className="btn-ghost btn-sm" onClick={logout}>Выйти</button>
        </div>
      </header>
      <main className="content"><Outlet /></main>
    </div>
  );
}
