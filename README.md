# go-han
Schedule management of meal (gohan) at home

## Configuration

Copy a template file and customise it for your environment

```
cp .env.example .env
db_init.sh
```

## Run

```
docker compose up --build
```

## Remove

```
docker compose down -v
```

## Test (Backend)

```
docker compose exec backend go test ./...
```

## Debug(Database)

```
docker exec -it my_postgres psql -U myuser -d mydatabase
```

```
SELECT id, name FROM users;
SELECT * FROM meals WHERE date BETWEEN '2024-02-04' AND '2024-02-09';
```
