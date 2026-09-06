import { BrowserRouter, Navigate, Route, Routes } from "react-router-dom";
import { AuthProvider } from "./auth/AuthContext";
import { RequireAuth } from "./auth/RequireAuth";
import { Layout } from "./components/Layout";
import { Login } from "./pages/Login";
import { Register } from "./pages/Register";
import { Workouts } from "./pages/Workouts";
import { WorkoutDetail } from "./pages/WorkoutDetail";
import { Exercises } from "./pages/Exercises";
import { Stats } from "./pages/Stats";
import { Weight } from "./pages/Weight";
import { Plans } from "./pages/Plans";

export function App() {
  return (
    <BrowserRouter>
      <AuthProvider>
        <Routes>
          <Route path="/login" element={<Login />} />
          <Route path="/register" element={<Register />} />
          <Route element={<RequireAuth><Layout /></RequireAuth>}>
            <Route path="/workouts" element={<Workouts />} />
            <Route path="/workouts/:id" element={<WorkoutDetail />} />
            <Route path="/exercises" element={<Exercises />} />
            <Route path="/stats" element={<Stats />} />
            <Route path="/weight" element={<Weight />} />
            <Route path="/plans" element={<Plans />} />
          </Route>
          <Route path="*" element={<Navigate to="/workouts" replace />} />
        </Routes>
      </AuthProvider>
    </BrowserRouter>
  );
}
