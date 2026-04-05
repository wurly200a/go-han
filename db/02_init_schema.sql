-- Users table to store user information
CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    is_cook  BOOL NOT NULL DEFAULT false,
    is_eater BOOL NOT NULL DEFAULT true
);

CREATE TABLE IF NOT EXISTS meal_periods (
    id SERIAL PRIMARY KEY
);

-- Meal options table (Master data)
-- This stores unique meal options (e.g., None, Home, Obento)
CREATE TABLE IF NOT EXISTS meal_options (
    id SERIAL PRIMARY KEY
);

-- Meals table to store users' meal choices
CREATE TABLE IF NOT EXISTS meals (
    id SERIAL PRIMARY KEY,
    user_id INT REFERENCES users(id) ON DELETE CASCADE,  -- Foreign key to users table
    date DATE NOT NULL,  -- The date of the meal
    meal_period INT REFERENCES meal_periods(id) ON DELETE SET NULL,  -- 0: lunch, 1: dinner
    meal_option INT REFERENCES meal_options(id) ON DELETE SET NULL,  -- Meal option (referencing meal_options)
    UNIQUE (user_id, date, meal_period)
);

CREATE TABLE IF NOT EXISTS user_defaults (
    user_id INT REFERENCES users(id) ON DELETE CASCADE,
    day_of_week INT NOT NULL,  -- 0: Sun, 1: Mon,... 6: Sat
    lunch INT,
    dinner INT,
    PRIMARY KEY (user_id, day_of_week)
);

-- Cook schedule: weekday-based default cook assignment per meal period.
-- cook_user_id NULL means 各自 (no designated cook).
CREATE TABLE IF NOT EXISTS cook_default_schedules (
    day_of_week  INT NOT NULL,
    meal_period  INT NOT NULL,
    cook_user_id INT REFERENCES users(id) ON DELETE SET NULL,
    PRIMARY KEY (day_of_week, meal_period)
);

-- Cook schedule: individual date override per meal period.
-- cook_user_id NULL means explicitly 各自 (overrides any weekday default).
CREATE TABLE IF NOT EXISTS cook_schedules (
    date         DATE NOT NULL,
    meal_period  INT  NOT NULL,
    cook_user_id INT REFERENCES users(id) ON DELETE SET NULL,
    PRIMARY KEY (date, meal_period)
);
