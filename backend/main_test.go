package main

import (
//	"fmt"
	"bytes"
	"encoding/json"
//	"log"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// setupRouter configures the Gin router with all required endpoints.
func setupRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.Default()
	r.GET("/health", healthCheck)
	r.GET("/api/users", getUsers)
	r.PUT("/api/users/:user_id/roles", updateUserRoles)
	r.GET("/api/meals", getMeals)
	r.PUT("/api/meals/bulk-update", bulkUpdateMeals)
	r.GET("/api/user-defaults/:user_id", getUserDefaults)
	r.PUT("/api/user-defaults/:user_id", updateUserDefaults)
	r.GET("/api/cook-schedules", getCookSchedules)
	r.PUT("/api/cook-schedules", bulkUpdateCookSchedules)
	r.DELETE("/api/cook-schedules", deleteCookSchedules)
	r.GET("/api/cook-default-schedules", getCookDefaultSchedules)
	r.PUT("/api/cook-default-schedules", updateCookDefaultSchedules)
	return r
}

// TestHealthCheck verifies that the /health endpoint returns a healthy status.
func TestHealthCheck(t *testing.T) {
	mockDB, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
	assert.NoError(t, err)
	defer mockDB.Close()

	db = mockDB
	mock.ExpectPing().WillReturnError(nil)

	r := setupRouter()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/health", nil)
	r.ServeHTTP(w, req)
	t.Log("\n", func() string { var b bytes.Buffer; json.Indent(&b, w.Body.Bytes(), "", "  "); return b.String() }())

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "healthy")
}

// TestGetMeals verifies the /api/meals endpoint.
// The refactored getMeals uses a single query that pivots meal_period into
// lunch/dinner columns and joins user defaults via generate_series.
func TestGetMeals(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer mockDB.Close()
	db = mockDB

	// Single combined query: users × date range, with meals pivoted and defaults joined.
	// Rows represent the final per-user-per-date result already merged by SQL.
	rows := sqlmock.NewRows([]string{"user_id", "user_name", "date", "lunch", "dinner", "default_lunch", "default_dinner"}).
		// 2025-02-16 (Sunday): both users have explicit meal records overriding defaults
		AddRow(1, "John", "2025-02-16", 1, 1, 2, 2). // John: lunch=None, dinner=None; default Sun=Home/Home
		AddRow(2, "Paul", "2025-02-16", 1, 1, 2, 2). // Paul: lunch=None, dinner=None; default Sun=Home/Home
		// 2025-02-17 (Monday): partial records; Paul has no lunch record (returns 0)
		AddRow(1, "John", "2025-02-17", 3, 1, 1, 2). // John: lunch=Bento, dinner=None; default Mon=None/Home
		AddRow(2, "Paul", "2025-02-17", 0, 2, 2, 2)  // Paul: lunch=0(not set), dinner=Home; default Mon=Home/Home

	mock.ExpectQuery(regexp.QuoteMeta(getMealsQuery)).
		WithArgs("2025-02-16", "2025-02-17").
		WillReturnRows(rows)

	r := setupRouter()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/meals?date=2025-02-16&days=2", nil)
	r.ServeHTTP(w, req)
	t.Log("\n", func() string { var b bytes.Buffer; json.Indent(&b, w.Body.Bytes(), "", "  "); return b.String() }())
	assert.Equal(t, http.StatusOK, w.Code)

	expectedBody := `{
      "2025-02-16": [
        {"user_id": 1, "user_name": "John", "lunch": 1, "dinner": 1, "defaultLunch": 2, "defaultDinner": 2},
        {"user_id": 2, "user_name": "Paul", "lunch": 1, "dinner": 1, "defaultLunch": 2, "defaultDinner": 2}
      ],
      "2025-02-17": [
        {"user_id": 1, "user_name": "John", "lunch": 3, "dinner": 1, "defaultLunch": 1, "defaultDinner": 2},
        {"user_id": 2, "user_name": "Paul", "lunch": 0, "dinner": 2, "defaultLunch": 2, "defaultDinner": 2}
      ]
    }`
	assert.JSONEq(t, expectedBody, w.Body.String())
}

// TestBulkUpdateMeals verifies the /api/meals/bulk-update endpoint.
// This test simulates the flow of a transaction: Begin, Prepare, Exec, and Commit.
func TestBulkUpdateMeals(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer mockDB.Close()
	db = mockDB

	// --- Query: users ---
	rowsUsers := sqlmock.NewRows([]string{"id", "name"}).
		AddRow(1, "John").
		AddRow(2, "Paul")
	queryUsers := "SELECT id, name FROM users"
	t.Log(regexp.QuoteMeta(queryUsers))
	t.Log(rowsUsers)
	mock.ExpectQuery(regexp.QuoteMeta(queryUsers)).WillReturnRows(rowsUsers)

	mock.ExpectBegin()
//	prep := mock.ExpectPrepare(regexp.QuoteMeta("INSERT INTO meals (user_id, date, lunch, dinner) VALUES ($1, $2, $3, $4) ON CONFLICT (user_id, date) DO UPDATE SET lunch = EXCLUDED.lunch, dinner = EXCLUDED.dinner"))
	prep := mock.ExpectPrepare(regexp.QuoteMeta("INSERT INTO meals (user_id, date, meal_period, meal_option) VALUES ($1, $2, $3, $4) ON CONFLICT (user_id, date, meal_period) DO UPDATE SET meal_option = EXCLUDED.meal_option"))
	// Simulate two update records.
	prep.ExpectExec().WithArgs(1, "2024-02-04", 1, 3).WillReturnResult(sqlmock.NewResult(1, 1))
	prep.ExpectExec().WithArgs(1, "2024-02-04", 2, 2).WillReturnResult(sqlmock.NewResult(2, 1))
	prep.ExpectExec().WithArgs(2, "2024-02-04", 1, 1).WillReturnResult(sqlmock.NewResult(3, 1))
	prep.ExpectExec().WithArgs(2, "2024-02-04", 2, 3).WillReturnResult(sqlmock.NewResult(4, 1))
	prep.ExpectExec().WithArgs(1, "2024-02-05", 1, 2).WillReturnResult(sqlmock.NewResult(1, 1))
	prep.ExpectExec().WithArgs(2, "2024-02-05", 2, 1).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	updates := []MealUpdate{
		{UserID: 1, Date: "2024-02-04", Lunch: 3, Dinner: 2},
		{UserID: 2, Date: "2024-02-04", Lunch: 1, Dinner: 3},
		{UserID: 1, Date: "2024-02-05", Lunch: 2},
		{UserID: 2, Date: "2024-02-05", Dinner: 1},
	}
	payload, _ := json.Marshal(updates)
	r := setupRouter()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/api/meals/bulk-update", bytes.NewBuffer(payload))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	t.Log("\n", func() string { var b bytes.Buffer; json.Indent(&b, w.Body.Bytes(), "", "  "); return b.String() }())
	assert.Equal(t, http.StatusOK, w.Code)
	expectedResp := `{"message":"Meals updated"}`
	assert.JSONEq(t, expectedResp, w.Body.String())
}

// TestGetUserDefaults verifies the GET /api/user-defaults/:user_id endpoint.
func TestGetUserDefaults(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer mockDB.Close()
	db = mockDB

	// Example scenario: for user_id = "4", return 7 default records.
	rows := sqlmock.NewRows([]string{"day_of_week", "lunch", "dinner"}).
		AddRow(0, 3, 1).
		AddRow(1, 1, 2).
		AddRow(2, 3, 1).
		AddRow(3, 3, 1).
		AddRow(4, 1, 3).
		AddRow(5, 3, 1).
		AddRow(6, 3, 1)
	query := "SELECT day_of_week, lunch, dinner FROM user_defaults WHERE user_id = $1 ORDER BY day_of_week"
	t.Log(regexp.QuoteMeta(query))
	t.Log(rows)
	mock.ExpectQuery(regexp.QuoteMeta(query)).
		WithArgs("4").
		WillReturnRows(rows)

	r := setupRouter()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/user-defaults/4", nil)
	r.ServeHTTP(w, req)
	t.Log("\n", func() string { var b bytes.Buffer; json.Indent(&b, w.Body.Bytes(), "", "  "); return b.String() }())
	assert.Equal(t, http.StatusOK, w.Code)

	// Update expected JSON to include the user_id field.
	expectedJSON := `[
        {"day_of_week":0, "lunch":3, "dinner":1, "user_id":4},
        {"day_of_week":1, "lunch":1, "dinner":2, "user_id":4},
        {"day_of_week":2, "lunch":3, "dinner":1, "user_id":4},
        {"day_of_week":3, "lunch":3, "dinner":1, "user_id":4},
        {"day_of_week":4, "lunch":1, "dinner":3, "user_id":4},
        {"day_of_week":5, "lunch":3, "dinner":1, "user_id":4},
        {"day_of_week":6, "lunch":3, "dinner":1, "user_id":4}
    ]`
	assert.JSONEq(t, expectedJSON, w.Body.String())
}

// TestGetUsers verifies the GET /api/users endpoint.
func TestGetUsers(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer mockDB.Close()
	db = mockDB

	rows := sqlmock.NewRows([]string{"id", "name", "is_cook", "is_eater"}).
		AddRow(1, "Mother", true, false).
		AddRow(2, "Father", true, true).
		AddRow(3, "Taro", false, true)
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, name, is_cook, is_eater FROM users ORDER BY id")).
		WillReturnRows(rows)

	r := setupRouter()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/users", nil)
	r.ServeHTTP(w, req)
	t.Log("\n", func() string { var b bytes.Buffer; json.Indent(&b, w.Body.Bytes(), "", "  "); return b.String() }())
	assert.Equal(t, http.StatusOK, w.Code)

	expected := `[
		{"id":1,"name":"Mother","is_cook":true,"is_eater":false},
		{"id":2,"name":"Father","is_cook":true,"is_eater":true},
		{"id":3,"name":"Taro","is_cook":false,"is_eater":true}
	]`
	assert.JSONEq(t, expected, w.Body.String())
}

// TestUpdateUserRoles verifies the PUT /api/users/:user_id/roles endpoint.
func TestUpdateUserRoles(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer mockDB.Close()
	db = mockDB

	mock.ExpectExec(regexp.QuoteMeta("UPDATE users SET is_cook = $1, is_eater = $2 WHERE id = $3")).
		WithArgs(true, false, "1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	payload := `{"is_cook":true,"is_eater":false}`
	r := setupRouter()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/api/users/1/roles", bytes.NewBufferString(payload))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	t.Log("\n", func() string { var b bytes.Buffer; json.Indent(&b, w.Body.Bytes(), "", "  "); return b.String() }())
	assert.Equal(t, http.StatusOK, w.Code)
	assert.JSONEq(t, `{"message":"User roles updated"}`, w.Body.String())
}

// TestUpdateUserRolesNotFound verifies 404 when user_id does not exist.
func TestUpdateUserRolesNotFound(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer mockDB.Close()
	db = mockDB

	mock.ExpectExec(regexp.QuoteMeta("UPDATE users SET is_cook = $1, is_eater = $2 WHERE id = $3")).
		WithArgs(false, true, "99").
		WillReturnResult(sqlmock.NewResult(0, 0))

	payload := `{"is_cook":false,"is_eater":true}`
	r := setupRouter()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/api/users/99/roles", bytes.NewBufferString(payload))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

// TestUpdateUserDefaults verifies the PUT /api/user-defaults/:user_id endpoint.
func TestUpdateUserDefaults(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer mockDB.Close()
	db = mockDB

	mock.ExpectBegin()
	prep := mock.ExpectPrepare(regexp.QuoteMeta("INSERT INTO user_defaults (user_id, day_of_week, lunch, dinner) VALUES ($1, $2, $3, $4) ON CONFLICT (user_id, day_of_week) DO UPDATE SET lunch = EXCLUDED.lunch, dinner = EXCLUDED.dinner"))
	// Simulate updating two default records.
	prep.ExpectExec().WithArgs("4", 0, 3, 1).WillReturnResult(sqlmock.NewResult(1, 1))
	prep.ExpectExec().WithArgs("4", 1, 1, 2).WillReturnResult(sqlmock.NewResult(2, 1))
	mock.ExpectCommit()

	// Payload: two default settings.
	defaults := []struct {
		DayOfWeek int `json:"day_of_week"`
		Lunch     int `json:"lunch"`
		Dinner    int `json:"dinner"`
		UserID    int `json:"user_id"`
	}{
		{DayOfWeek: 0, Lunch: 3, Dinner: 1, UserID: 4},
		{DayOfWeek: 1, Lunch: 1, Dinner: 2, UserID: 4},
	}
	payload, _ := json.Marshal(defaults)
	r := setupRouter()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/api/user-defaults/4", bytes.NewBuffer(payload))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	t.Log("\n", func() string { var b bytes.Buffer; json.Indent(&b, w.Body.Bytes(), "", "  "); return b.String() }())
	assert.Equal(t, http.StatusOK, w.Code)
	expectedResp := `{"message":"User defaults updated"}`
	assert.JSONEq(t, expectedResp, w.Body.String())
}

// TestGetCookSchedules verifies GET /api/cook-schedules.
// Rows with a valid cook_user_id produce a CookAssignment object;
// rows with NULL produce nil (各自) in the JSON response.
func TestGetCookSchedules(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer mockDB.Close()
	db = mockDB

	rows := sqlmock.NewRows([]string{"date", "meal_period", "cook_user_id", "cook_user_name"}).
		AddRow("2026-04-06", 1, 5, "Mother").
		AddRow("2026-04-06", 2, nil, nil).
		AddRow("2026-04-07", 1, nil, nil).
		AddRow("2026-04-07", 2, 2, "Father")

	mock.ExpectQuery(regexp.QuoteMeta(getCookSchedulesQuery)).
		WithArgs("2026-04-06", "2026-04-07").
		WillReturnRows(rows)

	r := setupRouter()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/cook-schedules?date=2026-04-06&days=2", nil)
	r.ServeHTTP(w, req)
	t.Log("\n", func() string { var b bytes.Buffer; json.Indent(&b, w.Body.Bytes(), "", "  "); return b.String() }())
	assert.Equal(t, http.StatusOK, w.Code)

	expected := `{
		"2026-04-06": {"lunch": {"cook_user_id": 5, "cook_user_name": "Mother"}, "dinner": null},
		"2026-04-07": {"lunch": null, "dinner": {"cook_user_id": 2, "cook_user_name": "Father"}}
	}`
	assert.JSONEq(t, expected, w.Body.String())
}

// TestBulkUpdateCookSchedules verifies PUT /api/cook-schedules.
func TestBulkUpdateCookSchedules(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer mockDB.Close()
	db = mockDB

	mock.ExpectBegin()
	prep := mock.ExpectPrepare(regexp.QuoteMeta(bulkUpdateCookSchedulesStmt))
	prep.ExpectExec().WithArgs("2026-04-06", 1, sqlmock.AnyArg()).WillReturnResult(sqlmock.NewResult(1, 1))
	prep.ExpectExec().WithArgs("2026-04-06", 2, sqlmock.AnyArg()).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	cookID := 5
	updates := []CookScheduleUpdate{
		{Date: "2026-04-06", MealPeriod: 1, CookUserID: &cookID},
		{Date: "2026-04-06", MealPeriod: 2, CookUserID: nil},
	}
	payload, _ := json.Marshal(updates)
	r := setupRouter()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/api/cook-schedules", bytes.NewBuffer(payload))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	t.Log("\n", func() string { var b bytes.Buffer; json.Indent(&b, w.Body.Bytes(), "", "  "); return b.String() }())
	assert.Equal(t, http.StatusOK, w.Code)
	assert.JSONEq(t, `{"message":"Cook schedules updated"}`, w.Body.String())
}

// TestDeleteCookSchedules verifies DELETE /api/cook-schedules.
func TestDeleteCookSchedules(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer mockDB.Close()
	db = mockDB

	mock.ExpectBegin()
	prep := mock.ExpectPrepare(regexp.QuoteMeta(deleteCookSchedulesStmt))
	prep.ExpectExec().WithArgs("2026-04-06", 1).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	entries := []CookScheduleDelete{{Date: "2026-04-06", MealPeriod: 1}}
	payload, _ := json.Marshal(entries)
	r := setupRouter()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/api/cook-schedules", bytes.NewBuffer(payload))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	t.Log("\n", func() string { var b bytes.Buffer; json.Indent(&b, w.Body.Bytes(), "", "  "); return b.String() }())
	assert.Equal(t, http.StatusOK, w.Code)
	assert.JSONEq(t, `{"message":"Cook schedules deleted"}`, w.Body.String())
}

// TestGetCookDefaultSchedules verifies GET /api/cook-default-schedules.
func TestGetCookDefaultSchedules(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer mockDB.Close()
	db = mockDB

	rows := sqlmock.NewRows([]string{"day_of_week", "meal_period", "cook_user_id", "cook_user_name"}).
		AddRow(1, 1, 5, "Mother").
		AddRow(1, 2, 5, "Mother").
		AddRow(3, 1, nil, nil)

	mock.ExpectQuery(regexp.QuoteMeta(getCookDefaultSchedulesQuery)).WillReturnRows(rows)

	r := setupRouter()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/cook-default-schedules", nil)
	r.ServeHTTP(w, req)
	t.Log("\n", func() string { var b bytes.Buffer; json.Indent(&b, w.Body.Bytes(), "", "  "); return b.String() }())
	assert.Equal(t, http.StatusOK, w.Code)

	expected := `[
		{"day_of_week":1,"meal_period":1,"cook_user_id":5,"cook_user_name":"Mother"},
		{"day_of_week":1,"meal_period":2,"cook_user_id":5,"cook_user_name":"Mother"},
		{"day_of_week":3,"meal_period":1,"cook_user_id":null,"cook_user_name":null}
	]`
	assert.JSONEq(t, expected, w.Body.String())
}

// TestUpdateCookDefaultSchedules verifies PUT /api/cook-default-schedules.
func TestUpdateCookDefaultSchedules(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer mockDB.Close()
	db = mockDB

	mock.ExpectBegin()
	prep := mock.ExpectPrepare(regexp.QuoteMeta(updateCookDefaultSchedulesStmt))
	prep.ExpectExec().WithArgs(1, 1, sqlmock.AnyArg()).WillReturnResult(sqlmock.NewResult(1, 1))
	prep.ExpectExec().WithArgs(1, 2, sqlmock.AnyArg()).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	cookID := 5
	entries := []CookDefaultScheduleUpdate{
		{DayOfWeek: 1, MealPeriod: 1, CookUserID: &cookID},
		{DayOfWeek: 1, MealPeriod: 2, CookUserID: nil},
	}
	payload, _ := json.Marshal(entries)
	r := setupRouter()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/api/cook-default-schedules", bytes.NewBuffer(payload))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	t.Log("\n", func() string { var b bytes.Buffer; json.Indent(&b, w.Body.Bytes(), "", "  "); return b.String() }())
	assert.Equal(t, http.StatusOK, w.Code)
	assert.JSONEq(t, `{"message":"Cook default schedules updated"}`, w.Body.String())
}
