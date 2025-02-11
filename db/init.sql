GRANT ALL PRIVILEGES ON DATABASE mydatabase TO myuser;

\c mydatabase;

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

-- Meal option translations table for multilingual support
-- Each meal option can have multiple translations based on language code
CREATE TABLE IF NOT EXISTS meal_option_translations (
    meal_option_id INT REFERENCES meal_options(id) ON DELETE CASCADE,
    language_code TEXT NOT NULL,  -- Example: 'ja' for Japanese, 'en' for English
    name TEXT NOT NULL,           -- Translated meal option name
    PRIMARY KEY (meal_option_id, language_code)
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

-- Insert default meal options
INSERT INTO meal_options (id) VALUES (1), (2), (3);

-- Insert translations for meal options
INSERT INTO meal_option_translations (meal_option_id, language_code, name) VALUES
(1, 'en', 'None'),  (1, 'ja', 'なし'),
(2, 'en', 'Home'),  (2, 'ja', '家'),
(3, 'en', 'Obento'),(3, 'ja', '弁当');

-- Insert sample users
INSERT INTO users (name, age) VALUES ('Saburo', 16), ('Jiro', 19), ('Taro', 21), ('Father', 47);

INSERT INTO user_defaults (user_id, day_of_week, lunch, dinner) VALUES
(1, 0, 2, 2),
(1, 1, 2, 2),
(1, 2, 2, 2),
(1, 3, 2, 2),
(1, 4, 2, 2),
(1, 5, 2, 2),
(1, 6, 2, 2),
(2, 0, 2, 2),
(2, 1, 2, 2),
(2, 2, 2, 2),
(2, 3, 2, 2),
(2, 4, 2, 2),
(2, 5, 2, 2),
(2, 6, 2, 2),
(3, 0, 2, 2),
(3, 1, 2, 2),
(3, 2, 2, 2),
(3, 3, 2, 2),
(3, 4, 2, 2),
(3, 5, 2, 2),
(3, 6, 2, 2),
(4, 0, 2, 2),
(4, 1, 2, 2),
(4, 2, 2, 2),
(4, 3, 2, 2),
(4, 4, 2, 2),
(4, 5, 2, 2),
(4, 6, 2, 2);

-- Insert sample meal records (using meal_options IDs)
INSERT INTO meals (user_id, date, lunch, dinner) VALUES
(1, '2024-02-04', 3, 2),  -- 'Obento' for lunch, 'Home' for dinner
(1, '2024-02-05', 3, 1),  -- 'Obento' for lunch, 'None' for dinner
(2, '2024-02-04', 1, 3),  -- 'None' for lunch, 'Obento' for dinner
(3, '2024-02-04', 3, 1),  -- 'Obento' for lunch, 'None' for dinner
(4, '2024-02-04', 3, 1);  -- 'Obento' for lunch, 'None' for dinner
