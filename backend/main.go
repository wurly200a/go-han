package main

import (
    "log"
	"encoding/json"
    "database/sql"
    "net/http"
    "github.com/gin-gonic/gin"
    _ "github.com/lib/pq"
    "os"
    "time"
    "strconv"
)

type Meal struct {
    UserID int    `json:"user_id"`
    UserName string    `json:"user_name"`
    Date   string `json:"date"`
    Lunch  string `json:"lunch"`
    Dinner string `json:"dinner"`
}

type MealUpdate struct {
    UserID int    `json:"user_id"`
    UserName string    `json:"user_name"`
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
    dateParam := c.Query("date")
    daysParam := c.Query("days")

    startDate, err := time.Parse("2006-01-02", dateParam)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid date format. Use YYYY-MM-DD."})
        return
    }

    days, err := strconv.Atoi(daysParam)
    if err != nil || days < 1 {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid days parameter. Must be a positive integer."})
        return
    }

    endDate := startDate.AddDate(0, 0, days-1).Format("2006-01-02")

    usersQuery := "SELECT id, name FROM users"
    log.Printf("Executing SQL: %s", usersQuery)
    userRows, err := db.Query(usersQuery)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    defer userRows.Close()

    users := make(map[int]string)
    for userRows.Next() {
        var id int
        var name string
        if err := userRows.Scan(&id, &name); err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
            return
        }
        users[id] = name
    }
    log.Printf("users: %v", users)

    mealsQuery := `
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
        WHERE m.date BETWEEN $1 AND $2
        ORDER BY m.date, m.user_id`

	log.Printf("Executing SQL: %s with params: %s, %s", mealsQuery, startDate.Format("2006-01-02"), endDate)

    rows, err := db.Query(mealsQuery, startDate, endDate)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    defer rows.Close()

    for rows.Next() {
        var userID int
        var date string
        var lunch, dinner string
    
        err := rows.Scan(&userID, &date, &lunch, &dinner)
        if err != nil {
            log.Fatalf("Error scanning row: %v", err)
        }
    
        log.Printf("User ID: %d, Date: %s, Lunch: %s, Dinner: %s", userID, date, lunch, dinner)
    }
    
    if err := rows.Err(); err != nil {
        log.Fatalf("Error iterating over rows: %v", err)
    }

    mealsMap := make(map[string]map[int]Meal)
    for i := 0; i < days; i++ {
        dateStr := startDate.AddDate(0, 0, i).Format("2006-01-02")
        mealsMap[dateStr] = make(map[int]Meal)
        for id, name := range users {
            mealsMap[dateStr][id] = Meal{UserID: id, UserName: name, Lunch: "なし", Dinner: "なし"}
        }
    }

    for rows.Next() {
        var meal Meal
        var dateStr string
        if err := rows.Scan(&meal.UserID, &dateStr, &meal.Lunch, &meal.Dinner); err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
            return
        }
        mealsMap[dateStr][meal.UserID] = meal
    }

    finalMeals := make(map[string][]Meal)
    for date, usersMap := range mealsMap {
        for _, meal := range usersMap {
            finalMeals[date] = append(finalMeals[date], meal)
        }
    }
    responseJSON, _ := json.Marshal(finalMeals)
    log.Printf("Result: %s", responseJSON)

    c.JSON(http.StatusOK, finalMeals)
}

//func updateMeal(c *gin.Context) {
//    var m MealUpdate
//    if err := c.ShouldBindJSON(&m); err != nil {
//        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
//        return
//    }
//
//    _, err := db.Exec("INSERT INTO meals (user_id, date, lunch, dinner) VALUES ($1, $2, $3, $4) ON CONFLICT (user_id, date) DO UPDATE SET lunch = EXCLUDED.lunch, dinner = EXCLUDED.dinner", m.UserID, m.Date, m.Lunch, m.Dinner)
//    if err != nil {
//        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
//        return
//    }
//    c.JSON(http.StatusOK, gin.H{"message": "Meal updated"})
//}

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
//    r.PUT("/api/meals/:user_id", updateMeal)

    r.Run(":8080")
}
