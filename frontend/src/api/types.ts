export type User = {
  id: number;
  name: string;
  second_name: string;
  email: string;
  created_at: string;
};

export type LoginResponse = { token: string; expires_at: string };

export type Exercise = {
  id: number;
  name: string;
  description: string;
  user_id: number | null;
  created_at: string;
};

export type Muscle = { muscle_group_id: number; role: "primary" | "secondary" };

export type Workout = {
  id: number;
  user_id: number;
  started_at: string;
  ended_at: string | null;
  note: string;
};

export type Set = {
  id: number;
  exercise_id: number;
  workout_id: number;
  set_number: number;
  reps: number;
  weight: number;
  created_at: string;
};

export type Dashboard = {
  period: { from: string; to: string };
  summary: { workouts: number; sets: number; reps: number; volume: number };
  muscles: MuscleLoad[];
  records: Record_[];
};

export type MuscleLoad = {
  code: string;
  name: string;
  region: string;
  volume: number;
  reps: number;
  sets: number;
};

export type Record_ = {
  exercise_id: number;
  exercise_name: string;
  weight_kg: number | null;
  reps: number;
  achieved_at: string;
};

export type MuscleGroup = {
  id: number;
  code: string;
  name: string;
  region_code: string;
  region_name: string;
};
