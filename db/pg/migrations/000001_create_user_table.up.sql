CREATE TABLE users (
    id serial,
    username varchar(50) NOT NULL,
    password varchar(512) NOT NULL,
    created_at timestamptz NOT NULL DEFAULT NOW(),

    PRIMARY KEY (id),
    UNIQUE (username)
);
