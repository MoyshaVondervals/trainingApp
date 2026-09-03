//go:build integration

package postgres

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"
	"trainingApp/internal/exercise"
	"trainingApp/internal/set"
	"trainingApp/internal/stats"
	"trainingApp/internal/user"
	"trainingApp/internal/weight"
	"trainingApp/internal/workout"

	"github.com/jmoiron/sqlx"
)

var testDB *sqlx.DB

func TestMain(m *testing.M) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		fmt.Println("TEST_DATABASE_URL не задан, интеграционные тесты пропущены")
		os.Exit(0)
	}

	db, err := Open(context.Background(), dsn)
	if err != nil {
		fmt.Println("не удалось подключиться к тестовой базе:", err)
		os.Exit(1)
	}
	testDB = db

	code := m.Run()
	db.Close()
	os.Exit(code)
}

func ctx(t *testing.T) context.Context {
	t.Helper()
	c, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(cancel)
	return c
}

func newUser(t *testing.T) user.User {
	t.Helper()
	repo := NewUserRepo(testDB)
	email := fmt.Sprintf("it-%d-%s@example.test", time.Now().UnixNano(), t.Name())
	created, err := repo.Create(ctx(t), user.User{
		Name:       "Тест",
		SecondName: "Тестов",
		Email:      email,
		Password:   "$2a$10$notarealhashnotarealhashnotarealhashnotarealhashnotare",
	})
	if err != nil {
		t.Fatalf("создание пользователя: %v", err)
	}

	t.Cleanup(func() {
		c := context.Background()
		testDB.ExecContext(c, `DELETE FROM sets WHERE workout_id IN (SELECT id FROM workouts WHERE user_id = $1)`, created.ID)
		testDB.ExecContext(c, `DELETE FROM sets WHERE exercise_id IN (SELECT id FROM exercises WHERE user_id = $1)`, created.ID)
		testDB.ExecContext(c, `DELETE FROM workouts WHERE user_id = $1`, created.ID)
		testDB.ExecContext(c, `DELETE FROM exercises WHERE user_id = $1`, created.ID)
		if _, err := testDB.ExecContext(c, `DELETE FROM users WHERE id = $1`, created.ID); err != nil {
			t.Errorf("очистка пользователя %d: %v", created.ID, err)
		}
	})
	return created
}

func newExercise(t *testing.T, userID int64, name string) exercise.Exercise {
	t.Helper()
	created, err := NewExerciseRepo(testDB).Create(ctx(t), exercise.Exercise{
		Name:   fmt.Sprintf("%s %d", name, time.Now().UnixNano()),
		UserID: &userID,
	})
	if err != nil {
		t.Fatalf("создание упражнения: %v", err)
	}
	return created
}

func newWorkout(t *testing.T, userID int64, startedAt time.Time) workout.Workout {
	t.Helper()
	created, err := NewWorkoutRepo(testDB).Create(ctx(t), workout.Workout{
		UserID:    userID,
		StartedAt: startedAt,
	})
	if err != nil {
		t.Fatalf("создание тренировки: %v", err)
	}
	return created
}

func groupIDByCode(t *testing.T, code string) int64 {
	t.Helper()
	var id int64
	if err := testDB.GetContext(ctx(t), &id, `SELECT id FROM muscle_groups WHERE code = $1`, code); err != nil {
		t.Fatalf("группа %q не найдена: %v", code, err)
	}
	return id
}

func TestWeightRepoUpsertReplacesSameDay(t *testing.T) {
	repo := NewWeightRepo(testDB)
	u := newUser(t)
	day := time.Date(2026, 8, 20, 0, 0, 0, 0, time.UTC)

	first, err := repo.Upsert(ctx(t), weight.BodyWeight{UserID: u.ID, WeightKg: 82.4, MeasuredOn: day, Note: "утром"})
	if err != nil {
		t.Fatalf("первая запись: %v", err)
	}

	second, err := repo.Upsert(ctx(t), weight.BodyWeight{UserID: u.ID, WeightKg: 80.5, MeasuredOn: day})
	if err != nil {
		t.Fatalf("повторная запись за тот же день: %v", err)
	}

	if first.ID != second.ID {
		t.Errorf("ожидалось обновление строки %d, создана новая %d", first.ID, second.ID)
	}
	if second.WeightKg != 80.5 {
		t.Errorf("weight_kg = %v, ожидалось 80.5", second.WeightKg)
	}
	if second.Note != "" {
		t.Errorf("note = %q, ожидалась пустая: обновление затирает старое значение", second.Note)
	}

	all, err := repo.List(ctx(t), u.ID, day.AddDate(0, 0, -1), day.AddDate(0, 0, 1), 10)
	if err != nil {
		t.Fatalf("список замеров: %v", err)
	}
	if len(all) != 1 {
		t.Fatalf("замеров = %d, ожидался 1", len(all))
	}
}

func TestWeightRepoListRespectsPeriodAndOrder(t *testing.T) {
	repo := NewWeightRepo(testDB)
	u := newUser(t)
	base := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)

	for i, kg := range []float32{80, 79.5, 79} {
		day := base.AddDate(0, 0, i*10)
		if _, err := repo.Upsert(ctx(t), weight.BodyWeight{UserID: u.ID, WeightKg: kg, MeasuredOn: day}); err != nil {
			t.Fatalf("запись замера: %v", err)
		}
	}

	got, err := repo.List(ctx(t), u.ID, base, base.AddDate(0, 0, 10), 10)
	if err != nil {
		t.Fatalf("список замеров: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("замеров в периоде = %d, ожидалось 2", len(got))
	}
	if !got[0].MeasuredOn.After(got[1].MeasuredOn) {
		t.Errorf("порядок не по убыванию даты: %v, %v", got[0].MeasuredOn, got[1].MeasuredOn)
	}
}

func TestWeightRepoDeleteIsolatesUsers(t *testing.T) {
	repo := NewWeightRepo(testDB)
	owner := newUser(t)
	stranger := newUser(t)
	day := time.Date(2026, 8, 25, 0, 0, 0, 0, time.UTC)

	created, err := repo.Upsert(ctx(t), weight.BodyWeight{UserID: owner.ID, WeightKg: 77, MeasuredOn: day})
	if err != nil {
		t.Fatalf("запись замера: %v", err)
	}

	if err := repo.Delete(ctx(t), stranger.ID, created.ID); !errors.Is(err, weight.ErrNotFound) {
		t.Fatalf("чужое удаление вернуло %v, ожидалась ErrNotFound", err)
	}
	if err := repo.Delete(ctx(t), owner.ID, created.ID); err != nil {
		t.Fatalf("удаление владельцем: %v", err)
	}
	if err := repo.Delete(ctx(t), owner.ID, created.ID); !errors.Is(err, weight.ErrNotFound) {
		t.Fatalf("повторное удаление вернуло %v, ожидалась ErrNotFound", err)
	}
}

func TestWorkoutRepoFinishIsIdempotentlyRejected(t *testing.T) {
	repo := NewWorkoutRepo(testDB)
	u := newUser(t)
	w := newWorkout(t, u.ID, time.Now().Add(-time.Hour))

	finished, err := repo.FinishTraining(ctx(t), u.ID, w.ID)
	if err != nil {
		t.Fatalf("завершение тренировки: %v", err)
	}
	if finished.EndedAt == nil {
		t.Fatal("ended_at не проставлен")
	}

	if _, err := repo.FinishTraining(ctx(t), u.ID, w.ID); !errors.Is(err, workout.ErrNotFound) {
		t.Fatalf("повторное завершение вернуло %v, ожидалась ErrNotFound", err)
	}
}

func TestWorkoutRepoIsolatesUsers(t *testing.T) {
	repo := NewWorkoutRepo(testDB)
	owner := newUser(t)
	stranger := newUser(t)
	w := newWorkout(t, owner.ID, time.Now().Add(-2*time.Hour))

	if _, err := repo.GetByID(ctx(t), stranger.ID, w.ID); !errors.Is(err, workout.ErrNotFound) {
		t.Fatalf("чужое чтение вернуло %v, ожидалась ErrNotFound", err)
	}
	if err := repo.Delete(ctx(t), stranger.ID, w.ID); !errors.Is(err, workout.ErrNotFound) {
		t.Fatalf("чужое удаление вернуло %v, ожидалась ErrNotFound", err)
	}
	if _, err := repo.GetByID(ctx(t), owner.ID, w.ID); err != nil {
		t.Fatalf("тренировка владельца должна остаться: %v", err)
	}
}

func TestSetRepoRejectsForeignWorkout(t *testing.T) {
	repo := NewSetRepo(testDB)
	owner := newUser(t)
	stranger := newUser(t)

	ex := newExercise(t, owner.ID, "Жим для теста")
	w := newWorkout(t, owner.ID, time.Now().Add(-time.Hour))

	_, err := repo.Create(ctx(t), stranger.ID, set.Set{
		ExerciseID: ex.ID,
		WorkoutID:  w.ID,
		SetNumber:  1,
		Reps:       10,
		Weight:     50,
	})
	if err == nil {
		t.Fatal("подход в чужую тренировку должен отклоняться")
	}

	created, err := repo.Create(ctx(t), owner.ID, set.Set{
		ExerciseID: ex.ID,
		WorkoutID:  w.ID,
		SetNumber:  1,
		Reps:       10,
		Weight:     50,
	})
	if err != nil {
		t.Fatalf("подход владельца: %v", err)
	}
	if created.ID == 0 {
		t.Fatal("идентификатор подхода не заполнен")
	}
}

func TestSetRepoRejectsDuplicateSetNumber(t *testing.T) {
	repo := NewSetRepo(testDB)
	u := newUser(t)
	ex := newExercise(t, u.ID, "Тяга для теста")
	w := newWorkout(t, u.ID, time.Now().Add(-time.Hour))

	s := set.Set{ExerciseID: ex.ID, WorkoutID: w.ID, SetNumber: 1, Reps: 12, Weight: 40}
	if _, err := repo.Create(ctx(t), u.ID, s); err != nil {
		t.Fatalf("первый подход: %v", err)
	}
	if _, err := repo.Create(ctx(t), u.ID, s); err == nil {
		t.Fatal("повтор номера подхода в одном упражнении должен нарушать UNIQUE")
	}
}

func TestExerciseMusclesReplaceIsAtomic(t *testing.T) {
	repo := NewExerciseMusclesRepo(testDB)
	u := newUser(t)
	ex := newExercise(t, u.ID, "Упражнение с мышцами")

	lats := groupIDByCode(t, "lats")
	biceps := groupIDByCode(t, "biceps")
	traps := groupIDByCode(t, "traps_middle")

	first := []exercise.Muscle{
		{MuscleGroupID: lats, Role: exercise.RolePrimary},
		{MuscleGroupID: biceps, Role: exercise.RoleSecondary},
	}
	if err := repo.ReplaceForExercise(ctx(t), u.ID, ex.ID, first); err != nil {
		t.Fatalf("первичная запись набора: %v", err)
	}

	second := []exercise.Muscle{{MuscleGroupID: traps, Role: exercise.RolePrimary}}
	if err := repo.ReplaceForExercise(ctx(t), u.ID, ex.ID, second); err != nil {
		t.Fatalf("замена набора: %v", err)
	}

	got, err := repo.ListByExercise(ctx(t), u.ID, ex.ID)
	if err != nil {
		t.Fatalf("чтение набора: %v", err)
	}
	if len(got) != 1 || got[0].MuscleGroupID != traps {
		t.Fatalf("набор = %+v, ожидалась единственная группа %d", got, traps)
	}
}

func TestExerciseMusclesReplaceRejectsForeignExercise(t *testing.T) {
	repo := NewExerciseMusclesRepo(testDB)
	owner := newUser(t)
	stranger := newUser(t)
	ex := newExercise(t, owner.ID, "Чужое упражнение")

	lats := groupIDByCode(t, "lats")
	err := repo.ReplaceForExercise(ctx(t), stranger.ID, ex.ID, []exercise.Muscle{
		{MuscleGroupID: lats, Role: exercise.RolePrimary},
	})
	if !errors.Is(err, exercise.ErrNotFound) {
		t.Fatalf("замена чужого набора вернула %v, ожидалась ErrNotFound", err)
	}

	got, err := repo.ListByExercise(ctx(t), owner.ID, ex.ID)
	if err != nil {
		t.Fatalf("чтение набора: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("набор изменён посторонним: %+v", got)
	}
}

func TestExerciseMusclesListGroupsReturnsSeed(t *testing.T) {
	groups, err := NewExerciseMusclesRepo(testDB).ListGroups(ctx(t))
	if err != nil {
		t.Fatalf("справочник групп: %v", err)
	}
	if len(groups) < 40 {
		t.Fatalf("групп в справочнике = %d, ожидалось наполнение сидами", len(groups))
	}

	first := groups[0]
	if first.Code == "" || first.Name == "" || first.RegionCode == "" || first.RegionName == "" {
		t.Fatalf("неполная запись справочника: %+v", first)
	}
}

func TestStatsRepoSummaryAndMuscleLoad(t *testing.T) {
	u := newUser(t)
	ex := newExercise(t, u.ID, "Упражнение для статистики")

	lats := groupIDByCode(t, "lats")
	biceps := groupIDByCode(t, "biceps")
	if err := NewExerciseMusclesRepo(testDB).ReplaceForExercise(ctx(t), u.ID, ex.ID, []exercise.Muscle{
		{MuscleGroupID: lats, Role: exercise.RolePrimary},
		{MuscleGroupID: biceps, Role: exercise.RoleSecondary},
	}); err != nil {
		t.Fatalf("разметка мышц: %v", err)
	}

	startedAt := time.Now().Add(-24 * time.Hour)
	w := newWorkout(t, u.ID, startedAt)

	setRepo := NewSetRepo(testDB)
	for i, reps := range []int{10, 8} {
		_, err := setRepo.Create(ctx(t), u.ID, set.Set{
			ExerciseID: ex.ID,
			WorkoutID:  w.ID,
			SetNumber:  int64(i + 1),
			Reps:       reps,
			Weight:     50,
		})
		if err != nil {
			t.Fatalf("подход %d: %v", i+1, err)
		}
	}

	period, err := stats.NewPeriod(startedAt.Add(-time.Hour), time.Now().Add(time.Hour))
	if err != nil {
		t.Fatalf("период: %v", err)
	}

	repo := NewStatsRepo(testDB)

	summary, err := repo.Summary(ctx(t), u.ID, period)
	if err != nil {
		t.Fatalf("сводка: %v", err)
	}
	if summary.Workouts != 1 || summary.Sets != 2 || summary.Reps != 18 {
		t.Errorf("сводка = %+v, ожидалось 1 тренировка, 2 подхода, 18 повторов", summary)
	}
	if summary.Volume != 900 {
		t.Errorf("объём = %v, ожидалось 900", summary.Volume)
	}

	rows, err := repo.MuscleLoad(ctx(t), u.ID, period)
	if err != nil {
		t.Fatalf("нагрузка по группам: %v", err)
	}

	byRole := map[string]stats.GroupRoleLoad{}
	for _, row := range rows {
		byRole[row.Role] = row
	}
	if len(byRole) != 2 {
		t.Fatalf("ожидались строки для обеих ролей, получено %+v", rows)
	}

	loads := stats.AggregateByGroup(rows)
	var latsVolume, bicepsVolume float64
	for _, l := range loads {
		switch l.Code {
		case "lats":
			latsVolume = l.Volume
		case "biceps":
			bicepsVolume = l.Volume
		}
	}
	if latsVolume != 900 {
		t.Errorf("объём primary-группы = %v, ожидалось 900", latsVolume)
	}
	if bicepsVolume != 450 {
		t.Errorf("объём secondary-группы = %v, ожидалось 450 (коэффициент 0,5)", bicepsVolume)
	}
}

func TestStatsRepoIgnoresOtherUsers(t *testing.T) {
	owner := newUser(t)
	stranger := newUser(t)

	ex := newExercise(t, owner.ID, "Упражнение чужого пользователя")
	w := newWorkout(t, owner.ID, time.Now().Add(-3*time.Hour))
	if _, err := NewSetRepo(testDB).Create(ctx(t), owner.ID, set.Set{
		ExerciseID: ex.ID, WorkoutID: w.ID, SetNumber: 1, Reps: 5, Weight: 20,
	}); err != nil {
		t.Fatalf("подход: %v", err)
	}

	period, err := stats.NewPeriod(time.Now().Add(-24*time.Hour), time.Now().Add(time.Hour))
	if err != nil {
		t.Fatalf("период: %v", err)
	}

	summary, err := NewStatsRepo(testDB).Summary(ctx(t), stranger.ID, period)
	if err != nil {
		t.Fatalf("сводка постороннего: %v", err)
	}
	if summary.Workouts != 0 || summary.Sets != 0 || summary.Volume != 0 {
		t.Fatalf("посторонний видит чужие данные: %+v", summary)
	}
}
