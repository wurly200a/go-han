
GRANT ALL PRIVILEGES ON DATABASE mydatabase TO myuser;

\c mydatabase;

CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    age INT
);

CREATE TABLE IF NOT EXISTS meals (
    id SERIAL PRIMARY KEY,
    user_id INT REFERENCES users(id) ON DELETE CASCADE,
    date DATE NOT NULL,
    lunch BOOLEAN NOT NULL DEFAULT TRUE,
    dinner BOOLEAN NOT NULL DEFAULT TRUE,
    UNIQUE (user_id, date)
);

INSERT INTO users (name, age) VALUES ('Saburo', 16), ('Jiro', 19), ('Taro', 21), ('Father', 47);

INSERT INTO meals (user_id, date, lunch, dinner) VALUES
(1, '2024-02-04', true, true),
(2, '2024-02-04', false, true),
(3, '2024-02-04', true, false),
(4, '2024-02-04', true, false),
(1, '2024-02-05', true, true),
(2, '2024-02-05', true, false),
(3, '2024-02-05', false, true),
(4, '2024-02-05', false, true),
(1, '2024-02-06', true, true),
(2, '2024-02-06', true, true),
(3, '2024-02-06', false, false),
(4, '2024-02-06', false, false);
