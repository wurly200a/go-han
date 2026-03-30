# go-han
Schedule management of meal (gohan) at home

## Configuration

Copy a template file and customise it for your environment

```
cp .env.example .env
db_init.sh
```

### Notification configuration

Set the following environment variable in your .env file to enable notifications for last-minute changes (within 24 hours):

SLACK_WEBHOOK_URL=your_slack_incoming_webhook_url

## Run

```
docker compose up --build
```

## Remove

```
docker compose down -v
```

## Deploy

Deploy target host is managed via git config to avoid storing the server address in the repository.

### Initial setup (one-time)

```
git config deploy.host <SERVER_IP>
```

### Run deploy

```
make deploy
```

This will:
1. `git fetch` + `git rebase origin/main` on the server
2. Rebuild and restart `backend` and `frontend` containers
3. Leave `postgres` running untouched

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
