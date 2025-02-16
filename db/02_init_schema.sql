-- Users table to store user information
CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    age INT
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
    lunch INT REFERENCES meal_options(id) ON DELETE SET NULL,  -- Lunch option (referencing meal_options)
    dinner INT REFERENCES meal_options(id) ON DELETE SET NULL,  -- Dinner option (referencing meal_options)
    UNIQUE (user_id, date)  -- Ensures one meal record per user per day
);

CREATE TABLE IF NOT EXISTS user_defaults (
    user_id INT REFERENCES users(id) ON DELETE CASCADE,
    day_of_week INT NOT NULL,  -- 0: 日, 1: 月, …, 6: 土
    lunch INT,
    dinner INT,
    PRIMARY KEY (user_id, day_of_week)
);
