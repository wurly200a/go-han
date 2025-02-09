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
