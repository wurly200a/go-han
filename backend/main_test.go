package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func setupRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.Default()
	r.GET("/health", healthCheck)
	r.GET("/api/meals", getMeals)
	r.PUT("/api/meals/:user_id", updateMeal)
	return r
}

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

    assert.Equal(t, http.StatusOK, w.Code)
    assert.Contains(t, w.Body.String(), "healthy")
}

func TestGetMeals(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer mockDB.Close()

	db = mockDB

	rows := sqlmock.NewRows([]string{"user_id", "date", "lunch", "dinner"}).
		AddRow(1, "2024-02-04T00:00:00Z", true, true).
		AddRow(2, "2024-02-04T00:00:00Z", false, true)

	mock.ExpectQuery("SELECT user_id, date, lunch, dinner FROM meals").
		WillReturnRows(rows)

	r := setupRouter()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/meals", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
    expectedBody := `[{"user_id":1,"date":"2024-02-04T00:00:00Z","lunch":true,"dinner":true},{"user_id":2,"date":"2024-02-04T00:00:00Z","lunch":false,"dinner":true}]`
    assert.JSONEq(t, expectedBody, w.Body.String())
}


func TestUpdateMeal(t *testing.T) {
    gin.SetMode(gin.TestMode)

    r := gin.New()
    r.Use(gin.Logger())

    dbMock, mock, err := sqlmock.New()
    if err != nil {
        t.Fatalf("failed to open sqlmock database: %s", err)
    }
    defer dbMock.Close()
    db = dbMock

    reqBody := `{"user_id": 1, "date": "2024-02-09", "lunch": true, "dinner": false}`
    req, err := http.NewRequest(http.MethodPut, "/api/meals/1", bytes.NewBufferString(reqBody))
    if err != nil {
        t.Fatalf("failed to create request: %s", err)
    }
    req.Header.Set("Content-Type", "application/json")

    mock.ExpectExec(`INSERT INTO meals \(user_id, date, lunch, dinner\) VALUES \(\$1, \$2, \$3, \$4\) ON CONFLICT \(user_id, date\) DO UPDATE SET lunch = EXCLUDED\.lunch, dinner = EXCLUDED\.dinner`).
        WithArgs(1, "2024-02-09", true, false).
        WillReturnResult(sqlmock.NewResult(1, 1))

    w := httptest.NewRecorder()
    c, _ := gin.CreateTestContext(w)
    c.Request = req

    r.PUT("/api/meals/1", updateMeal)
    r.ServeHTTP(w, req)

    assert.Equal(t, http.StatusOK, w.Code)
    expectedBody := `{"message":"Meal updated"}`
    assert.JSONEq(t, expectedBody, w.Body.String())

    if err := mock.ExpectationsWereMet(); err != nil {
        t.Errorf("unfulfilled expectations: %s", err)
    }
}
