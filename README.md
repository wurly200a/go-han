# go-han
Schedule management of meal (gohan) at home

## Build

```
docker compose up -d --build
```

## Run

```
docker compose up --build
```

## Test (Backend)

```
docker compose exec backend go test ./...
```

## Remove

```
docker compose down -v
```

## Debug

```
docker exec -it my_postgres psql -U myuser -d mydatabase
```

```
SELECT id, name FROM users;
SELECT * FROM meals WHERE date BETWEEN '2024-02-04' AND '2024-02-09';
```

```
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
WHERE m.date BETWEEN '2024-02-04' AND '2024-02-04'
ORDER BY m.date, m.user_id;
```


