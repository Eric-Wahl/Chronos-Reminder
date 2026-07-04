import { httpClient } from "./http";
import type {
  Reminder,
  RemindersResponse,
  ReminderError,
  ReminderErrorsResponse,
  ApiResponse,
} from "./types";

/**
 * Reminders Service
 * Handles all reminder-related API calls
 */
class RemindersService {
  /**
   * Normalize reminder data from API response
   * Ensures all date fields are strings and keeps recurrence_type as uppercase string
   */
  private normalizeReminder(
    reminder: Record<string, unknown> & Partial<Reminder>,
  ): Reminder {
    return {
      id: String(reminder.id || ""),
      account_id: String(reminder.account_id || ""),
      remind_at_utc: String(reminder.remind_at_utc || ""),
      snoozed_at_utc: reminder.snoozed_at_utc
        ? String(reminder.snoozed_at_utc)
        : null,
      next_fire_utc: reminder.next_fire_utc
        ? String(reminder.next_fire_utc)
        : null,
      message: String(reminder.message || ""),
      created_at: String(reminder.created_at || ""),
      recurrence_type: String(reminder.recurrence_type || "ONCE"),
      is_paused: Boolean(reminder.is_paused || false),
      destinations: Array.isArray(reminder.destinations)
        ? reminder.destinations
        : [],
    };
  }

  /**
   * Fetch all reminders for the authenticated user
   */
  async getReminders(): Promise<Reminder[]> {
    const response =
      await httpClient.get<ApiResponse<RemindersResponse>>("/api/reminders");
    const data = (response.data || response) as RemindersResponse;
    const reminders = data.reminders || [];

    // Normalize all reminders to ensure proper data types
    return reminders.map((reminder) =>
      this.normalizeReminder(
        reminder as Record<string, unknown> & Partial<Reminder>,
      ),
    );
  }

  /**
   * Fetch a single reminder by ID
   */
  async getReminder(reminderId: string): Promise<Reminder | null> {
    try {
      const response = await httpClient.get<ApiResponse<Reminder>>(
        `/api/reminders/${reminderId}`,
      );
      const reminder = (response.data || response) as Reminder;
      return this.normalizeReminder(
        reminder as Record<string, unknown> & Partial<Reminder>,
      );
    } catch (error) {
      console.error("Failed to fetch reminder:", error);
      return null;
    }
  }

  /**
   * Fetch all reminder errors for the authenticated user
   */
  async getReminderErrors(): Promise<ReminderError[]> {
    try {
      const response = await httpClient.get<
        ApiResponse<ReminderErrorsResponse>
      >("/api/reminders/errors");
      const data = (response.data || response) as ReminderErrorsResponse;
      return data.errors || [];
    } catch (error) {
      console.error("Failed to fetch reminder errors:", error);
      return [];
    }
  }

  /**
   * Delete a reminder by ID
   */
  async deleteReminder(reminderId: string): Promise<boolean> {
    try {
      await httpClient.delete(`/api/reminders/${reminderId}`);
      return true;
    } catch (error) {
      console.error("Failed to delete reminder:", error);
      return false;
    }
  }

  /**
   * Create a new reminder
   */
  async createReminder(data: {
    date: string; // ISO 8601 date format
    time: string; // HH:mm format
    message: string;
    recurrence: string; // Uppercase string (e.g., "DAILY")
    destinations: Array<{
      type: "discord_dm" | "discord_channel" | "webhook" | "email" | "android_push";
      metadata: Record<string, unknown>;
    }>;
  }): Promise<Reminder | null> {
    try {
      const response = await httpClient.post<ApiResponse<Reminder>>(
        "/api/reminders",
        data,
      );
      const reminder = (response.data || response) as Reminder;
      return this.normalizeReminder(
        reminder as Record<string, unknown> & Partial<Reminder>,
      );
    } catch (error) {
      console.error("Failed to create reminder:", error);
      // Rethrow (instead of swallowing into a null return) so the caller can
      // surface the backend's specific error message, e.g. the reminder
      // limit being reached.
      throw error instanceof Error ? error : new Error("Failed to create reminder");
    }
  }

  /**
   * Update an existing reminder
   */
  async updateReminder(
    reminderId: string,
    data: {
      message?: string;
      date?: string;
      time?: string;
      recurrence?: string; // Uppercase string (e.g., "DAILY")
      destinations?: Array<{
        type: "discord_dm" | "discord_channel" | "webhook" | "email" | "android_push";
        metadata: Record<string, unknown>;
      }>;
    },
  ): Promise<Reminder | null> {
    try {
      const response = await httpClient.put<ApiResponse<Reminder>>(
        `/api/reminders/${reminderId}`,
        data,
      );
      const reminder = (response.data || response) as Reminder;
      return this.normalizeReminder(
        reminder as Record<string, unknown> & Partial<Reminder>,
      );
    } catch (error) {
      console.error("Failed to update reminder:", error);
      return null;
    }
  }

  /**
   * Pause a reminder
   */
  async pauseReminder(reminderId: string): Promise<boolean> {
    try {
      await httpClient.post(`/api/reminders/${reminderId}/pause`, {});
      return true;
    } catch (error) {
      console.error("Failed to pause reminder:", error);
      return false;
    }
  }

  /**
   * Resume a reminder
   */
  async resumeReminder(reminderId: string): Promise<boolean> {
    try {
      await httpClient.post(`/api/reminders/${reminderId}/resume`, {});
      return true;
    } catch (error) {
      console.error("Failed to resume reminder:", error);
      return false;
    }
  }

  /**
   * Duplicate a reminder
   */
  async duplicateReminder(reminderId: string): Promise<Reminder | null> {
    try {
      const response = await httpClient.post<ApiResponse<Reminder>>(
        `/api/reminders/${reminderId}/duplicate`,
        {},
      );
      const reminder = (response.data || response) as Reminder;
      return this.normalizeReminder(
        reminder as Record<string, unknown> & Partial<Reminder>,
      );
    } catch (error) {
      console.error("Failed to duplicate reminder:", error);
      return null;
    }
  }
}

// Export singleton instance
export const remindersService = new RemindersService();
