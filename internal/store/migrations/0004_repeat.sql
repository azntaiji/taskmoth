-- Schema version 4 (decision G15.1 in PLAN.md): recurring tasks. repeat holds
-- the rule in task.Rule's encoded form ("w/1/mon,wed"); NULL means the task
-- doesn't repeat. series_id links the occurrences of one series: the ID of
-- the task the rule was first set on.

ALTER TABLE tasks ADD COLUMN repeat TEXT CHECK (repeat IS NULL OR repeat <> '');
ALTER TABLE tasks ADD COLUMN series_id TEXT;

CREATE INDEX tasks_series ON tasks(series_id) WHERE series_id IS NOT NULL;
