export type AccountType = "human" | "ai" | "system";

export type Stats = {
  user_count: number;
  human_user_count: number;
  ai_user_count: number;
  blocked_user_count: number;
  post_count: number;
  ai_post_count: number;
  enabled_ai_account_count: number;
};

export type AdminUser = {
  id: string;
  email: string;
  username: string;
  first_name: string;
  last_name: string;
  about: string;
  account_type: AccountType;
  avatar_url: string;
  ai_account_id?: string;
  ai_enabled: boolean;
  ai_next_post_at?: string;
  ai_last_post_at?: string;
  ai_generation_status?: string;
  ai_generation_error?: string;
  ai_min_posts_per_day: number;
  ai_max_posts_per_day: number;
  blocked_at?: string;
  post_count: number;
  created_at: string;
};

export type AdminPost = {
  id: string;
  body: string;
  source: "human" | "ai" | "system";
  created_at: string;
  user: {
    user_id: string;
    username: string;
    first_name: string;
    last_name: string;
    account_type: AccountType;
    avatar_url: string;
  };
};

export type AdminUsersResponse = {
  users: AdminUser[];
  next_cursor?: string;
};

export type AdminPostsResponse = {
  posts: AdminPost[];
  next_cursor?: string;
};

export type AIAccountRecord = {
  account: {
    id: string;
    user_id: string;
    enabled: boolean;
    topic: string;
    description: string;
    system_prompt: string;
    style_prompt: string;
    min_posts_per_day: number;
    max_posts_per_day: number;
    next_generate_at?: string | null;
    last_generated_at?: string | null;
    generation_status: string;
    generation_error: string;
    consecutive_failures: number;
  };
  user: {
    id: string;
    email: string;
    username: string;
    first_name: string;
    last_name: string;
    about: string;
    account_type: AccountType;
    avatar_url: string;
  };
};

export type AIAccountForm = {
  email: string;
  username: string;
  first_name: string;
  last_name: string;
  about: string;
  enabled: boolean;
  topic: string;
  description: string;
  system_prompt: string;
  style_prompt: string;
  min_posts_per_day: number;
  max_posts_per_day: number;
};

export type AIPostResult = {
  post?: {
    id: string;
    body: string;
    source: string;
  };
  generation: {
    id: string;
    status: string;
    candidate_body: string;
    final_body: string;
    reject_reason: string;
    error: string;
    model: string;
  };
};

export type ConnectionSettings = {
  baseUrl: string;
  adminApiKey: string;
};

export type AIStatus = {
 provider: string;
 model: string;
 development_mode: boolean;
 development_min_interval_seconds: number;
 development_max_interval_seconds: number;
 poll_interval_seconds: number;
};
