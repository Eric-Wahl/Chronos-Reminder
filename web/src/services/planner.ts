import { httpClient } from "./http";
import type { PlannerItem, PlannerPeriod } from "./types";

/**
 * Day Planner Service
 * Handles the account's day-planning checklist (web/mobile only feature,
 * no Discord bot surface). Items can optionally be linked to a DFM item so
 * the checked state stays in sync between the two.
 */
class PlannerService {
  /**
   * Fetch all of the account's planner items
   */
  async getItems(): Promise<PlannerItem[]> {
    try {
      const response = await httpClient.get<{ items: PlannerItem[] }>(
        "/api/planner/items"
      );
      return response.items || [];
    } catch (error) {
      console.error("Failed to fetch planner items:", error);
      return [];
    }
  }

  /**
   * Add a new item, optionally linked to an existing DFM item
   */
  async addItem(data: {
    content: string;
    period: PlannerPeriod;
    dfm_item_id?: string;
  }): Promise<PlannerItem> {
    try {
      return await httpClient.post<PlannerItem>("/api/planner/items", data);
    } catch (error) {
      console.error("Failed to add planner item:", error);
      throw error instanceof Error
        ? error
        : new Error("Failed to add planner item");
    }
  }

  /**
   * Update an item's content, checked state and/or period
   */
  async updateItem(
    itemId: string,
    data: { content?: string; checked?: boolean; period?: PlannerPeriod }
  ): Promise<PlannerItem | null> {
    try {
      return await httpClient.put<PlannerItem>(
        `/api/planner/items/${itemId}`,
        data
      );
    } catch (error) {
      console.error("Failed to update planner item:", error);
      return null;
    }
  }

  /**
   * Delete a single item
   */
  async deleteItem(itemId: string): Promise<boolean> {
    try {
      await httpClient.delete(`/api/planner/items/${itemId}`);
      return true;
    } catch (error) {
      console.error("Failed to delete planner item:", error);
      return false;
    }
  }

  /**
   * Persist a new order/period for multiple items at once, e.g. after a
   * drag-and-drop reorder or a move between Morning/Afternoon.
   */
  async reorder(
    items: Array<{ id: string; position: number; period: PlannerPeriod }>
  ): Promise<PlannerItem[]> {
    try {
      const response = await httpClient.put<{ items: PlannerItem[] }>(
        "/api/planner/reorder",
        { items }
      );
      return response.items || [];
    } catch (error) {
      console.error("Failed to reorder planner items:", error);
      throw error instanceof Error
        ? error
        : new Error("Failed to reorder planner items");
    }
  }

  /**
   * Erase every item and start a fresh plan
   */
  async clearAll(): Promise<boolean> {
    try {
      await httpClient.delete("/api/planner/items");
      return true;
    } catch (error) {
      console.error("Failed to clear planner items:", error);
      return false;
    }
  }
}

// Export singleton instance
export const plannerService = new PlannerService();
