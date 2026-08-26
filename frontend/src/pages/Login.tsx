import { useState } from "react";
import { Link, Navigate, useNavigate } from "react-router-dom";
import { useAuth } from "../auth/AuthContext";
import { ErrorBox, msg } from "../components/ui";

export function Login() {
  const { session, login } = useAuth();
  const navigate = useNavigate();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  if (session) return <Navigate to="/workouts" replace />;

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    setBusy(true);
    setError(null);
    try {
      await login(email, password);
      navigate("/workouts", { replace: true });
    } catch (err) {
      setError(msg(err));
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="auth-wrap">
      <form className="card auth-card" onSubmit={submit}>
        <div className="auth-logo">TRAINING<span>APP</span></div>
        <div className="auth-sub">Учёт тренировок и нагрузки</div>
        <ErrorBox error={error} />
        <div className="field">
          <label htmlFor="email">Email</label>
          <input id="email" type="email" required value={email}
                 onChange={(e) => setEmail(e.target.value)} autoComplete="email" />
        </div>
        <div className="field">
          <label htmlFor="password">Пароль</label>
          <input id="password" type="password" required value={password}
                 onChange={(e) => setPassword(e.target.value)} autoComplete="current-password" />
        </div>
        <div className="field">
          <button className="btn-primary" style={{ width: "100%" }} disabled={busy}>
            {busy ? "Входим…" : "Войти"}
          </button>
        </div>
        <div className="small muted" style={{ textAlign: "center" }}>
          Нет аккаунта? <Link to="/register">Зарегистрироваться</Link>
        </div>
      </form>
    </div>
  );
}
