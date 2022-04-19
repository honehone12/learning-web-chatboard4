DROP TABLE replies;
DROP TABLE topics;
DROP TABLE users;

CREATE TABLE users (
  id         SERIAL PRIMARY KEY,
  uu_id      VARCHAR(255) NOT NULL UNIQUE,
  name       VARCHAR(255) NOT NULL UNIQUE,
  email      VARCHAR(255) NOT NULL UNIQUE,
  password   TEXT NOT NULL,
  salt       VARCHAR(255) NOT NULL,
  token      TEXT,
  created_at TIMESTAMP NOT NULL   
);

CREATE TABLE topics (
  id          SERIAL PRIMARY KEY,
  uu_id       VARCHAR(255) NOT NULL UNIQUE,
  topic       TEXT,
  num_replies SERIAL,
  owner       VARCHAR(255),
  user_id     SERIAL REFERENCES users(id),
  last_update TIMESTAMP NOT NULL,
  created_at  TIMESTAMP NOT NULL       
);

CREATE TABLE replies (
  id          SERIAL PRIMARY KEY,
  uu_id       VARCHAR(255) NOT NULL UNIQUE,
  body        TEXT,
  contributor VARCHAR(255),
  user_id     SERIAL REFERENCES users(id),
  topic_id   SERIAL REFERENCES topics(id),
  created_at  TIMESTAMP NOT NULL  
);
