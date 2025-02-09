package main

import (
    "database/sql"
    "net/http"
    "github.com/gin-gonic/gin"
    _ "github.com/lib/pq"
    "os"
)

type Meal struct {
    UserID int    `json:"user_id"`
    Date   string `json:"date"`
    Lunch  bool   `json:"lunch"`
    Dinner bool   `json:"dinner"`
}

var db *sql.DB

func getMeals(c *gin.Context) {
    rows, err := db.Query("SELECT user_id, date, lunch, dinner FROM meals")
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    defer rows.Close()

    var meals []Meal
    for rows.Next() {
        var m Meal
        if err := rows.Scan(&m.UserID, &m.Date, &m.Lunch, &m.Dinner); err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
            return
        }
        meals = append(meals, m)
    }
    c.JSON(http.StatusOK, meals)
}

func updateMeal(c *gin.Context) {
    var m Meal
    if err := c.ShouldBindJSON(&m); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    _, err := db.Exec("INSERT INTO meals (user_id, date, lunch, dinner) VALUES ($1, $2, $3, $4) ON CONFLICT (user_id, date) DO UPDATE SET lunch = EXCLUDED.lunch, dinner = EXCLUDED.dinner", m.UserID, m.Date, m.Lunch, m.Dinner)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    c.JSON(http.StatusOK, gin.H{"message": "Meal updated"})
}

func main() {
    var err error
    db, err = sql.Open("postgres", os.Getenv("DATABASE_URL"))
    if err != nil {
        panic(err)
    }
    defer db.Close()

    r := gin.Default()
    r.GET("/api/meals", getMeals)
    r.PUT("/api/meals/:user_id", updateMeal)

    r.Run(":8080")
}
