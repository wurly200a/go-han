package main

import (
    "log"
	"encoding/json"
    "database/sql"
    "net/http"
    "github.com/gin-gonic/gin"
    _ "github.com/lib/pq"
    "os"
)

type Meal struct {
    UserID int    `json:"user_id"`
    Date   string `json:"date"`
    Lunch  string `json:"lunch"`
    Dinner string `json:"dinner"`
}

type MealUpdate struct {
    UserID int    `json:"user_id"`
    Date   string `json:"date"`
    Lunch  int    `json:"lunch"`
    Dinner int    `json:"dinner"`
}

var db *sql.DB

func healthCheck(c *gin.Context) {
    if err := db.Ping(); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"status": "unhealthy", "error": err.Error()})
        return
    }
    c.JSON(http.StatusOK, gin.H{"status":"healthy"})
}

func getMeals(c *gin.Context) {
//    query := "SELECT user_id, date, lunch, dinner FROM meals"
	query := `
    SELECT 
        m.user_id, 
        m.date, 
        lunch_trans.name AS lunch, 
        dinner_trans.name AS dinner
    FROM meals m
    LEFT JOIN meal_option_translations lunch_trans 
        ON m.lunch = lunch_trans.meal_option_id AND lunch_trans.language_code = 'ja'
    LEFT JOIN meal_option_translations dinner_trans 
        ON m.dinner = dinner_trans.meal_option_id AND dinner_trans.language_code = 'ja'
    `
    log.Printf("Executing SQL: %s", query)

    rows, err := db.Query(query)
    if err != nil {
        log.Printf("SQL Error: %v", err)
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    defer rows.Close()

    var meals []Meal
    for rows.Next() {
        var m Meal
        if err := rows.Scan(&m.UserID, &m.Date, &m.Lunch, &m.Dinner); err != nil {
            log.Printf("Row Scan Error: %v", err)
            c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
            return
        }
        meals = append(meals, m)
    }

    responseJSON, _ := json.Marshal(meals)
    log.Printf("SQL Result: %s", responseJSON)

    c.JSON(http.StatusOK, meals)
}

func updateMeal(c *gin.Context) {
    var m MealUpdate
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
    r.GET("/health", healthCheck)
    r.GET("/api/meals", getMeals)
    r.PUT("/api/meals/:user_id", updateMeal)

    r.Run(":8080")
}
