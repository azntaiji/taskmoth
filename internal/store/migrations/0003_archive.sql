-- Schema version 3 (decision G4.4 in PLAN.md): the trash becomes the
-- archive, and projects can be archived too. Archiving a project stamps the
-- project and its tasks with the same archived_at.

ALTER TABLE tasks RENAME COLUMN deleted_at TO archived_at;
ALTER TABLE projects ADD COLUMN archived_at TEXT;
