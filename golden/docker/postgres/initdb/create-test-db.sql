-- Runs once, when the pgdata volume is first created (see the
-- initdb mount in docker-compose.yml). `make test` migrates and runs
-- the backend suite against this database, never against `app`.
CREATE DATABASE app_test;
