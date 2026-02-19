-- Create class tables for miao-error-book feature
-- This migration adds support for class management, member roles, memo visibility control, and tag templates.

-- Class table: stores class information
CREATE TABLE IF NOT EXISTS class (
  id INT AUTO_INCREMENT PRIMARY KEY,
  name VARCHAR(255) NOT NULL,
  description TEXT,
  invite_code VARCHAR(50) UNIQUE NOT NULL,
  settings JSON DEFAULT NULL,
  created_ts TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_ts TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  INDEX idx_class_invite_code (invite_code),
  INDEX idx_class_created_ts (created_ts)
) ENGINE=InnoDB CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

-- Class member table: stores membership and role information
CREATE TABLE IF NOT EXISTS class_member (
  id INT AUTO_INCREMENT PRIMARY KEY,
  class_id INT NOT NULL,
  user_id INT NOT NULL,
  role ENUM('TEACHER', 'ASSISTANT', 'STUDENT', 'PARENT') NOT NULL DEFAULT 'STUDENT',
  joined_ts TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  UNIQUE KEY uk_class_member_user (class_id, user_id),
  FOREIGN KEY (class_id) REFERENCES class(id) ON DELETE CASCADE,
  FOREIGN KEY (user_id) REFERENCES user(id) ON DELETE CASCADE,
  INDEX idx_class_member_class_id (class_id),
  INDEX idx_class_member_user_id (user_id),
  INDEX idx_class_member_role (role)
) ENGINE=InnoDB CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

-- Class memo visibility table: controls memo visibility within a class
CREATE TABLE IF NOT EXISTS class_memo_visibility (
  id INT AUTO_INCREMENT PRIMARY KEY,
  class_id INT NOT NULL,
  memo_id INT NOT NULL,
  visibility ENUM('PUBLIC', 'ANONYMOUS', 'TEACHER_ONLY') NOT NULL DEFAULT 'PUBLIC',
  created_ts TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_ts TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  UNIQUE KEY uk_class_memo_visibility (class_id, memo_id),
  FOREIGN KEY (class_id) REFERENCES class(id) ON DELETE CASCADE,
  FOREIGN KEY (memo_id) REFERENCES memo(id) ON DELETE CASCADE,
  INDEX idx_class_memo_visibility_class_id (class_id),
  INDEX idx_class_memo_visibility_memo_id (memo_id)
) ENGINE=InnoDB CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

-- Class tag template table: predefined tags for a class
CREATE TABLE IF NOT EXISTS class_tag_template (
  id INT AUTO_INCREMENT PRIMARY KEY,
  class_id INT NOT NULL,
  name VARCHAR(255) NOT NULL,
  color VARCHAR(20) DEFAULT '#808080',
  created_ts TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_ts TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  UNIQUE KEY uk_class_tag_template (class_id, name),
  FOREIGN KEY (class_id) REFERENCES class(id) ON DELETE CASCADE,
  INDEX idx_class_tag_template_class_id (class_id)
) ENGINE=InnoDB CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;