# Atlus

---

A 30-days-of-code like event at SJEC, aiming to help students to learn and build upon their critical thinking and coding skills.

_some highlights_

- github auth is mandatory
- every user will get their own unique inputs to solve

---

### Setup

```
cd atlus
go mod download
go run main.go
```

### .env

```
GITHUB_CLIENT_ID=
GITHUB_CLIENT_SECRET=
DATABASE_URL=postgres://<user>:<password>@<hostname>:<port>/<dbname>
HOSTNAME=localhost
PORT=8000
```

- github clientID and clientSecret can be found [here](https://github.com/settings/applications/new)

### Flags

```
--dev # truncate the database tables
```
