import type {
  BodyWeight, Dashboard, Exercise, ExerciseWithRole, LastPerformance, LoginResponse, Muscle,
  MuscleGroup, Plan, PlanItem, Set, User, Workout,
} from "./types";

const TOKEN_KEY = "trainingapp.token";

export function getToken(): string | null {
  return localStorage.getItem(TOKEN_KEY);
}

export function setToken(token: string | null) {
  if (token) localStorage.setItem(TOKEN_KEY, token);
  else localStorage.removeItem(TOKEN_KEY);
}

export class ApiError extends Error {
  constructor(public status: number, message: string) {
    super(message);
  }
}

const serverMessages: Record<string, string> = {
  "no account with this email": "Аккаунта с таким email нет",
  "invalid password": "Неверный пароль",
  "email already registered": "Этот email уже зарегистрирован",
};

let onUnauthorized: () => void = () => {};
export function setUnauthorizedHandler(fn: () => void) {
  onUnauthorized = fn;
}

async function request<T>(method: string, path: string, body?: unknown): Promise<T> {
  const headers: Record<string, string> = {};
  const token = getToken();
  if (token) headers["Authorization"] = `Bearer ${token}`;
  if (body !== undefined) headers["Content-Type"] = "application/json";

  const res = await fetch(path, {
    method,
    headers,
    body: body === undefined ? undefined : JSON.stringify(body),
  });

  const isAuthAttempt = path === "/api/v1/auth/login" || path === "/api/v1/auth/register";
  if (res.status === 401 && !isAuthAttempt) {
    onUnauthorized();
    throw new ApiError(401, "Сессия истекла, войдите заново");
  }
  if (res.status === 204) return undefined as T;

  const text = await res.text();
  const data = text ? JSON.parse(text) : null;

  if (!res.ok) {
    const msg = data && typeof data.error === "string" ? data.error : `Ошибка ${res.status}`;
    throw new ApiError(res.status, serverMessages[msg] ?? msg);
  }
  return data as T;
}

export const api = {
  register: (b: { username: string; second_name: string; email: string; password: string }) =>
    request<User>("POST", "/api/v1/auth/register", b),
  login: (b: { email: string; password: string }) =>
    request<LoginResponse>("POST", "/api/v1/auth/login", b),
  refresh: () => request<LoginResponse>("POST", "/api/v1/auth/refresh"),

  exercises: (limit = 100) => request<Exercise[]>("GET", `/api/v1/exercises?limit=${limit}`),
  exercise: (id: number) => request<Exercise>("GET", `/api/v1/exercises/${id}`),
  createExercise: (b: { name: string; description?: string }) =>
    request<Exercise>("POST", "/api/v1/exercises", b),
  updateExercise: (id: number, b: { name: string; description?: string }) =>
    request<Exercise>("PATCH", `/api/v1/exercises/${id}`, b),
  deleteExercise: (id: number) => request<void>("DELETE", `/api/v1/exercises/${id}`),
  muscles: (id: number) => request<Muscle[]>("GET", `/api/v1/exercises/${id}/muscles`),
  setMuscles: (id: number, b: Muscle[]) =>
    request<Muscle[]>("PUT", `/api/v1/exercises/${id}/muscles`, b),
  muscleGroups: () => request<MuscleGroup[]>("GET", "/api/v1/muscle-groups"),
  exercisesByMuscle: (code: string, limit = 20) =>
    request<ExerciseWithRole[]>("GET", `/api/v1/muscle-groups/${encodeURIComponent(code)}/exercises?limit=${limit}`),

  workouts: (limit = 100) => request<Workout[]>("GET", `/api/v1/workouts?limit=${limit}`),
  workout: (id: number) => request<Workout>("GET", `/api/v1/workouts/${id}`),
  createWorkout: (b: { started_at?: string; note?: string; plan_id?: number }) =>
    request<Workout>("POST", "/api/v1/workouts", b),
  updateWorkout: (id: number, b: { note?: string; started_at?: string }) =>
    request<Workout>("PATCH", `/api/v1/workouts/${id}`, b),
  deleteWorkout: (id: number) => request<void>("DELETE", `/api/v1/workouts/${id}`),
  finishWorkout: (id: number) => request<Workout>("POST", `/api/v1/workouts/finish/${id}`),

  setsByWorkout: (id: number) => request<Set[]>("GET", `/api/v1/sets/byWorkout/${id}`),
  lastSets: (exerciseId: number, excludeWorkoutId?: number, planId?: number | null) => {
    const q = new URLSearchParams();
    if (excludeWorkoutId) q.set("exclude_workout", String(excludeWorkoutId));
    if (planId) q.set("plan_id", String(planId));
    const s = q.toString();
    return request<LastPerformance>("GET", `/api/v1/sets/last/${exerciseId}${s ? `?${s}` : ""}`);
  },

  plans: (limit = 50) => request<Plan[]>("GET", `/api/v1/plans?limit=${limit}`),
  plan: (id: number) => request<Plan>("GET", `/api/v1/plans/${id}`),
  createPlan: (b: { name: string; note?: string; exercises: PlanItem[] }) =>
    request<Plan>("POST", "/api/v1/plans", b),
  updatePlan: (id: number, b: { name: string; note?: string; exercises: PlanItem[] }) =>
    request<Plan>("PUT", `/api/v1/plans/${id}`, b),
  deletePlan: (id: number) => request<void>("DELETE", `/api/v1/plans/${id}`),
  createSet: (b: {
    exercise_id: number; workout_id: number; set_number: number; reps: number; weight: number;
  }) => request<Set>("POST", "/api/v1/sets", b),
  updateSet: (id: number, b: { set_number: number; reps: number; weight: number }) =>
    request<Set>("PATCH", `/api/v1/sets/${id}`, b),
  deleteSet: (id: number) => request<void>("DELETE", `/api/v1/sets/${id}`),

  weights: (from?: string, to?: string, limit = 100) => {
    const q = new URLSearchParams({ limit: String(limit) });
    if (from) q.set("from", from);
    if (to) q.set("to", to);
    return request<BodyWeight[]>("GET", `/api/v1/weights?${q.toString()}`);
  },
  createWeight: (b: { weight_kg: number; measured_on?: string; note?: string }) =>
    request<BodyWeight>("POST", "/api/v1/weights", b),
  deleteWeight: (id: number) => request<void>("DELETE", `/api/v1/weights/${id}`),

  stats: (from?: string, to?: string) => {
    const q = new URLSearchParams();
    if (from) q.set("from", from);
    if (to) q.set("to", to);
    const s = q.toString();
    return request<Dashboard>("GET", `/api/v1/stats${s ? `?${s}` : ""}`);
  },
};
