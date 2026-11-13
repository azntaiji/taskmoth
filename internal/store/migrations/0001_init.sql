-- Schema version 1 (decision G2.4 in PLAN.md).
-- IDs are UUIDv7 text. Dates are local YYYY-MM-DD. Timestamps are UTC RFC 3339.

CREATE TABLE projects (
  id          TEXT PRIMARY KEY NOT NULL,
  name        TEXT NOT NULL UNIQUE
              CHECK (name <> '' AND name NOT GLOB '*[^a-z0-9_-]*'),
  created_at  TEXT NOT NULL,
  updated_at  TEXT NOT NULL
) STRICT;

CREATE TABLE tasks (
  id                TEXT PRIMARY KEY NOT NULL,
  name              TEXT NOT NULL
                    CHECK (trim(name) <> '' AND instr(name, char(10)) = 0 AND instr(name, char(13)) = 0),
  description       TEXT NOT NULL DEFAULT '',
  project_id        TEXT REFERENCES projects(id) ON DELETE RESTRICT,
  priority          INTEGER NOT NULL DEFAULT 4 CHECK (priority BETWEEN 1 AND 4),
  start_date        TEXT CHECK (start_date IS NULL OR start_date GLOB '[0-9][0-9][0-9][0-9]-[0-9][0-9]-[0-9][0-9]'),
  due_date          TEXT CHECK (due_date   IS NULL OR due_date   GLOB '[0-9][0-9][0-9][0-9]-[0-9][0-9]-[0-9][0-9]'),
  duration_minutes  INTEGER CHECK (duration_minutes IS NULL OR duration_minutes > 0),
  created_at        TEXT NOT NULL,
  updated_at        TEXT NOT NULL,
  completed_at      TEXT,
  deleted_at        TEXT
) STRICT;

CREATE INDEX tasks_project ON tasks(project_id);
CREATE INDEX tasks_due     ON tasks(due_date)   WHERE deleted_at IS NULL;
CREATE INDEX tasks_start   ON tasks(start_date) WHERE deleted_at IS NULL;
