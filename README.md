# dreik API

## Building

#### 1/ First things first, clone the repo

```console
$ git clone https://github.com/dreik/api.git && cd ./api
```

---

#### 2/ Set up your config files. Use [`./config/example.yml`](./config/example.yml) as a template

---

#### 3/ Set up `.env` file

```console
$ echo CONFIG_PATH=./config/local.yaml > .env
$ cat .env
CONFIG_PATH=./config/local.yaml
```

---

#### 4/ Rebuild swagger, coverage report and run tests

> [!IMPORTANT]  
> Make sure, you have installed: [`swaggo/swag`](https://github.com/swaggo/swag) for rebuilding swagger documentation

```console
$ make all
...
... swagger building logs 
... tests results and coverage report
...
```

---

#### 5/ Running

Inside docker compose

```console
$ docker build -t dreik-api:latest .
$ docker compose up
```

or outside the docker

```console
$ docker compose -f ./docker-compose.dev.yaml up -d
$ go build -o bin/api cmd/api/main.go
```
> [!NOTE]  
> For `local` and `dev` environments, there are available coverage report (on `/coverage` endpoint) and SwaggerUI (on `/docs/index.html`). For production environment, this routes are not configured, like CORS headers.

---

#### 6/ Install `migrate` tool

```console
$ go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
```

---

#### 7/ Apply database migrations

```console
$ migrate -path migrations -database "postgres://postgres:password@localhost:5432/postgres?sslmode=disable" up
```

---

#### 8/ Check it out

```console
$ curl -i http://localhost:6969/api/healthcheck
HTTP/1.1 200 OK
...
```

## Other notes

#### How to explore my database?

```console
$ docker exec -it dreik-api:latest psql -U username
```

Than you can use SQL queries

```sql
select * from users;
```

---

#### Drop database
```console
$ migrate -path migrations -database "postgres://postgres:password@localhost:5432/postgres?sslmode=disable" down
```
