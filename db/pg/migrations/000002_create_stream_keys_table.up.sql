CREATE TABLE stream_keys (
    id serial,
    stream_key varchar(512),
    user_id integer NOT NULL,
    active bool DEFAULT TRUE,
    created_at timestamptz NOT NULL DEFAULT NOW(),

    PRIMARY KEY (id),
    FOREIGN KEY (user_id) REFERENCES users(id),
    UNIQUE (stream_key, user_id)
);
