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

### Schema

```
CREATE TABLE users (
    github_id INT PRIMARY KEY,
    input_id INT GENERATED ALWAYS AS IDENTITY UNIQUE,
    username TEXT NOT NULL,
    github_url TEXT,
    avatar TEXT,
    email TEXT,
    current_level INTEGER DEFAULT 1,
    streak INTEGER DEFAULT 0,
    last_submission TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE sessions (
    session_id TEXT PRIMARY KEY,
    github_id INT UNIQUE REFERENCES users(github_id),
    input_id INT REFERENCES users(input_id),
    created_at TIMESTAMP DEFAULT NOW(),
    expires_at TIMESTAMP NOT NULL,
    last_activity TIMESTAMP DEFAULT NOW()
);
```
