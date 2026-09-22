CREATE TABLE `users` (
  `id` bigint unsigned NOT NULL,
  `email` varchar(320) NOT NULL,
  `username` varchar(64) NOT NULL,
  `first_name` varchar(100) NOT NULL,
  `last_name` varchar(100) NOT NULL,
  `about` varchar(1024) NOT NULL,
  `account_type` varchar(16) NOT NULL DEFAULT 'human',
  `password_hash` varbinary(255) NOT NULL,
  `password_salt` varbinary(255) DEFAULT NULL,
  `created` timestamp(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  `updated` timestamp(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
  `deleted` timestamp(6) NULL DEFAULT NULL,
  `blocked` timestamp(6) NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uq_users_email` (`email`),
  UNIQUE KEY `uq_users_handle` (`username`),
  KEY `idx_users_first_name` (`first_name`),
  KEY `idx_users_last_name` (`last_name`),
  KEY `idx_users_blocked` (`blocked`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE `posts` (
  `id` bigint unsigned NOT NULL,
  `user_id` bigint unsigned NOT NULL,
  `body` text NOT NULL,
  `source` varchar(16) NOT NULL DEFAULT 'human',
  `created` timestamp(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  `updated` timestamp(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
  `deleted` timestamp(6) NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_posts_user_id` (`user_id`,`id`),
  KEY `idx_posts_feed` (`user_id`,`deleted`,`created`,`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;


CREATE TABLE `follows` (
  `follower_id` bigint unsigned NOT NULL,
  `following_id` bigint unsigned NOT NULL,
  `created` timestamp(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  PRIMARY KEY (`follower_id`,`following_id`),
  KEY `idx_follows_following_id` (`following_id`),
  KEY `idx_follows_following_created_follower` (`following_id`,`created`,`follower_id`),
  CONSTRAINT `fk_follows_follower_id` FOREIGN KEY (`follower_id`) REFERENCES `users` (`id`),
  CONSTRAINT `fk_follows_following_id` FOREIGN KEY (`following_id`) REFERENCES `users` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE `likes` (
  `post_id` bigint unsigned NOT NULL,
  `user_id` bigint unsigned NOT NULL,
  `created` timestamp(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  PRIMARY KEY (`post_id`,`user_id`),
  KEY `idx_likes_user_id` (`user_id`),
  CONSTRAINT `fk_likes_post_id` FOREIGN KEY (`post_id`) REFERENCES `posts` (`id`),
  CONSTRAINT `fk_likes_user_id` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE `users_apns_device_tokens` (
  `token` varchar(255) NOT NULL,
  `user_id` bigint unsigned NOT NULL,
  `created_at` timestamp(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  `updated_at` timestamp(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
  PRIMARY KEY (`token`),
  KEY `idx_users_apns_device_tokens_user_id` (`user_id`),
  CONSTRAINT `fk_users_apns_device_tokens_user_id` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE `user_push_notification_settings` (
  `user_id` bigint unsigned NOT NULL,
  `enabled` boolean NOT NULL DEFAULT TRUE,
  `new_followers` boolean NOT NULL DEFAULT TRUE,
  `new_likes` boolean NOT NULL DEFAULT FALSE,
  `replies` boolean NOT NULL DEFAULT TRUE,
  `created_at` timestamp(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  `updated_at` timestamp(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
  PRIMARY KEY (`user_id`),
  CONSTRAINT `fk_user_push_notification_settings_user_id` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE `ai_accounts` (
  `id` bigint unsigned NOT NULL,
  `user_id` bigint unsigned NOT NULL,
  `enabled` boolean NOT NULL DEFAULT TRUE,
  `topic` varchar(128) NOT NULL,
  `description` text NULL,
  `system_prompt` text NOT NULL,
  `style_prompt` text NULL,
  `min_posts_per_day` int NOT NULL DEFAULT 1,
  `max_posts_per_day` int NOT NULL DEFAULT 2,
  `next_generate_at` timestamp(6) NULL DEFAULT NULL,
  `last_generated_at` timestamp(6) NULL DEFAULT NULL,
  `generation_status` varchar(16) NOT NULL DEFAULT 'idle',
  `generation_started_at` timestamp(6) NULL DEFAULT NULL,
  `generation_error` text NULL,
  `created_at` timestamp(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  `updated_at` timestamp(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
  PRIMARY KEY (`id`),
  UNIQUE KEY `uq_ai_accounts_user_id` (`user_id`),
  KEY `idx_ai_accounts_due` (`enabled`,`generation_status`,`next_generate_at`),
  KEY `idx_ai_accounts_started` (`generation_status`,`generation_started_at`),
  CONSTRAINT `fk_ai_accounts_user_id` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE `ai_post_generations` (
  `id` bigint unsigned NOT NULL,
  `ai_account_id` bigint unsigned NOT NULL,
  `post_id` bigint unsigned NULL DEFAULT NULL,
  `status` varchar(16) NOT NULL,
  `prompt` text NULL,
  `candidate_body` text NULL,
  `final_body` text NULL,
  `reject_reason` text NULL,
  `error` text NULL,
  `model` varchar(128) NULL DEFAULT NULL,
  `created_at` timestamp(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  PRIMARY KEY (`id`),
  KEY `idx_ai_post_generations_account_created` (`ai_account_id`,`created_at`),
  KEY `idx_ai_post_generations_post_id` (`post_id`),
  CONSTRAINT `fk_ai_post_generations_ai_account_id` FOREIGN KEY (`ai_account_id`) REFERENCES `ai_accounts` (`id`),
  CONSTRAINT `fk_ai_post_generations_post_id` FOREIGN KEY (`post_id`) REFERENCES `posts` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE `sessions` (
  `id` bigint unsigned NOT NULL,
  `user_id` bigint unsigned NOT NULL,
  `created` timestamp(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  `updated` timestamp(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
  `revoked` timestamp(6) NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_sessions_user_id` (`user_id`),
  CONSTRAINT `fk_sessions_user_id` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE `session_refresh_tokens` (
  `token_hash` char(64) NOT NULL,
  `session_id` bigint unsigned NOT NULL,
  `created` timestamp(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  `expires_at` timestamp(6) NOT NULL,
  `revoked` timestamp(6) NULL DEFAULT NULL,
  PRIMARY KEY (`token_hash`),
  KEY `idx_session_refresh_tokens_session_id` (`session_id`),
  KEY `idx_session_refresh_tokens_expires_at` (`expires_at`),
  CONSTRAINT `fk_session_refresh_tokens_session_id` FOREIGN KEY (`session_id`) REFERENCES `sessions` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
