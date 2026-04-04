-- Migration: add is_cook / is_eater role columns to users table.
-- Safe to run on a live database: ADD COLUMN with DEFAULT does not rewrite the
-- table in PostgreSQL 11+ and is effectively instantaneous.
ALTER TABLE users
    ADD COLUMN IF NOT EXISTS is_cook  BOOL NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS is_eater BOOL NOT NULL DEFAULT true;
