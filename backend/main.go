package main

import (
	"database/sql"
//	"encoding/json"
//	"log"
	"net/http"
	"os"
	"sort"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
)

var db *sql.DB

// Meal represents meal information for a user on a specific date.
type Meal struct {
	UserID        int    `json:"user_id"`
	UserName      string `json:"user_name"`
	Lunch         int    `json:"lunch"`
	Dinner        int    `json:"dinner"`
	DefaultLunch  int    `json:"defaultLunch"`
	DefaultDinner int    `json:"defaultDinner"`
}

// MealUpdate represents an update for a meal record.
type MealUpdate struct {
	UserID   int    `json:"user_id"`
	UserName string `json:"user_name"`
	Date     string `json:"date"`
	Lunch    int    `json:"lunch"`
	Dinner   int    `json:"dinner"`
}

// UserDefault represents the default meal settings for a user for a given day of week.
type UserDefault struct {
	UserID    int `json:"user_id"`
	DayOfWeek int `json:"day_of_week"` // 0: Sunday, ... 6: Saturday
	Lunch     int `json:"lunch"`
	Dinner    int `json:"dinner"`
}

// Mapping from meal option id to Japanese text.
//var mealOptionText = map[int]string{
//	1: "なし",
//	2: "家",
//	3: "弁当",
//}

func healthCheck(c *gin.Context) {
	if err := db.Ping(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "unhealthy", "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "healthy"})
}

// getMeals retrieves meal information for a range of dates.
// For each user and each date, if there is no meal record, the user's default for that day-of-week is used.
// The returned JSON includes defaultLunch and defaultDinner fields.
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

	// Retrieve user information.
	usersQuery := "SELECT id, name FROM users"
//	log.Printf("Executing SQL: %s", usersQuery)
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
//	log.Printf("users: %v", users)

	// Retrieve user defaults.
	defaultsQuery := "SELECT user_id, day_of_week, lunch, dinner FROM user_defaults"
	defRows, err := db.Query(defaultsQuery)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer defRows.Close()

	// Map: userDefaults[user_id][day_of_week] = UserDefault
	userDefaults := make(map[int]map[int]UserDefault)
	for defRows.Next() {
		var ud UserDefault
		if err := defRows.Scan(&ud.UserID, &ud.DayOfWeek, &ud.Lunch, &ud.Dinner); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if userDefaults[ud.UserID] == nil {
			userDefaults[ud.UserID] = make(map[int]UserDefault)
		}
		userDefaults[ud.UserID][ud.DayOfWeek] = ud
	}

	// Retrieve meal records.
	mealsQuery := `
        SELECT 
            m.user_id, 
            m.date, 
            m.meal_period,
            m.meal_option
        FROM meals m
        WHERE m.date BETWEEN $1 AND $2
        ORDER BY m.date, m.user_id`
//	log.Printf("Executing SQL: %s with params: %s, %s", mealsQuery, startDate.Format("2006-01-02"), endDate)

	rows, err := db.Query(mealsQuery, startDate.Format("2006-01-02"), endDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	type mealRow struct {
		userID int
		date   time.Time
		meal_period  int
		meal_option int
	}
	var mealRows []mealRow
	for rows.Next() {
		var mr mealRow
		var dateVal interface{}
		if err := rows.Scan(&mr.userID, &dateVal, &mr.meal_period, &mr.meal_option); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		switch d := dateVal.(type) {
		case time.Time:
			mr.date = d
		case string:
			mr.date, err = time.Parse("2006-01-02", d)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "unsupported date type"})
			return
		}
//		log.Printf("User ID: %d, Date: %s, Lunch: %s, Dinner: %s", mr.userID, mr.date.Format("2006-01-02"), mr.lunch, mr.dinner)
		mealRows = append(mealRows, mr)
	}

	// Build mapping of actual meal records.
	type mealRow2 struct {
		userID int
		date   time.Time
		lunch  int
		dinner int
	}

	mealsData := make(map[string]map[int]mealRow2)

    for _, mr := range mealRows {
        dateStr := mr.date.Format("2006-01-02")
        if mealsData[dateStr] == nil {
            mealsData[dateStr] = make(map[int]mealRow2)
        }
        // Retrieve current value (or zero value if not set)
        current, ok := mealsData[dateStr][mr.userID]
        if !ok {
            current = mealRow2{userID: mr.userID, date: mr.date}
        }
        if mr.meal_period == 1 {
            current.lunch = mr.meal_option
        } else if mr.meal_period == 2 {
            current.dinner = mr.meal_option
        }
        // Write back to the map
        mealsData[dateStr][mr.userID] = current
    }

	// Build final result.
	finalMeals := make(map[string][]Meal)
	for i := 0; i < days; i++ {
		currentDate := startDate.AddDate(0, 0, i)
		dateStr := currentDate.Format("2006-01-02")
		weekday := int(currentDate.Weekday()) // 0: Sunday ... 6: Saturday
		for userID, userName := range users {
			// Get default from userDefaults.
			defaultLunch := 1
			defaultDinner := 1
			if ud, ok := userDefaults[userID][weekday]; ok {
				defaultLunch = ud.Lunch
				defaultDinner = ud.Dinner
			}
			m := Meal{
				UserID:        userID,
				UserName:      userName,
				Lunch:         0,
				Dinner:        0,
				DefaultLunch:  defaultLunch,
				DefaultDinner: defaultDinner,
			}
			// Override if an actual meal record exists.
			if rec, ok := mealsData[dateStr][userID]; ok {
				m.Lunch = rec.lunch
				m.Dinner = rec.dinner
			}
			finalMeals[dateStr] = append(finalMeals[dateStr], m)
		}
		sort.Slice(finalMeals[dateStr], func(i, j int) bool {
			return finalMeals[dateStr][i].UserID < finalMeals[dateStr][j].UserID
		})
	}
//	responseJSON, _ := json.Marshal(finalMeals)
//	log.Printf("Result: %s", responseJSON)

	c.JSON(http.StatusOK, finalMeals)
}

// bulkUpdateMeals performs a bulk update/insertion of meal records.
func bulkUpdateMeals(c *gin.Context) {
	var updates []MealUpdate
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	tx, err := db.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
//	stmt, err := tx.Prepare("INSERT INTO meals (user_id, date, lunch, dinner) VALUES ($1, $2, $3, $4) ON CONFLICT (user_id, date) DO UPDATE SET lunch = EXCLUDED.lunch, dinner = EXCLUDED.dinner")
	stmt, err := tx.Prepare("INSERT INTO meals (user_id, date, meal_period, meal_option) VALUES ($1, $2, $3, $4) ON CONFLICT (user_id, date, meal_period) DO UPDATE SET meal_option = EXCLUDED.meal_option")
	if err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer stmt.Close()
	for _, m := range updates {
		if _, err := stmt.Exec(m.UserID, m.Date, 1, m.Lunch ); err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		if _, err := stmt.Exec(m.UserID, m.Date, 2, m.Dinner); err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}
	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Meals updated"})
}

// getUserDefaults returns the default meal settings for a specific user.
func getUserDefaults(c *gin.Context) {
	userID := c.Param("user_id")
	rows, err := db.Query("SELECT day_of_week, lunch, dinner FROM user_defaults WHERE user_id = $1 ORDER BY day_of_week", userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	var defaults []UserDefault
	for rows.Next() {
		var ud UserDefault
		ud.UserID, _ = strconv.Atoi(userID)
		if err := rows.Scan(&ud.DayOfWeek, &ud.Lunch, &ud.Dinner); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defaults = append(defaults, ud)
	}
	c.JSON(http.StatusOK, defaults)
}

// updateUserDefaults updates the default meal settings for a specific user.
func updateUserDefaults(c *gin.Context) {
	userID := c.Param("user_id")
	var defaults []UserDefault
	if err := c.ShouldBindJSON(&defaults); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	tx, err := db.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	stmt, err := tx.Prepare("INSERT INTO user_defaults (user_id, day_of_week, lunch, dinner) VALUES ($1, $2, $3, $4) ON CONFLICT (user_id, day_of_week) DO UPDATE SET lunch = EXCLUDED.lunch, dinner = EXCLUDED.dinner")
	if err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer stmt.Close()
	for _, ud := range defaults {
		if _, err := stmt.Exec(userID, ud.DayOfWeek, ud.Lunch, ud.Dinner); err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}
	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "User defaults updated"})
}

func main() {
	var err error
	db, err = sql.Open("postgres", os.Getenv("DATABASE_URL"))
	if err != nil {
		panic(err)
	}
	defer db.Close()

	r := gin.Default()
	r.GET("/api/health", healthCheck)
	r.GET("/api/meals", getMeals)
	r.PUT("/api/meals/bulk-update", bulkUpdateMeals)
	r.GET("/api/user-defaults/:user_id", getUserDefaults)
	r.PUT("/api/user-defaults/:user_id", updateUserDefaults)
	r.Run(":8080")
}
