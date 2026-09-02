package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	_ "github.com/lib/pq"
)

var db *sql.DB

// ---------- модели ----------

type Exercise struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

type Workout struct {
	ID        int        `json:"id"`
	StartedAt time.Time  `json:"started_at"`
	EndedAt   *time.Time `json:"ended_at"`
}

type WorkoutSet struct {
	ID         int     `json:"id"`
	WorkoutID  int     `json:"workout_id"`
	ExerciseID int     `json:"exercise_id"`
	Weight     float64 `json:"weight"`
	Reps       int     `json:"reps"`
}

// ---------- вспомогательные функции ----------

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func enableCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// ---------- обработчики: упражнения ----------

func listExercises(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query(`SELECT id, name, created_at FROM exercises ORDER BY name`)
	if err != nil {
		writeError(w, 500, err.Error())
		return
	}
	defer rows.Close()

	exercises := []Exercise{}
	for rows.Next() {
		var e Exercise
		if err := rows.Scan(&e.ID, &e.Name, &e.CreatedAt); err != nil {
			writeError(w, 500, err.Error())
			return
		}
		exercises = append(exercises, e)
	}
	writeJSON(w, 200, exercises)
}

func createExercise(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, 400, "неверный формат запроса")
		return
	}
	if input.Name == "" {
		writeError(w, 400, "название не может быть пустым")
		return
	}

	var e Exercise
	err := db.QueryRow(
		`INSERT INTO exercises (name) VALUES ($1) RETURNING id, name, created_at`,
		input.Name,
	).Scan(&e.ID, &e.Name, &e.CreatedAt)
	if err != nil {
		writeError(w, 500, err.Error())
		return
	}
	writeJSON(w, 201, e)
}

func deleteExercise(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		writeError(w, 400, "неверный id")
		return
	}
	_, err = db.Exec(`DELETE FROM exercises WHERE id = $1`, id)
	if err != nil {
		writeError(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, map[string]bool{"ok": true})
}

// ---------- обработчики: тренировки ----------

func createWorkout(w http.ResponseWriter, r *http.Request) {
	var wk Workout
	err := db.QueryRow(
		`INSERT INTO workouts (started_at) VALUES (now()) RETURNING id, started_at, ended_at`,
	).Scan(&wk.ID, &wk.StartedAt, &wk.EndedAt)
	if err != nil {
		writeError(w, 500, err.Error())
		return
	}
	writeJSON(w, 201, wk)
}

func finishWorkout(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		writeError(w, 400, "неверный id")
		return
	}
	var wk Workout
	err = db.QueryRow(
		`UPDATE workouts SET ended_at = now() WHERE id = $1 RETURNING id, started_at, ended_at`,
		id,
	).Scan(&wk.ID, &wk.StartedAt, &wk.EndedAt)
	if err != nil {
		writeError(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, wk)
}

func listWorkouts(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query(`SELECT id, started_at, ended_at FROM workouts ORDER BY started_at DESC`)
	if err != nil {
		writeError(w, 500, err.Error())
		return
	}
	defer rows.Close()

	workouts := []Workout{}
	for rows.Next() {
		var wk Workout
		if err := rows.Scan(&wk.ID, &wk.StartedAt, &wk.EndedAt); err != nil {
			writeError(w, 500, err.Error())
			return
		}
		workouts = append(workouts, wk)
	}
	writeJSON(w, 200, workouts)
}

// ---------- обработчики: подходы ----------

func addSet(w http.ResponseWriter, r *http.Request) {
	workoutID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		writeError(w, 400, "неверный id тренировки")
		return
	}
	var input struct {
		ExerciseID int     `json:"exercise_id"`
		Weight     float64 `json:"weight"`
		Reps       int     `json:"reps"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, 400, "неверный формат запроса")
		return
	}

	var s WorkoutSet
	err = db.QueryRow(
		`INSERT INTO workout_sets (workout_id, exercise_id, weight, reps)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id, workout_id, exercise_id, weight, reps`,
		workoutID, input.ExerciseID, input.Weight, input.Reps,
	).Scan(&s.ID, &s.WorkoutID, &s.ExerciseID, &s.Weight, &s.Reps)
	if err != nil {
		writeError(w, 500, err.Error())
		return
	}
	writeJSON(w, 201, s)
}

func listSets(w http.ResponseWriter, r *http.Request) {
	workoutID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		writeError(w, 400, "неверный id тренировки")
		return
	}
	rows, err := db.Query(
		`SELECT id, workout_id, exercise_id, weight, reps FROM workout_sets WHERE workout_id = $1 ORDER BY id`,
		workoutID,
	)
	if err != nil {
		writeError(w, 500, err.Error())
		return
	}
	defer rows.Close()

	sets := []WorkoutSet{}
	for rows.Next() {
		var s WorkoutSet
		if err := rows.Scan(&s.ID, &s.WorkoutID, &s.ExerciseID, &s.Weight, &s.Reps); err != nil {
			writeError(w, 500, err.Error())
			return
		}
		sets = append(sets, s)
	}
	writeJSON(w, 200, sets)
}

// ---------- точка входа ----------

func main() {
	connStr := os.Getenv("DATABASE_URL")
	if connStr == "" {
		// подставь свой пароль, который задавал через ALTER USER
		connStr = "postgres://postgres:bpan53_RUS@localhost:5432/postgres?sslmode=disable"
	}

	var err error
	db, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal("не удалось открыть соединение с базой: ", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatal("не удалось подключиться к базе: ", err)
	}
	log.Println("подключение к базе установлено успешно")

	mux := http.NewServeMux()
	mux.HandleFunc("GET /exercises", listExercises)
	mux.HandleFunc("POST /exercises", createExercise)
	mux.HandleFunc("DELETE /exercises/{id}", deleteExercise)

	mux.HandleFunc("GET /workouts", listWorkouts)
	mux.HandleFunc("POST /workouts", createWorkout)
	mux.HandleFunc("POST /workouts/{id}/finish", finishWorkout)

	mux.HandleFunc("GET /workouts/{id}/sets", listSets)
	mux.HandleFunc("POST /workouts/{id}/sets", addSet)

	handler := enableCORS(mux)

	log.Println("сервер запущен на http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", handler))
}
