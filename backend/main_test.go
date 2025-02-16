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
	r.GET("/api/meals", getMeals)
	r.PUT("/api/meals/bulk-update", bulkUpdateMeals)
	r.GET("/api/user-defaults/:user_id", getUserDefaults)
	r.PUT("/api/user-defaults/:user_id", updateUserDefaults)
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
// It simulates queries for:
// 1. User information,
// 2. User defaults, and
// 3. Actual meal records,
// and checks that the returned JSON includes the defaultLunch and defaultDinner fields.
func TestGetMeals(t *testing.T) {
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

	// --- Query: user_defaults ---
	rowsDefaults := sqlmock.NewRows([]string{"user_id", "day_of_week", "lunch", "dinner"}).
		AddRow(1, 0, 2, 2). // John, Sun, 'Home' for lunch, 'Home' for dinner
		AddRow(1, 1, 1, 2). // John, Mon, 'None' for lunch, 'Home' for dinner
		AddRow(1, 2, 1, 2). // John, Tue, 'None' for lunch, 'Home' for dinner
		AddRow(1, 3, 1, 2). // John, Wed, 'None' for lunch, 'Home' for dinner
		AddRow(1, 4, 1, 2). // John, Thu, 'None' for lunch, 'Home' for dinner
		AddRow(1, 5, 1, 2). // John, Fri, 'None' for lunch, 'Home' for dinner
		AddRow(1, 6, 2, 2). // John, Sat, 'Home' for lunch, 'Home' for dinner
		AddRow(2, 0, 2, 2). // Paul, Sun, 'Home' for lunch, 'Home' for dinner
		AddRow(2, 1, 2, 2). // Paul, Mon, 'Home' for lunch, 'Home' for dinner
		AddRow(2, 2, 2, 2). // Paul, Tue, 'Home' for lunch, 'Home' for dinner
		AddRow(2, 3, 2, 2). // Paul, Wed, 'Home' for lunch, 'Home' for dinner
		AddRow(2, 4, 2, 2). // Paul, Thu, 'Home' for lunch, 'Home' for dinner
		AddRow(2, 5, 2, 2). // Paul, Fri, 'Home' for lunch, 'Home' for dinner
		AddRow(2, 6, 2, 2)  // Paul, Sat, 'Home' for lunch, 'Home' for dinner
	// The current implementation of getMeals uses this query without a WHERE clause.
	queryDefaults := "SELECT user_id, day_of_week, lunch, dinner FROM user_defaults"
	t.Log(regexp.QuoteMeta(queryDefaults))
	t.Log(rowsDefaults)
	mock.ExpectQuery(regexp.QuoteMeta(queryDefaults)).WillReturnRows(rowsDefaults)

	// --- Query: meals ---
	// Actual meal records that override the defaults.
	rowsMeals := sqlmock.NewRows([]string{"user_id", "date", "meal_period", "meal_option"}).
		AddRow(1, "2025-02-16", 1, 1). // John, Sun, 'None'   for lunch
		AddRow(1, "2025-02-16", 2, 1). // John, Sun, 'None'   for dinner
		AddRow(2, "2025-02-16", 1, 1). // Paul, Sun, 'None'   for lunch
		AddRow(2, "2025-02-16", 2, 1). // Paul, Sun, 'None'   for dinner
		AddRow(1, "2025-02-17", 1, 3). // John, Mon, 'Obento' for lunch
		AddRow(1, "2025-02-17", 2, 1). // John, Mon, 'None'   for dinner
//		AddRow(2, "2025-02-17", 1, 1). // Paul, Mon, 'Home'   for lunch
		AddRow(2, "2025-02-17", 2, 2)  // Paul, Mon, 'Home'   for dinner
	queryMeals := `
        SELECT 
            m.user_id, 
            m.date, 
            m.meal_period,
            m.meal_option
        FROM meals m
        WHERE m.date BETWEEN $1 AND $2
        ORDER BY m.date, m.user_id`
	t.Log(regexp.QuoteMeta(queryMeals))
	t.Log(rowsMeals)
	mock.ExpectQuery(regexp.QuoteMeta(queryMeals)).
		WithArgs("2025-02-16", "2025-02-17"). // Sunday and Monday
		WillReturnRows(rowsMeals)

	r := setupRouter()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/meals?date=2025-02-16&days=2", nil)
	r.ServeHTTP(w, req)
	t.Log("\n", func() string { var b bytes.Buffer; json.Indent(&b, w.Body.Bytes(), "", "  "); return b.String() }())
	assert.Equal(t, http.StatusOK, w.Code)

	// Expected JSON (including defaultLunch and defaultDinner fields).
	expectedBody := `{
      "2025-02-16": [
        {
          "user_id": 1,
          "user_name": "John",
          "lunch": 1,
          "dinner": 1,
          "defaultLunch": 2,
          "defaultDinner": 2
        },
        {
          "user_id": 2,
          "user_name": "Paul",
          "lunch": 1,
          "dinner": 1,
          "defaultLunch": 2,
          "defaultDinner": 2
        }
      ],
      "2025-02-17": [
        {
          "user_id": 1,
          "user_name": "John",
          "lunch": 3,
          "dinner": 1,
          "defaultLunch": 1,
          "defaultDinner": 2
        },
        {
          "user_id": 2,
          "user_name": "Paul",
          "lunch": 0,
          "dinner": 2,
          "defaultLunch": 2,
          "defaultDinner": 2
        }
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

	mock.ExpectBegin()
//	prep := mock.ExpectPrepare(regexp.QuoteMeta("INSERT INTO meals (user_id, date, lunch, dinner) VALUES ($1, $2, $3, $4) ON CONFLICT (user_id, date) DO UPDATE SET lunch = EXCLUDED.lunch, dinner = EXCLUDED.dinner"))
	prep := mock.ExpectPrepare(regexp.QuoteMeta("INSERT INTO meals (user_id, date, meal_period, meal_option) VALUES ($1, $2, $3, $4) ON CONFLICT (user_id, date, meal_period) DO UPDATE SET meal_option = EXCLUDED.meal_option"))
	// Simulate two update records.
	prep.ExpectExec().WithArgs(1, "2024-02-04", 1, 3).WillReturnResult(sqlmock.NewResult(1, 1))
	prep.ExpectExec().WithArgs(1, "2024-02-04", 2, 2).WillReturnResult(sqlmock.NewResult(2, 1))
	prep.ExpectExec().WithArgs(2, "2024-02-04", 1, 1).WillReturnResult(sqlmock.NewResult(3, 1))
	prep.ExpectExec().WithArgs(2, "2024-02-04", 2, 3).WillReturnResult(sqlmock.NewResult(4, 1))
	mock.ExpectCommit()

	updates := []MealUpdate{
		{UserID: 1, Date: "2024-02-04", Lunch: 3, Dinner: 2},
		{UserID: 2, Date: "2024-02-04", Lunch: 1, Dinner: 3},
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
