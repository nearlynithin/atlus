CREATE TABLE
    IF NOT EXISTS users (
        github_id INT PRIMARY KEY,
        input_id INT GENERATED ALWAYS AS IDENTITY UNIQUE,
        username TEXT NOT NULL,
        github_url TEXT,
        avatar TEXT,
        email TEXT,
        current_level INTEGER DEFAULT 1,
        streak INTEGER DEFAULT 0,
        created_at TIMESTAMP DEFAULT NOW ()
    );

CREATE TABLE
    IF NOT EXISTS sessions (
        session_id TEXT PRIMARY KEY,
        github_id INT UNIQUE REFERENCES users (github_id),
        input_id INT REFERENCES users (input_id),
        created_at TIMESTAMP DEFAULT NOW (),
        expires_at TIMESTAMP NOT NULL,
        last_activity TIMESTAMP DEFAULT NOW ()
    );

CREATE TABLE
    IF NOT EXISTS levels (
        level_id INT PRIMARY KEY,
        name TEXT NOT NULL,
        release_time TIMESTAMP NOT NULL
    );

CREATE TABLE
	IF NOT EXISTS submissions (
		github_id INT REFERENCES users(github_id),
		level_id INT REFERENCES levels(level_id),
		last_submission TIMESTAMP NOT NULL,
		time_taken INTERVAL ,
		attempts INT DEFAULT 0,
		passed BOOLEAN DEFAULT FALSE,
		PRIMARY KEY (github_id, level_id)
	);
