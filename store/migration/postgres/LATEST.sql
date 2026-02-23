-- system_setting
CREATE TABLE system_setting (
  name TEXT NOT NULL PRIMARY KEY,
  value TEXT NOT NULL,
  description TEXT NOT NULL
);

-- user
CREATE TABLE "user" (
  id SERIAL PRIMARY KEY,
  created_ts BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM NOW()),
  updated_ts BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM NOW()),
  row_status TEXT NOT NULL DEFAULT 'NORMAL',
  username TEXT NOT NULL UNIQUE,
  role TEXT NOT NULL DEFAULT 'USER',
  email TEXT NOT NULL DEFAULT '',
  nickname TEXT NOT NULL DEFAULT '',
  password_hash TEXT NOT NULL,
  avatar_url TEXT NOT NULL,
  description TEXT NOT NULL DEFAULT ''
);

-- user_setting
CREATE TABLE user_setting (
  user_id INTEGER NOT NULL,
  key TEXT NOT NULL,
  value TEXT NOT NULL,
  UNIQUE(user_id, key)
);

-- memo
CREATE TABLE memo (
  id SERIAL PRIMARY KEY,
  uid TEXT NOT NULL UNIQUE,
  creator_id INTEGER NOT NULL,
  created_ts BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM NOW()),
  updated_ts BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM NOW()),
  row_status TEXT NOT NULL DEFAULT 'NORMAL',
  content TEXT NOT NULL,
  visibility TEXT NOT NULL DEFAULT 'PRIVATE',
  pinned BOOLEAN NOT NULL DEFAULT FALSE,
  payload JSONB NOT NULL DEFAULT '{}'
);

-- memo_relation
CREATE TABLE memo_relation (
  memo_id INTEGER NOT NULL,
  related_memo_id INTEGER NOT NULL,
  type TEXT NOT NULL,
  UNIQUE(memo_id, related_memo_id, type)
);

-- attachment
CREATE TABLE attachment (
  id SERIAL PRIMARY KEY,
  uid TEXT NOT NULL UNIQUE,
  creator_id INTEGER NOT NULL,
  created_ts BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM NOW()),
  updated_ts BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM NOW()),
  filename TEXT NOT NULL,
  blob BYTEA,
  type TEXT NOT NULL DEFAULT '',
  size INTEGER NOT NULL DEFAULT 0,
  memo_id INTEGER DEFAULT NULL,
  storage_type TEXT NOT NULL DEFAULT '',
  reference TEXT NOT NULL DEFAULT '',
  payload TEXT NOT NULL DEFAULT '{}'
);

-- activity
CREATE TABLE activity (
  id SERIAL PRIMARY KEY,
  creator_id INTEGER NOT NULL,
  created_ts BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM NOW()),
  type TEXT NOT NULL DEFAULT '',
  level TEXT NOT NULL DEFAULT 'INFO',
  payload JSONB NOT NULL DEFAULT '{}'
);

-- idp
CREATE TABLE idp (
  id SERIAL PRIMARY KEY,
  name TEXT NOT NULL,
  type TEXT NOT NULL,
  identifier_filter TEXT NOT NULL DEFAULT '',
  config JSONB NOT NULL DEFAULT '{}'
);

-- inbox
CREATE TABLE inbox (
  id SERIAL PRIMARY KEY,
  created_ts BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM NOW()),
  sender_id INTEGER NOT NULL,
  receiver_id INTEGER NOT NULL,
  status TEXT NOT NULL,
  message TEXT NOT NULL
);

-- reaction
CREATE TABLE reaction (
  id SERIAL PRIMARY KEY,
  created_ts BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM NOW()),
  creator_id INTEGER NOT NULL,
  content_id TEXT NOT NULL,
  reaction_type TEXT NOT NULL,
  UNIQUE(creator_id, content_id, reaction_type)
);

-- Class tables for miao-error-book feature

-- class
CREATE TABLE class (
  id SERIAL PRIMARY KEY,
  uid TEXT NOT NULL UNIQUE,
  name VARCHAR(255) NOT NULL,
  description TEXT,
  creator_id INTEGER NOT NULL,
  invite_code VARCHAR(50) UNIQUE,
  visibility VARCHAR(256) NOT NULL DEFAULT 'PUBLIC',
  settings JSONB DEFAULT NULL,
  created_ts BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM NOW()),
  updated_ts BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM NOW())
);

-- class_member
CREATE TABLE class_member (
  id SERIAL PRIMARY KEY,
  class_id INTEGER NOT NULL,
  user_id INTEGER NOT NULL,
  role VARCHAR(20) NOT NULL DEFAULT 'STUDENT',
  joined_ts BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM NOW()),
  invited_by INTEGER,
  UNIQUE (class_id, user_id),
  FOREIGN KEY (class_id) REFERENCES class(id) ON DELETE CASCADE,
  FOREIGN KEY (user_id) REFERENCES "user"(id) ON DELETE CASCADE,
  CHECK (role IN ('TEACHER', 'ASSISTANT', 'STUDENT', 'PARENT'))
);

-- class_memo_visibility
CREATE TABLE class_memo_visibility (
  id SERIAL PRIMARY KEY,
  class_id INTEGER NOT NULL,
  memo_id INTEGER NOT NULL,
  visibility VARCHAR(20) NOT NULL DEFAULT 'PUBLIC',
  shared_by INTEGER NOT NULL,
  shared_ts BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM NOW()),
  description TEXT,
  UNIQUE (class_id, memo_id),
  FOREIGN KEY (class_id) REFERENCES class(id) ON DELETE CASCADE,
  FOREIGN KEY (memo_id) REFERENCES memo(id) ON DELETE CASCADE,
  CHECK (visibility IN ('PUBLIC', 'ANONYMOUS', 'TEACHER_ONLY'))
);

-- class_tag_template
CREATE TABLE class_tag_template (
  id SERIAL PRIMARY KEY,
  class_id INTEGER NOT NULL,
  name VARCHAR(255) NOT NULL,
  color VARCHAR(20) DEFAULT '#808080',
  description TEXT,
  created_ts BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM NOW()),
  updated_ts BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM NOW()),
  UNIQUE (class_id, name),
  FOREIGN KEY (class_id) REFERENCES class(id) ON DELETE CASCADE
);

-- Create indexes for better performance
CREATE INDEX idx_class_invite_code ON class (invite_code);
CREATE INDEX idx_class_created_ts ON class (created_ts);
CREATE INDEX idx_class_uid ON class (uid);
CREATE INDEX idx_class_creator_id ON class (creator_id);
CREATE INDEX idx_class_visibility ON class (visibility);
CREATE INDEX idx_class_member_class_id ON class_member (class_id);
CREATE INDEX idx_class_member_user_id ON class_member (user_id);
CREATE INDEX idx_class_member_role ON class_member (role);
CREATE INDEX idx_class_memo_visibility_class_id ON class_memo_visibility (class_id);
CREATE INDEX idx_class_memo_visibility_memo_id ON class_memo_visibility (memo_id);
CREATE INDEX idx_class_tag_template_class_id ON class_tag_template (class_id);
