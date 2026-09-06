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
	"trainingApp/internal/plan"
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
		testDB.ExecContext(c, `DELETE FROM plans WHERE user_id = $1`, created.ID)
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

func TestSetRepoLastPerformance(t *testing.T) {
	repo := NewSetRepo(testDB)
	u := newUser(t)
	ex := newExercise(t, u.ID, "Упражнение с историей")

	older := newWorkout(t, u.ID, time.Now().Add(-72*time.Hour))
	recent := newWorkout(t, u.ID, time.Now().Add(-48*time.Hour))
	current := newWorkout(t, u.ID, time.Now().Add(-time.Hour))

	add := func(w workout.Workout, number int64, reps int, kg float32) {
		t.Helper()
		if _, err := repo.Create(ctx(t), u.ID, set.Set{
			ExerciseID: ex.ID, WorkoutID: w.ID, SetNumber: number, Reps: reps, Weight: kg,
		}); err != nil {
			t.Fatalf("подход: %v", err)
		}
	}

	add(older, 1, 10, 30)
	add(recent, 1, 10, 40)
	add(recent, 2, 8, 42.5)
	add(current, 1, 12, 45)

	got, err := repo.LastPerformance(ctx(t), u.ID, ex.ID, current.ID, nil)
	if err != nil {
		t.Fatalf("подсказка: %v", err)
	}
	if got.WorkoutID != recent.ID {
		t.Fatalf("взята тренировка %d, ожидалась предыдущая %d", got.WorkoutID, recent.ID)
	}
	if len(got.Sets) != 2 {
		t.Fatalf("подходов = %d, ожидалось 2", len(got.Sets))
	}
	if got.Sets[0].SetNumber != 1 || got.Sets[1].Weight != 42.5 {
		t.Fatalf("подходы вернулись не по порядку: %+v", got.Sets)
	}

	withoutExclude, err := repo.LastPerformance(ctx(t), u.ID, ex.ID, 0, nil)
	if err != nil {
		t.Fatalf("подсказка без исключения: %v", err)
	}
	if withoutExclude.WorkoutID != current.ID {
		t.Fatalf("без exclude ожидалась текущая тренировка %d, получена %d", current.ID, withoutExclude.WorkoutID)
	}
}

func TestSetRepoLastPerformanceIsolatesUsers(t *testing.T) {
	repo := NewSetRepo(testDB)
	owner := newUser(t)
	stranger := newUser(t)

	ex := newExercise(t, owner.ID, "Упражнение владельца")
	w := newWorkout(t, owner.ID, time.Now().Add(-24*time.Hour))
	if _, err := repo.Create(ctx(t), owner.ID, set.Set{
		ExerciseID: ex.ID, WorkoutID: w.ID, SetNumber: 1, Reps: 10, Weight: 60,
	}); err != nil {
		t.Fatalf("подход: %v", err)
	}

	if _, err := repo.LastPerformance(ctx(t), stranger.ID, ex.ID, 0, nil); !errors.Is(err, set.ErrNotFound) {
		t.Fatalf("посторонний получил %v, ожидалась ErrNotFound", err)
	}
}

func newPlan(t *testing.T, userID int64, items []plan.Item) plan.Plan {
	t.Helper()
	created, err := NewPlanRepo(testDB).Create(ctx(t), plan.Plan{
		UserID:    userID,
		Name:      fmt.Sprintf("План %d", time.Now().UnixNano()),
		Exercises: items,
	})
	if err != nil {
		t.Fatalf("создание плана: %v", err)
	}
	return created
}

func TestPlanRepoKeepsExerciseOrder(t *testing.T) {
	repo := NewPlanRepo(testDB)
	u := newUser(t)
	first := newExercise(t, u.ID, "Первое")
	second := newExercise(t, u.ID, "Второе")

	created := newPlan(t, u.ID, []plan.Item{
		{ExerciseID: second.ID, Position: 2, TargetSets: 3, TargetReps: 12},
		{ExerciseID: first.ID, Position: 1, TargetSets: 4, TargetReps: 10},
	})

	got, err := repo.GetByID(ctx(t), u.ID, created.ID)
	if err != nil {
		t.Fatalf("чтение плана: %v", err)
	}
	if len(got.Exercises) != 2 {
		t.Fatalf("упражнений = %d, ожидалось 2", len(got.Exercises))
	}
	if got.Exercises[0].ExerciseID != first.ID || got.Exercises[1].ExerciseID != second.ID {
		t.Fatalf("порядок не по position: %+v", got.Exercises)
	}
	if got.Exercises[0].ExerciseName == "" {
		t.Error("название упражнения не подтянулось из exercises")
	}
	if got.Exercises[0].TargetSets != 4 || got.Exercises[0].TargetReps != 10 {
		t.Errorf("цели потеряны: %+v", got.Exercises[0])
	}
}

func TestPlanRepoUpdateReplacesComposition(t *testing.T) {
	repo := NewPlanRepo(testDB)
	u := newUser(t)
	old := newExercise(t, u.ID, "Старое")
	fresh := newExercise(t, u.ID, "Новое")

	created := newPlan(t, u.ID, []plan.Item{
		{ExerciseID: old.ID, Position: 1, TargetSets: 4, TargetReps: 10},
	})

	updated, err := repo.Update(ctx(t), u.ID, plan.Plan{
		ID:   created.ID,
		Name: created.Name + " (правка)",
		Exercises: []plan.Item{
			{ExerciseID: fresh.ID, Position: 1, TargetSets: 5, TargetReps: 5},
		},
	})
	if err != nil {
		t.Fatalf("обновление плана: %v", err)
	}
	if updated.Name == created.Name {
		t.Error("имя не обновилось")
	}

	got, err := repo.GetByID(ctx(t), u.ID, created.ID)
	if err != nil {
		t.Fatalf("чтение плана: %v", err)
	}
	if len(got.Exercises) != 1 || got.Exercises[0].ExerciseID != fresh.ID {
		t.Fatalf("состав не заменён: %+v", got.Exercises)
	}
}

func TestPlanRepoRejectsForeignExercise(t *testing.T) {
	repo := NewPlanRepo(testDB)
	owner := newUser(t)
	stranger := newUser(t)
	foreign := newExercise(t, stranger.ID, "Личное постороннего")

	_, err := repo.Create(ctx(t), plan.Plan{
		UserID: owner.ID,
		Name:   fmt.Sprintf("План %d", time.Now().UnixNano()),
		Exercises: []plan.Item{
			{ExerciseID: foreign.ID, Position: 1, TargetSets: 4, TargetReps: 10},
		},
	})
	if !errors.Is(err, plan.ErrNotFound) {
		t.Fatalf("чужое упражнение в плане дало %v, ожидалась ErrNotFound", err)
	}
}

func TestPlanRepoIsolatesUsers(t *testing.T) {
	repo := NewPlanRepo(testDB)
	owner := newUser(t)
	stranger := newUser(t)
	ex := newExercise(t, owner.ID, "Упражнение владельца")
	created := newPlan(t, owner.ID, []plan.Item{
		{ExerciseID: ex.ID, Position: 1, TargetSets: 4, TargetReps: 10},
	})

	if _, err := repo.GetByID(ctx(t), stranger.ID, created.ID); !errors.Is(err, plan.ErrNotFound) {
		t.Fatalf("чужое чтение дало %v, ожидалась ErrNotFound", err)
	}
	if err := repo.Delete(ctx(t), stranger.ID, created.ID); !errors.Is(err, plan.ErrNotFound) {
		t.Fatalf("чужое удаление дало %v, ожидалась ErrNotFound", err)
	}
}

func TestWorkoutRepoRejectsForeignPlan(t *testing.T) {
	owner := newUser(t)
	stranger := newUser(t)
	ex := newExercise(t, owner.ID, "Упражнение плана")
	p := newPlan(t, owner.ID, []plan.Item{
		{ExerciseID: ex.ID, Position: 1, TargetSets: 4, TargetReps: 10},
	})

	_, err := NewWorkoutRepo(testDB).Create(ctx(t), workout.Workout{
		UserID:    stranger.ID,
		StartedAt: time.Now(),
		PlanID:    &p.ID,
	})
	if !errors.Is(err, workout.ErrPlanNotFound) {
		t.Fatalf("чужой план дал %v, ожидалась ErrPlanNotFound", err)
	}
}

func TestPlanDeleteKeepsWorkouts(t *testing.T) {
	u := newUser(t)
	ex := newExercise(t, u.ID, "Упражнение плана")
	p := newPlan(t, u.ID, []plan.Item{
		{ExerciseID: ex.ID, Position: 1, TargetSets: 4, TargetReps: 10},
	})

	workoutRepo := NewWorkoutRepo(testDB)
	w, err := workoutRepo.Create(ctx(t), workout.Workout{
		UserID: u.ID, StartedAt: time.Now().Add(-time.Hour), PlanID: &p.ID,
	})
	if err != nil {
		t.Fatalf("тренировка по плану: %v", err)
	}
	if w.PlanID == nil || *w.PlanID != p.ID {
		t.Fatalf("plan_id не сохранён: %+v", w.PlanID)
	}

	if err := NewPlanRepo(testDB).Delete(ctx(t), u.ID, p.ID); err != nil {
		t.Fatalf("удаление плана: %v", err)
	}

	got, err := workoutRepo.GetByID(ctx(t), u.ID, w.ID)
	if err != nil {
		t.Fatalf("тренировка должна пережить удаление плана: %v", err)
	}
	if got.PlanID != nil {
		t.Fatalf("plan_id = %v, ожидался NULL после удаления плана", *got.PlanID)
	}
}

func TestSetRepoLastPerformancePrefersSamePlan(t *testing.T) {
	repo := NewSetRepo(testDB)
	u := newUser(t)
	ex := newExercise(t, u.ID, "Упражнение плана")
	p := newPlan(t, u.ID, []plan.Item{
		{ExerciseID: ex.ID, Position: 1, TargetSets: 4, TargetReps: 10},
	})

	workoutRepo := NewWorkoutRepo(testDB)
	byPlan, err := workoutRepo.Create(ctx(t), workout.Workout{
		UserID: u.ID, StartedAt: time.Now().Add(-96 * time.Hour), PlanID: &p.ID,
	})
	if err != nil {
		t.Fatalf("тренировка по плану: %v", err)
	}
	withoutPlan := newWorkout(t, u.ID, time.Now().Add(-24*time.Hour))
	current := newWorkout(t, u.ID, time.Now())

	add := func(w workout.Workout, kg float32) {
		t.Helper()
		if _, err := repo.Create(ctx(t), u.ID, set.Set{
			ExerciseID: ex.ID, WorkoutID: w.ID, SetNumber: 1, Reps: 10, Weight: kg,
		}); err != nil {
			t.Fatalf("подход: %v", err)
		}
	}
	add(byPlan, 60)
	add(withoutPlan, 50)

	got, err := repo.LastPerformance(ctx(t), u.ID, ex.ID, current.ID, &p.ID)
	if err != nil {
		t.Fatalf("подсказка по плану: %v", err)
	}
	if got.WorkoutID != byPlan.ID {
		t.Fatalf("взята тренировка %d, ожидалась тренировка по тому же плану %d", got.WorkoutID, byPlan.ID)
	}

	any, err := repo.LastPerformance(ctx(t), u.ID, ex.ID, current.ID, nil)
	if err != nil {
		t.Fatalf("подсказка без плана: %v", err)
	}
	if any.WorkoutID != withoutPlan.ID {
		t.Fatalf("без плана ожидалась самая свежая тренировка %d, получена %d", withoutPlan.ID, any.WorkoutID)
	}
}

func TestSetRepoDuplicateNumberReturnsAlreadyExists(t *testing.T) {
	repo := NewSetRepo(testDB)
	u := newUser(t)
	ex := newExercise(t, u.ID, "Упражнение для дублей")
	w := newWorkout(t, u.ID, time.Now().Add(-time.Hour))

	first := set.Set{ExerciseID: ex.ID, WorkoutID: w.ID, SetNumber: 1, Reps: 10, Weight: 40}
	if _, err := repo.Create(ctx(t), u.ID, first); err != nil {
		t.Fatalf("первый подход: %v", err)
	}

	if _, err := repo.Create(ctx(t), u.ID, first); !errors.Is(err, set.ErrAlreadyExists) {
		t.Fatalf("повтор номера дал %v, ожидалась ErrAlreadyExists", err)
	}

	same := set.Set{ExerciseID: ex.ID, WorkoutID: w.ID, SetNumber: 2, Reps: 10, Weight: 40}
	second, err := repo.Create(ctx(t), u.ID, same)
	if err != nil {
		t.Fatalf("подход с теми же весом и повторами, но другим номером: %v", err)
	}

	if _, err := repo.Update(ctx(t), u.ID, set.Set{ID: second.ID, SetNumber: 1, Reps: 10, Weight: 40}); !errors.Is(err, set.ErrAlreadyExists) {
		t.Fatalf("перенумерация на занятый номер дала %v, ожидалась ErrAlreadyExists", err)
	}
}

func TestSetRepoListByWorkoutReturnsAllSets(t *testing.T) {
	repo := NewSetRepo(testDB)
	u := newUser(t)
	w := newWorkout(t, u.ID, time.Now().Add(-2*time.Hour))

	const exercises, perExercise = 7, 4
	for e := 0; e < exercises; e++ {
		ex := newExercise(t, u.ID, fmt.Sprintf("Упражнение %d", e))
		for n := 1; n <= perExercise; n++ {
			if _, err := repo.Create(ctx(t), u.ID, set.Set{
				ExerciseID: ex.ID, WorkoutID: w.ID, SetNumber: int64(n), Reps: 10, Weight: 40,
			}); err != nil {
				t.Fatalf("подход %d/%d: %v", e, n, err)
			}
		}
	}

	got, err := repo.ListByWorkout(ctx(t), u.ID, w.ID, 900)
	if err != nil {
		t.Fatalf("список подходов: %v", err)
	}
	if len(got) != exercises*perExercise {
		t.Fatalf("подходов = %d, ожидалось %d: выдача обрезается лимитом", len(got), exercises*perExercise)
	}

	byExercise := map[int64]int{}
	for _, s := range got {
		byExercise[s.ExerciseID]++
	}
	for id, count := range byExercise {
		if count != perExercise {
			t.Errorf("упражнение %d: подходов %d, ожидалось %d", id, count, perExercise)
		}
	}
}
