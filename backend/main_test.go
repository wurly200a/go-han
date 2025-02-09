package main

import (
    "testing"
    "net/http"
    "net/http/httptest"
    "github.com/gin-gonic/gin"
    _ "github.com/lib/pq"
)

func TestGetUsers(t *testing.T) {
    r := gin.Default()
    r.GET("/users", func(c *gin.Context) {
        c.JSON(200, gin.H{"users": []map[string]interface{}{}})
    })

    req, _ := http.NewRequest("GET", "/users", nil)
    w := httptest.NewRecorder()
    r.ServeHTTP(w, req)

    if w.Code != 200 {
        t.Fatalf("Expected status 200 but got %d", w.Code)
    }
}
