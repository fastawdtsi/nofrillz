CREATE TABLE `post_bookmarks` (
  `user_id` bigint unsigned NOT NULL,
  `post_id` bigint unsigned NOT NULL,
  `created` timestamp(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  PRIMARY KEY (`user_id`,`post_id`),
  KEY `idx_post_bookmarks_user_created_post_id` (`user_id`,`created`,`post_id`),
  KEY `idx_post_bookmarks_post_id` (`post_id`),
  CONSTRAINT `fk_post_bookmarks_user_id` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE,
  CONSTRAINT `fk_post_bookmarks_post_id` FOREIGN KEY (`post_id`) REFERENCES `posts` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
