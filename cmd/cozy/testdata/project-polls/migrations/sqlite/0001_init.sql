CREATE TABLE IF NOT EXISTS questions (
  id INTEGER PRIMARY KEY,
  question_text TEXT NOT NULL,
  pub_date TEXT NOT NULL,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  version INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS choices (
  id INTEGER PRIMARY KEY,
  question_id INTEGER NOT NULL,
  choice_text TEXT NOT NULL,
  votes INTEGER NOT NULL DEFAULT 0,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  version INTEGER NOT NULL
);
