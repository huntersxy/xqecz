ALTER TABLE ai_generation_tasks
  ADD COLUMN attempts INT NOT NULL DEFAULT 0;
