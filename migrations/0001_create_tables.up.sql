CREATE TABLE teams (
  id SERIAL PRIMARY KEY,
  name TEXT UNIQUE NOT NULL
);

CREATE TABLE users (
  id TEXT PRIMARY KEY, 
  name TEXT NOT NULL,
  is_active BOOLEAN NOT NULL DEFAULT true
);

CREATE TABLE team_members (
  team_id INT NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
  user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE, 
  PRIMARY KEY (team_id, user_id)
);

CREATE TYPE pr_status AS ENUM ('OPEN','MERGED');

CREATE TABLE prs (
  id TEXT PRIMARY KEY,  
  title TEXT NOT NULL,
  author_id TEXT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,  
  status pr_status NOT NULL DEFAULT 'OPEN',
  created_at TIMESTAMP WITH TIME ZONE DEFAULT now(),
  merged_at TIMESTAMP WITH TIME ZONE
);

CREATE TABLE pr_reviewers (
  pr_id TEXT NOT NULL REFERENCES prs(id) ON DELETE CASCADE,     
  user_id TEXT NOT NULL REFERENCES users(id) ON DELETE RESTRICT, 
  PRIMARY KEY (pr_id, user_id)
);

CREATE TABLE assignment_events (
  id SERIAL PRIMARY KEY,
  pr_id TEXT NOT NULL REFERENCES prs(id) ON DELETE CASCADE,      
  user_id TEXT NOT NULL REFERENCES users(id) ON DELETE RESTRICT, 
  event_time TIMESTAMP WITH TIME ZONE DEFAULT now()
);

CREATE INDEX idx_team_members_team ON team_members(team_id);
CREATE INDEX idx_users_active ON users(is_active);
CREATE INDEX idx_pr_reviewers_pr ON pr_reviewers(pr_id);
CREATE INDEX idx_prs_status ON prs(status);

