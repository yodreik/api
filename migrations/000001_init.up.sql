CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE users
(
    id UUID DEFAULT uuid_generate_v4() NOT NULL UNIQUE,
    email VARCHAR(254) NOT NULL UNIQUE,
    name VARCHAR(50) NOT NULL,
    password_hash CHAR(64) NOT NULL,
    is_email_confirmed BOOLEAN DEFAULT false NOT NULL,
    created_at TIMESTAMP DEFAULT now() NOT NULL
);

CREATE TABLE requests
(
    id UUID DEFAULT uuid_generate_v4() NOT NULL UNIQUE,
    kind VARCHAR(20) NOT NULL,
    email VARCHAR(254) NOT NULL REFERENCES users(email) ON DELETE CASCADE,
    token VARCHAR(64) NOT NULL UNIQUE,
    is_used BOOLEAN DEFAULT false NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT now() NOT NULL
);

CREATE TABLE workouts
(
    id UUID DEFAULT uuid_generate_v4() NOT NULL UNIQUE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    date DATE NOT NULL,
    duration INTEGER NOT NULL,
    kind VARCHAR(50) NOT NULL,
    created_at TIMESTAMP DEFAULT now() NOT NULL
);
