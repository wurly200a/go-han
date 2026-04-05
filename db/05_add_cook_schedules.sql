-- Migration: add cook_default_schedules and cook_schedules tables.
-- Both tables are new; no existing tables are modified.
CREATE TABLE IF NOT EXISTS cook_default_schedules (
    day_of_week  INT NOT NULL,
    meal_period  INT NOT NULL,
    cook_user_id INT REFERENCES users(id) ON DELETE SET NULL,
    PRIMARY KEY (day_of_week, meal_period)
);

CREATE TABLE IF NOT EXISTS cook_schedules (
    date         DATE NOT NULL,
    meal_period  INT  NOT NULL,
    cook_user_id INT REFERENCES users(id) ON DELETE SET NULL,
    PRIMARY KEY (date, meal_period)
);
