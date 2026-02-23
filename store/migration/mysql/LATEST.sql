-- system_setting
CREATE TABLE `system_setting` (
  `name` VARCHAR(256) NOT NULL PRIMARY KEY,
  `value` LONGTEXT NOT NULL,
  `description` TEXT NOT NULL
);

-- user
CREATE TABLE `user` (
  `id` INT NOT NULL AUTO_INCREMENT PRIMARY KEY,
  `created_ts` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_ts` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `row_status` VARCHAR(256) NOT NULL DEFAULT 'NORMAL',
  `username` VARCHAR(256) NOT NULL UNIQUE,
  `role` VARCHAR(256) NOT NULL DEFAULT 'USER',
  `email` VARCHAR(256) NOT NULL DEFAULT '',
  `nickname` VARCHAR(256) NOT NULL DEFAULT '',
  `password_hash` VARCHAR(256) NOT NULL,
  `avatar_url` LONGTEXT NOT NULL,
  `description` VARCHAR(256) NOT NULL DEFAULT ''
);

-- user_setting
CREATE TABLE `user_setting` (
  `user_id` INT NOT NULL,
  `key` VARCHAR(256) NOT NULL,
  `value` LONGTEXT NOT NULL,
  UNIQUE(`user_id`,`key`)
);

-- memo
CREATE TABLE `memo` (
  `id` INT NOT NULL AUTO_INCREMENT PRIMARY KEY,
  `uid` VARCHAR(256) NOT NULL UNIQUE,
  `creator_id` INT NOT NULL,
  `created_ts` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_ts` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `row_status` VARCHAR(256) NOT NULL DEFAULT 'NORMAL',
  `content` TEXT NOT NULL,
  `visibility` VARCHAR(256) NOT NULL DEFAULT 'PRIVATE',
  `pinned` BOOLEAN NOT NULL DEFAULT FALSE,
  `payload` JSON NOT NULL
);

-- memo_relation
CREATE TABLE `memo_relation` (
  `memo_id` INT NOT NULL,
  `related_memo_id` INT NOT NULL,
  `type` VARCHAR(256) NOT NULL,
  UNIQUE(`memo_id`,`related_memo_id`,`type`)
);

-- attachment
CREATE TABLE `attachment` (
  `id` INT NOT NULL AUTO_INCREMENT PRIMARY KEY,
  `uid` VARCHAR(256) NOT NULL UNIQUE,
  `creator_id` INT NOT NULL,
  `created_ts` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_ts` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `filename` TEXT NOT NULL,
  `blob` MEDIUMBLOB,
  `type` VARCHAR(256) NOT NULL DEFAULT '',
  `size` INT NOT NULL DEFAULT '0',
  `memo_id` INT DEFAULT NULL,
  `storage_type` VARCHAR(256) NOT NULL DEFAULT '',
  `reference` TEXT NOT NULL DEFAULT (''),
  `payload` TEXT NOT NULL
);

-- activity
CREATE TABLE `activity` (
  `id` INT NOT NULL AUTO_INCREMENT PRIMARY KEY,
  `creator_id` INT NOT NULL,
  `created_ts` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `type` VARCHAR(256) NOT NULL DEFAULT '',
  `level` VARCHAR(256) NOT NULL DEFAULT 'INFO',
  `payload` TEXT NOT NULL
);

-- idp
CREATE TABLE `idp` (
  `id` INT NOT NULL AUTO_INCREMENT PRIMARY KEY,
  `name` TEXT NOT NULL,
  `type` TEXT NOT NULL,
  `identifier_filter` VARCHAR(256) NOT NULL DEFAULT '',
  `config` TEXT NOT NULL
);

-- inbox
CREATE TABLE `inbox` (
  `id` INT NOT NULL AUTO_INCREMENT PRIMARY KEY,
  `created_ts` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `sender_id` INT NOT NULL,
  `receiver_id` INT NOT NULL,
  `status` TEXT NOT NULL,
  `message` TEXT NOT NULL
);

-- reaction
CREATE TABLE `reaction` (
  `id` INT NOT NULL AUTO_INCREMENT PRIMARY KEY,
  `created_ts` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `creator_id` INT NOT NULL,
  `content_id` VARCHAR(256) NOT NULL,
  `reaction_type` VARCHAR(256) NOT NULL,
  UNIQUE(`creator_id`,`content_id`,`reaction_type`)  
);

-- Class tables for miao-error-book feature

-- class
CREATE TABLE `class` (
  `id` INT AUTO_INCREMENT PRIMARY KEY,
  `uid` VARCHAR(256) NOT NULL UNIQUE,
  `name` VARCHAR(255) NOT NULL,
  `description` TEXT,
  `creator_id` INT NOT NULL,
  `invite_code` VARCHAR(50) UNIQUE,
  `visibility` VARCHAR(256) NOT NULL DEFAULT 'PUBLIC',
  `settings` JSON DEFAULT NULL,
  `created_ts` BIGINT NOT NULL,
  `updated_ts` BIGINT NOT NULL,
  INDEX `idx_class_invite_code` (`invite_code`),
  INDEX `idx_class_created_ts` (`created_ts`),
  INDEX `idx_class_uid` (`uid`),
  INDEX `idx_class_creator_id` (`creator_id`),
  INDEX `idx_class_visibility` (`visibility`)
) ENGINE=InnoDB CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

-- class_member
CREATE TABLE `class_member` (
  `id` INT AUTO_INCREMENT PRIMARY KEY,
  `class_id` INT NOT NULL,
  `user_id` INT NOT NULL,
  `role` ENUM('TEACHER', 'ASSISTANT', 'STUDENT', 'PARENT') NOT NULL DEFAULT 'STUDENT',
  `joined_ts` BIGINT NOT NULL,
  `invited_by` INT,
  UNIQUE KEY `uk_class_member_user` (`class_id`, `user_id`),
  FOREIGN KEY (`class_id`) REFERENCES `class`(`id`) ON DELETE CASCADE,
  FOREIGN KEY (`user_id`) REFERENCES `user`(`id`) ON DELETE CASCADE,
  FOREIGN KEY (`invited_by`) REFERENCES `user`(`id`) ON DELETE SET NULL,
  INDEX `idx_class_member_class_id` (`class_id`),
  INDEX `idx_class_member_user_id` (`user_id`),
  INDEX `idx_class_member_role` (`role`),
  INDEX `idx_class_member_invited_by` (`invited_by`)
) ENGINE=InnoDB CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

-- class_memo_visibility
CREATE TABLE `class_memo_visibility` (
  `id` INT AUTO_INCREMENT PRIMARY KEY,
  `class_id` INT NOT NULL,
  `memo_id` INT NOT NULL,
  `visibility` VARCHAR(256) NOT NULL DEFAULT 'PUBLIC',
  `shared_by` INT NOT NULL,
  `shared_ts` BIGINT NOT NULL,
  `description` TEXT,
  UNIQUE KEY `uk_class_memo_visibility` (`class_id`, `memo_id`),
  FOREIGN KEY (`class_id`) REFERENCES `class`(`id`) ON DELETE CASCADE,
  FOREIGN KEY (`memo_id`) REFERENCES `memo`(`id`) ON DELETE CASCADE,
  FOREIGN KEY (`shared_by`) REFERENCES `user`(`id`) ON DELETE CASCADE,
  INDEX `idx_class_memo_visibility_class_id` (`class_id`),
  INDEX `idx_class_memo_visibility_memo_id` (`memo_id`),
  INDEX `idx_class_memo_visibility_shared_by` (`shared_by`)
) ENGINE=InnoDB CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

-- class_tag_template
CREATE TABLE `class_tag_template` (
  `id` INT AUTO_INCREMENT PRIMARY KEY,
  `class_id` INT NOT NULL,
  `name` VARCHAR(255) NOT NULL,
  `color` VARCHAR(20) DEFAULT '#808080',
  `description` TEXT,
  `created_ts` BIGINT NOT NULL,
  `updated_ts` BIGINT NOT NULL,
  UNIQUE KEY `uk_class_tag_template` (`class_id`, `name`),
  FOREIGN KEY (`class_id`) REFERENCES `class`(`id`) ON DELETE CASCADE,
  INDEX `idx_class_tag_template_class_id` (`class_id`)
) ENGINE=InnoDB CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
