//go:build integration

package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
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
		"docker.io/postgres:17-alpine",
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

// TestBulkUpdateMealsIntegration verifies that bulkUpdateMeals correctly
// inserts new meal records and upserts existing ones against a real PostgreSQL instance.
//
// Step 1: insert new meals for John on 2025-02-16 (lunch=3, dinner=1)
//   → getMeals should reflect the inserted values
//
// Step 2: overwrite lunch only with a different value (lunch=1)
//   → getMeals should show the updated lunch; dinner unchanged
func TestBulkUpdateMealsIntegration(t *testing.T) {
	cleanup := startPostgres(t)
	defer cleanup()

	_, err := db.Exec(`
		INSERT INTO users (id, name) VALUES (1, 'John');
		INSERT INTO meal_periods (id) VALUES (1), (2);
		INSERT INTO meal_options (id) VALUES (1), (2), (3);
		INSERT INTO user_defaults (user_id, day_of_week, lunch, dinner) VALUES
			(1, 0, 2, 2); -- Sun default: Home/Home
	`)
	require.NoError(t, err)

	r := setupRouter()

	bulkUpdate := func(updates []MealUpdate) {
		t.Helper()
		body, _ := json.Marshal(updates)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("PUT", "/api/meals/bulk-update", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	}

	getMeals := func(query string) string {
		t.Helper()
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/meals?"+query, nil)
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
		return w.Body.String()
	}

	// Step 1: insert new meals
	bulkUpdate([]MealUpdate{{UserID: 1, Date: "2025-02-16", Lunch: 3, Dinner: 1}})
	assert.JSONEq(t, `{
		"2025-02-16": [
			{"user_id":1,"user_name":"John","lunch":3,"dinner":1,"defaultLunch":2,"defaultDinner":2}
		]
	}`, getMeals("date=2025-02-16&days=1"))

	// Step 2: upsert — overwrite lunch only; dinner should remain 1
	bulkUpdate([]MealUpdate{{UserID: 1, Date: "2025-02-16", Lunch: 1}})
	assert.JSONEq(t, `{
		"2025-02-16": [
			{"user_id":1,"user_name":"John","lunch":1,"dinner":1,"defaultLunch":2,"defaultDinner":2}
		]
	}`, getMeals("date=2025-02-16&days=1"))
}

// TestGetMealsWeekdayDefaultsIntegration verifies that EXTRACT(DOW FROM d.date)
// correctly maps each day of the week (0=Sun … 6=Sat) to its user_defaults entry.
// Each day is given a distinct (lunch, dinner) pair so a wrong DOW mapping is
// immediately visible in the assertion.
//
// Date range: 2025-02-16 (Sun) … 2025-02-22 (Sat) — one full week, no meal records.
func TestGetMealsWeekdayDefaultsIntegration(t *testing.T) {
	cleanup := startPostgres(t)
	defer cleanup()

	_, err := db.Exec(`
		INSERT INTO users (id, name) VALUES (1, 'Test');
		INSERT INTO meal_periods (id) VALUES (1), (2);
		INSERT INTO meal_options (id) VALUES (1), (2), (3);
		INSERT INTO user_defaults (user_id, day_of_week, lunch, dinner) VALUES
			(1, 0, 1, 1),
			(1, 1, 2, 1),
			(1, 2, 3, 1),
			(1, 3, 1, 2),
			(1, 4, 2, 2),
			(1, 5, 3, 2),
			(1, 6, 1, 3);
	`)
	require.NoError(t, err)

	r := setupRouter()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/meals?date=2025-02-16&days=7", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.JSONEq(t, `{
		"2025-02-16": [{"user_id":1,"user_name":"Test","lunch":0,"dinner":0,"defaultLunch":1,"defaultDinner":1}],
		"2025-02-17": [{"user_id":1,"user_name":"Test","lunch":0,"dinner":0,"defaultLunch":2,"defaultDinner":1}],
		"2025-02-18": [{"user_id":1,"user_name":"Test","lunch":0,"dinner":0,"defaultLunch":3,"defaultDinner":1}],
		"2025-02-19": [{"user_id":1,"user_name":"Test","lunch":0,"dinner":0,"defaultLunch":1,"defaultDinner":2}],
		"2025-02-20": [{"user_id":1,"user_name":"Test","lunch":0,"dinner":0,"defaultLunch":2,"defaultDinner":2}],
		"2025-02-21": [{"user_id":1,"user_name":"Test","lunch":0,"dinner":0,"defaultLunch":3,"defaultDinner":2}],
		"2025-02-22": [{"user_id":1,"user_name":"Test","lunch":0,"dinner":0,"defaultLunch":1,"defaultDinner":3}]
	}`, w.Body.String())
}

// TestGetUsersIntegration verifies that GET /api/users returns all users with
// correct is_cook and is_eater values.
func TestGetUsersIntegration(t *testing.T) {
	cleanup := startPostgres(t)
	defer cleanup()

	_, err := db.Exec(`
		INSERT INTO users (id, name, is_cook, is_eater) VALUES
			(1, 'Mother', true,  false),
			(2, 'Father', true,  true),
			(3, 'Taro',   false, true);
	`)
	require.NoError(t, err)

	r := setupRouter()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/users", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.JSONEq(t, `[
		{"id":1,"name":"Mother","is_cook":true,"is_eater":false},
		{"id":2,"name":"Father","is_cook":true,"is_eater":true},
		{"id":3,"name":"Taro","is_cook":false,"is_eater":true}
	]`, w.Body.String())
}

// TestUpdateUserRolesIntegration verifies that PUT /api/users/:user_id/roles
// correctly updates both flags and that a subsequent GET /api/users reflects the change.
func TestUpdateUserRolesIntegration(t *testing.T) {
	cleanup := startPostgres(t)
	defer cleanup()

	_, err := db.Exec(`INSERT INTO users (id, name) VALUES (1, 'Taro');`)
	require.NoError(t, err)

	r := setupRouter()

	// Taro starts as is_cook=false, is_eater=true (defaults).
	// Promote Taro to cook+eater.
	body := bytes.NewBufferString(`{"is_cook":true,"is_eater":true}`)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/api/users/1/roles", body)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("GET", "/api/users", nil)
	r.ServeHTTP(w2, req2)
	assert.Equal(t, http.StatusOK, w2.Code)
	assert.JSONEq(t, `[{"id":1,"name":"Taro","is_cook":true,"is_eater":true}]`, w2.Body.String())
}

// TestGetMealsEaterFilterIntegration verifies that users with is_eater=false
// do not appear in the GET /api/meals response.
func TestGetMealsEaterFilterIntegration(t *testing.T) {
	cleanup := startPostgres(t)
	defer cleanup()

	_, err := db.Exec(`
		INSERT INTO users (id, name, is_cook, is_eater) VALUES
			(1, 'Mother', true,  false),
			(2, 'Father', true,  true);
		INSERT INTO meal_periods (id) VALUES (1), (2);
		INSERT INTO meal_options (id) VALUES (1), (2), (3);
	`)
	require.NoError(t, err)

	r := setupRouter()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/meals?date=2025-02-16&days=1", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Mother (is_eater=false) must not appear; only Father should.
	body := w.Body.String()
	assert.NotContains(t, body, "Mother")
	assert.Contains(t, body, "Father")
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
