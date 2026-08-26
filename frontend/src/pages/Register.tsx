import { useState } from "react";
import { Link, Navigate, useNavigate } from "react-router-dom";
import { useAuth } from "../auth/AuthContext";
import { ErrorBox, msg } from "../components/ui";

export function Register() {
  const { session, register } = useAuth();
  const navigate = useNavigate();
  const [form, setForm] = useState({ username: "", second_name: "", email: "", password: "" });
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  if (session) return <Navigate to="/workouts" replace />;

  function field(key: keyof typeof form) {
    return {
      value: form[key],
      onChange: (e: React.ChangeEvent<HTMLInputElement>) =>
        setForm({ ...form, [key]: e.target.value }),
    };
  }

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    setBusy(true);
    setError(null);
    try {
      await register(form);
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
        <div className="auth-sub">Создание аккаунта</div>
        <ErrorBox error={error} />
        <div className="field">
          <label htmlFor="username">Имя</label>
          <input id="username" required maxLength={100} {...field("username")} />
        </div>
        <div className="field">
          <label htmlFor="second_name">Фамилия</label>
          <input id="second_name" required maxLength={100} {...field("second_name")} />
        </div>
        <div className="field">
          <label htmlFor="email">Email</label>
          <input id="email" type="email" required maxLength={255} {...field("email")} />
        </div>
        <div className="field">
          <label htmlFor="password">Пароль (минимум 8 символов)</label>
          <input id="password" type="password" required minLength={8} maxLength={72}
                 autoComplete="new-password" {...field("password")} />
        </div>
        <div className="field">
          <button className="btn-primary" style={{ width: "100%" }} disabled={busy}>
            {busy ? "Создаём…" : "Зарегистрироваться"}
          </button>
        </div>
        <div className="small muted" style={{ textAlign: "center" }}>
          Уже есть аккаунт? <Link to="/login">Войти</Link>
        </div>
      </form>
    </div>
  );
}
