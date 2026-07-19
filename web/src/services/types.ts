/**
 * API Response Wrapper
 */
export interface ApiResponse<T = unknown> {
  data?: T;
  error?: string;
  message?: string;
}

/**
 * Authentication Types
 */
export interface LoginResponse {
  id: string;
  email: string;
  username: string;
  token: string;
  expires_at: string;
  message: string;
}

export interface RegisterResponse {
  id: string;
  email: string;
  username: string;
  message: string;
}

export interface VerifyEmailResponse {
  id: string;
  email: string;
  username: string;
  token: string;
  expires_at: string;
  message: string;
  data?: { [key: string]: unknown };
}

export interface SessionData {
  user_id: string;
  email: string;
  username: string;
  expires_at: string;
}

export interface RegisterRequest {
  email: string;
  username: string;
  password: string;
  timezone: string;
  [key: string]: unknown;
}

export interface LoginRequest {
  email: string;
  password: string;
  remember_me: boolean;
  [key: string]: unknown;
}

/**
 * Password Reset Types
 */
export interface RequestPasswordResetRequest {
  email: string;
  [key: string]: unknown;
}

export interface RequestPasswordResetResponse {
  message: string;
}

export interface VerifyResetTokenRequest {
  email: string;
  token: string;
  [key: string]: unknown;
}

export interface VerifyResetTokenResponse {
  valid: boolean;
  message: string;
}

export interface ResetPasswordRequest {
  email: string;
  token: string;
  password: string;
  [key: string]: unknown;
}

export interface ResetPasswordResponse {
  message: string;
}

/**
 * Reminder Types
 */
export interface Reminder {
  id: string;
  account_id: string;
  remind_at_utc: string;
  snoozed_at_utc?: string | null;
  next_fire_utc?: string | null;
  message: string;
  created_at: string;
  recurrence_type: string; // Stored as uppercase string (e.g., "DAILY")
  is_paused: boolean;
  destinations?: ReminderDestination[];
}

export interface ReminderDestination {
  id: string;
  reminder_id: string;
  type: "discord_dm" | "discord_channel" | "webhook" | "email" | "android_push";
  metadata: Record<string, unknown>;
}

export interface RemindersResponse {
  reminders: Reminder[];
  count: number;
}

/**
 * Account Types
 */
export interface AccountPreferences {
  discord_send_image: boolean;
  discord_enable_snooze: boolean;
}

export interface Account {
  id: string;
  email: string;
  username: string;
  email_verified: boolean;
  has_password: boolean;
  timezone: string;
  created_at: string;
  identities?: AccountIdentity[];
  preferences?: AccountPreferences;
}

export interface AccountIdentity {
  id: string;
  account_id: string;
  provider: string;
  external_id: string;
  username?: string;
  avatar?: string;
  created_at: string;
}

export interface AccountResponse {
  id: string;
  email: string;
  username: string;
  timezone: string;
  created_at: string;
  identities?: AccountIdentity[];
}

/**
 * Error Types
 */
export interface ReminderError {
  id: string;
  reminder_id: string;
  error_message: string;
  created_at: string;
}

export interface ReminderErrorsResponse {
  errors: ReminderError[];
  count: number;
}

/**
 * Discord Guild Types
 */
export interface DiscordGuild {
  id: string;
  name: string;
  icon: string;
  owner: boolean;
  permissions: number;
  features: string[];
}

export interface DiscordChannel {
  id: string;
  name: string;
  type: number;
  position: number;
  topic?: string | null;
}

export interface DiscordRole {
  id: string;
  name: string;
  color: number;
  position: number;
  permissions: number;
  managed: boolean;
  mentionable: boolean;
}

export interface GetUserGuildsResponse {
  guilds: DiscordGuild[];
  error?: string;
}

export interface GetGuildChannelsResponse {
  channels: DiscordChannel[];
  error?: string;
}

export interface GetGuildRolesResponse {
  roles: DiscordRole[];
  error?: string;
}

/**
 * Don't Forget Me Types
 */
export interface DFMItem {
  id: string;
  note_id: string;
  content: string;
  checked: boolean;
  position: number;
  created_at: string;
  updated_at: string;
}

export interface DFMNote {
  id: string;
  remind_at_utc?: string | null;
  next_fire_utc?: string | null;
  recurrence_type: string; // Uppercase string (e.g., "DAILY")
  has_reminder: boolean;
  destinations: Array<"discord_dm" | "email">;
  items: DFMItem[];
  created_at: string;
  updated_at: string;
}

/**
 * Timezone Types
 */
export interface Timezone {
  id: number;
  name: string;
  gmt_offset: number;
  iana_location: string;
}

/**
 * API Key Types
 */
export interface APIKey {
  id: string;
  name: string;
  scopes: string;
  created_at: string;
  key?: string; // Only present on creation
}

export interface CreateAPIKeyResponse {
  id: string;
  name: string;
  scopes: string;
  created_at: string;
  key: string; // The plain text key (only shown once)
}

export interface ListAPIKeysResponse {
  keys: APIKey[];
}
