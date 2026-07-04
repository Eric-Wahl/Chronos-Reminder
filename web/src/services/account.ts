import { httpClient } from "./http";
import type { Account, AccountResponse, ApiResponse } from "./types";

/**
 * Account Service
 * Handles all account-related API calls
 */
class AccountService {
  /**
   * Normalize account data from API response
   * Ensures timezone is a string, not an object
   */
  private normalizeAccount(
    account: Record<string, unknown> & Partial<Account>
  ): Account {
    // Handle timezone field - it might be an object or a string
    let timezone = "UTC";
    if (account.timezone) {
      if (typeof account.timezone === "string") {
        timezone = account.timezone;
      } else if (
        typeof account.timezone === "object" &&
        account.timezone !== null
      ) {
        // If it's an object like {id, name, gmt_offset, iana_location}, use the name or iana_location
        const tzObj = account.timezone as Record<string, unknown>;
        timezone =
          (tzObj.iana_location as string) || (tzObj.name as string) || "UTC";
      }
    }

    const preferences =
      account.preferences && typeof account.preferences === "object"
        ? (account.preferences as unknown as Record<string, unknown>)
        : {};

    return {
      id: String(account.id || ""),
      email: String(account.email || ""),
      username: String(account.username || ""),
      email_verified: Boolean(account.email_verified),
      timezone,
      created_at: String(account.created_at || ""),
      identities: Array.isArray(account.identities) ? account.identities : [],
      preferences: {
        discord_send_image: preferences.discord_send_image !== false,
      },
    };
  }

  /**
   * Fetch the authenticated user's account information
   */
  async getAccount(): Promise<Account | null> {
    try {
      const response = await httpClient.get<ApiResponse<AccountResponse>>(
        "/api/account"
      );
      const account = (response.data || response) as Account;
      return this.normalizeAccount(
        account as Record<string, unknown> & Partial<Account>
      );
    } catch (error) {
      console.error("Failed to fetch account:", error);
      return null;
    }
  }

  /**
   * Update app identity password
   */
  async updateAppIdentityPassword(
    currentPassword: string,
    newPassword: string
  ): Promise<void> {
    try {
      await httpClient.post<ApiResponse<{ message: string }>>(
        "/api/account/identity/app/change-password",
        {
          current_password: currentPassword,
          new_password: newPassword,
        }
      );
    } catch (error) {
      if (error instanceof Error) {
        throw error;
      }
      throw new Error("Failed to update password");
    }
  }

  /**
   * Update account timezone
   */
  async updateTimezone(timezone: string): Promise<void> {
    try {
      await httpClient.put<ApiResponse<{ message: string }>>(
        "/api/account/timezone",
        {
          timezone,
        }
      );
    } catch (error) {
      if (error instanceof Error) {
        throw error;
      }
      throw new Error("Failed to update timezone");
    }
  }

  /**
   * Update the "send reminder image" Discord preference. Only meaningful
   * for accounts with a linked Discord identity.
   */
  async updateDiscordSendImagePreference(enabled: boolean): Promise<void> {
    try {
      await httpClient.put<ApiResponse<{ message: string }>>(
        "/api/account/preferences",
        { discord_send_image: enabled }
      );
    } catch (error) {
      if (error instanceof Error) {
        throw error;
      }
      throw new Error("Failed to update preference");
    }
  }

  /**
   * Update app identity username
   */
  async updateAppIdentityUsername(username: string): Promise<void> {
    try {
      await httpClient.put<ApiResponse<{ message: string }>>(
        "/api/account/identity/app/username",
        { username }
      );
    } catch (error) {
      if (error instanceof Error) {
        throw error;
      }
      throw new Error("Failed to update username");
    }
  }

  /**
   * Update app identity email
   */
  async updateAppIdentityEmail(email: string): Promise<void> {
    try {
      await httpClient.put<ApiResponse<{ message: string }>>(
        "/api/account/identity/app/email",
        { email }
      );
    } catch (error) {
      if (error instanceof Error) {
        throw error;
      }
      throw new Error("Failed to update email");
    }
  }

  /**
   * Add an email/password (app) identity to the current account.
   * Used by Discord-first (or mobile-first) accounts to enable web/mobile login.
   */
  async addAppIdentity(
    email: string,
    username: string,
    password: string
  ): Promise<void> {
    try {
      await httpClient.post<ApiResponse<{ message: string }>>(
        "/api/account/identity/app",
        { email, username, password }
      );
    } catch (error) {
      if (error instanceof Error) {
        throw error;
      }
      throw new Error("Failed to add email/password login");
    }
  }

  /**
   * Delete the current account
   */
  async deleteAccount(): Promise<void> {
    try {
      await httpClient.delete<ApiResponse<{ message: string }>>("/api/account");
    } catch (error) {
      if (error instanceof Error) {
        throw error;
      }
      throw new Error("Failed to delete account");
    }
  }
}

// Export singleton instance
export const accountService = new AccountService();
