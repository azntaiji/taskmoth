-- Schema version 2 (decision G3.3b in PLAN.md): user settings such as each
-- view's sort order. Keys are dotted names, e.g. "sort.today".

CREATE TABLE settings (
  key    TEXT PRIMARY KEY NOT NULL,
  value  TEXT NOT NULL
) STRICT;
