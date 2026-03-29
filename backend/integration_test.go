//go:build integration

package main

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

// startPostgres starts a PostgreSQL container with the app schema applied,
// sets the global db, and returns a cleanup function.
func startPostgres(t *testing.T) func() {
	t.Helper()
	ctx := context.Background()

	// WithOccurrence(2): postgres logs "ready" once before init scripts run,
	// then restarts and logs it again after scripts complete. Wait for the
	// second occurrence to ensure init scripts have finished.
	pgc, err := tcpostgres.Run(ctx,
		"docker.io/postgres:16-alpine",
		tcpostgres.WithDatabase("testdb"),
		tcpostgres.WithUsername("testuser"),
		tcpostgres.WithPassword("testpass"),
		tcpostgres.WithInitScripts("../db/02_init_schema.sql"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second),
		),
	)
	require.NoError(t, err)

	connStr, err := pgc.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	testDB, err := sql.Open("postgres", connStr)
	require.NoError(t, err)
	require.NoError(t, testDB.Ping())

	db = testDB

	return func() {
		testDB.Close()
		if err := testcontainers.TerminateContainer(pgc); err != nil {
			t.Logf("failed to terminate container: %v", err)
		}
	}
}

// seedGetMeals inserts the test fixtures used by TestGetMealsIntegration.
// The data intentionally mirrors the unit test scenario so both tests
// validate the same expected JSON.
//
// Users: John(1), Paul(2)  |  Dates: 2025-02-16(Sun), 2025-02-17(Mon)
//
// user_defaults:
//   John  Sun: lunch=2(家)   dinner=2(家)
//   John  Mon: lunch=1(なし) dinner=2(家)
//   Paul  Sun: lunch=2(家)   dinner=2(家)
//   Paul  Mon: lunch=2(家)   dinner=2(家)
//
// meals 2025-02-16:
//   John lunch=1, dinner=1  (both override defaults → highlight)
//   Paul lunch=1, dinner=1
//
// meals 2025-02-17:
//   John lunch=3(弁当), dinner=1
//   Paul dinner=2 only → lunch column returns 0 (not set)
func seedGetMeals(t *testing.T) {
	t.Helper()
	_, err := db.Exec(`
		INSERT INTO users (id, name) VALUES (1, 'John'), (2, 'Paul');
		INSERT INTO meal_periods (id) VALUES (1), (2);
		INSERT INTO meal_options (id) VALUES (1), (2), (3);
		INSERT INTO user_defaults (user_id, day_of_week, lunch, dinner) VALUES
			(1, 0, 2, 2),
			(1, 1, 1, 2),
			(2, 0, 2, 2),
			(2, 1, 2, 2);
		INSERT INTO meals (user_id, date, meal_period, meal_option) VALUES
			(1, '2025-02-16', 1, 1),
			(1, '2025-02-16', 2, 1),
			(2, '2025-02-16', 1, 1),
			(2, '2025-02-16', 2, 1),
			(1, '2025-02-17', 1, 3),
			(1, '2025-02-17', 2, 1),
			(2, '2025-02-17', 2, 2);
	`)
	require.NoError(t, err)
}

// TestGetMealsIntegration tests getMeals against a real PostgreSQL instance.
// It validates the SQL behaviors that unit tests with sqlmock cannot cover:
//   - generate_series expands the date range correctly
//   - CASE WHEN pivots meal_period rows into lunch/dinner columns
//   - LEFT JOIN with user_defaults provides per-weekday fallback values
//   - COALESCE returns 0 for lunch when no meal record exists (Paul Mon)
//   - TO_CHAR formats the date key as YYYY-MM-DD
func TestGetMealsIntegration(t *testing.T) {
	cleanup := startPostgres(t)
	defer cleanup()
	seedGetMeals(t)

	r := setupRouter()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/meals?date=2025-02-16&days=2", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	expected := `{
		"2025-02-16": [
			{"user_id": 1, "user_name": "John", "lunch": 1, "dinner": 1, "defaultLunch": 2, "defaultDinner": 2},
			{"user_id": 2, "user_name": "Paul", "lunch": 1, "dinner": 1, "defaultLunch": 2, "defaultDinner": 2}
		],
		"2025-02-17": [
			{"user_id": 1, "user_name": "John", "lunch": 3, "dinner": 1, "defaultLunch": 1, "defaultDinner": 2},
			{"user_id": 2, "user_name": "Paul", "lunch": 0, "dinner": 2,  "defaultLunch": 2, "defaultDinner": 2}
		]
	}`
	assert.JSONEq(t, expected, w.Body.String())
}

// TestGetMealsNoDefaultsIntegration verifies COALESCE fallback when a user has
// no entry in user_defaults: defaultLunch and defaultDinner must be 1 (なし).
func TestGetMealsNoDefaultsIntegration(t *testing.T) {
	cleanup := startPostgres(t)
	defer cleanup()

	_, err := db.Exec(`
		INSERT INTO users (id, name) VALUES (1, 'Solo');
		INSERT INTO meal_periods (id) VALUES (1), (2);
		INSERT INTO meal_options (id) VALUES (1), (2), (3);
	`)
	require.NoError(t, err)

	r := setupRouter()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/meals?date=2025-02-16&days=1", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	expected := `{
		"2025-02-16": [
			{"user_id": 1, "user_name": "Solo", "lunch": 0, "dinner": 0, "defaultLunch": 1, "defaultDinner": 1}
		]
	}`
	assert.JSONEq(t, expected, w.Body.String())
}
