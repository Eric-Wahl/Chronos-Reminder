import { httpClient } from "./http";
import type { DFMNote, DFMItem } from "./types";

/**
 * Don't Forget Me Service
 * Handles the private note, its items and its recurring reminder
 */
class DFMService {
  /**
   * Fetch the user's note with its items (created on first access)
   */
  async getNote(): Promise<DFMNote | null> {
    try {
      return await httpClient.get<DFMNote>("/api/dfm");
    } catch (error) {
      console.error("Failed to fetch DFM note:", error);
      return null;
    }
  }

  /**
   * Add an item to the note
   */
  async addItem(content: string): Promise<DFMItem | null> {
    try {
      return await httpClient.post<DFMItem>("/api/dfm/items", { content });
    } catch (error) {
      console.error("Failed to add DFM item:", error);
      // Rethrow (instead of swallowing into a null return) so the caller can
      // surface the backend's specific error message, e.g. the item limit
      // being reached.
      throw error instanceof Error ? error : new Error("Failed to add item");
    }
  }

  /**
   * Update an item's content and/or checked state
   */
  async updateItem(
    itemId: string,
    data: { content?: string; checked?: boolean }
  ): Promise<DFMItem | null> {
    try {
      return await httpClient.put<DFMItem>(`/api/dfm/items/${itemId}`, data);
    } catch (error) {
      console.error("Failed to update DFM item:", error);
      return null;
    }
  }

  /**
   * Delete an item from the note
   */
  async deleteItem(itemId: string): Promise<boolean> {
    try {
      await httpClient.delete(`/api/dfm/items/${itemId}`);
      return true;
    } catch (error) {
      console.error("Failed to delete DFM item:", error);
      return false;
    }
  }

  /**
   * Set the recurring reminder of the note
   */
  async setReminder(data: {
    date?: string; // ISO 8601, defaults to today
    time: string; // HH:mm format
    recurrence: string; // Uppercase string (e.g., "DAILY")
    destinations: Array<"discord_dm" | "email">;
  }): Promise<DFMNote | null> {
    try {
      return await httpClient.put<DFMNote>("/api/dfm/reminder", data);
    } catch (error) {
      console.error("Failed to set DFM reminder:", error);
      return null;
    }
  }

  /**
   * Remove the reminder of the note
   */
  async removeReminder(): Promise<DFMNote | null> {
    try {
      return await httpClient.delete<DFMNote>("/api/dfm/reminder");
    } catch (error) {
      console.error("Failed to remove DFM reminder:", error);
      return null;
    }
  }

  /**
   * Send the note to the user immediately (test delivery)
   */
  async sendNow(): Promise<boolean> {
    try {
      await httpClient.post("/api/dfm/send", {});
      return true;
    } catch (error) {
      console.error("Failed to send DFM note:", error);
      return false;
    }
  }

}

// Export singleton instance
export const dfmService = new DFMService();
