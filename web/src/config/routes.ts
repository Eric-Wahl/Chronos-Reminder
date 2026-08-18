/**
 * Central routing configuration for the application
 * Each route defines:
 * - path: the URL path
 * - requiresAuth: whether authentication is required
 * - name: human-readable name for the route
 * - showInNav: whether to show in navigation (optional)
 * - group: menu group name for dropdown menus
 * - submenu: whether this is a submenu item
 */

import { discordService } from "@/services";
import { URLS } from "./constants";

export interface Route {
  path: string;
  requiresAuth: boolean;
  name: string;
  showInNav?: boolean;
  group?: string;
  submenu?: boolean;
  external?: boolean;
}

export interface MenuGroup {
  name: string;
  label: string;
  requiresAuth: boolean;
  items: Route[];
  showcase?: {
    title?: string;
    items: Array<{
      icon: string; // emoji or icon name
      label?: string;
    }>;
  };
}

export const ROUTES = {
  HOME: {
    path: "/",
    requiresAuth: false,
    name: "home",
    showInNav: true,
  } as Route,

  // Reminders group
  REMINDERS: {
    path: "/reminders",
    requiresAuth: true,
    name: "myReminders",
    showInNav: true,
    group: "reminders",
  } as Route,

  REMINDERS_CREATE: {
    path: "/reminders/create",
    requiresAuth: true,
    name: "createReminder",
    submenu: true,
    group: "reminders",
  } as Route,

  REMINDER_DETAILS: {
    path: "/reminders/:reminderId",
    requiresAuth: true,
    name: "reminderDetails",
    showInNav: false,
  } as Route,

  DONT_FORGET_ME: {
    path: "/dont-forget-me",
    requiresAuth: true,
    name: "dontForgetMe",
    submenu: true,
    group: "reminders",
  } as Route,

  DAY_PLANNER: {
    path: "/day-planner",
    requiresAuth: true,
    name: "dayPlanner",
    submenu: true,
    group: "reminders",
  } as Route,

  // Resources group
  CHANGELOG: {
    path: "/changelog",
    requiresAuth: false,
    name: "changelog",
    submenu: true,
    group: "resources",
  } as Route,

  SELFHOST: {
    path: "/selfhost",
    requiresAuth: false,
    name: "selfHost",
    submenu: true,
    group: "resources",
  } as Route,

  // Help group
  CONTACT: {
    path: "/contact",
    requiresAuth: false,
    name: "contact",
    submenu: true,
    group: "help",
  } as Route,

  TERMS: {
    path: "/terms",
    requiresAuth: false,
    name: "terms",
    showInNav: false,
  } as Route,

  BOT_HELP: {
    path: URLS.docs,
    requiresAuth: false,
    name: "documentation",
    submenu: true,
    group: "documentation",
    external: true,
  } as Route,

  STATUS: {
    path: URLS.status,
    requiresAuth: false,
    name: "status",
    submenu: true,
    group: "help",
    external: true,
  } as Route,

  GITHUB: {
    path: URLS.github,
    requiresAuth: false,
    name: "github",
    submenu: true,
    group: "",
    external: true,
  } as Route,

  INVITE_BOT: {
    path: discordService.getBotInviteUrl(),
    requiresAuth: false,
    name: "inviteBot",
    submenu: true,
    group: "settings",
    external: true,
  } as Route,

  SUPPORT_SERVER: {
    path: URLS.discordInvite,
    requiresAuth: false,
    name: "supportServer",
    submenu: true,
    group: "help",
    external: true,
  } as Route,

  // Account (Settings group)
  ACCOUNT: {
    path: "/account",
    requiresAuth: true,
    name: "myAccount",
    submenu: true,
    group: "settings",
  } as Route,

  API_KEYS: {
    path: "/api-keys",
    requiresAuth: true,
    name: "apiKeys",
    submenu: true,
    group: "settings",
  } as Route,

  LOGIN: {
    path: "/login",
    requiresAuth: false,
    name: "signIn",
    showInNav: false,
  } as Route,

  AUTH_CALLBACK_DISCORD: {
    path: "/auth/callback/discord",
    requiresAuth: false,
    name: "Discord OAuth Callback",
    showInNav: false,
  } as Route,

  VERIFY_EMAIL: {
    path: "/verify",
    requiresAuth: false,
    name: "verifyEmail",
    showInNav: false,
  } as Route,

  FORGOT_PASSWORD: {
    path: "/forgot-password",
    requiresAuth: false,
    name: "forgotPassword",
    showInNav: false,
  } as Route,

  RESET_PASSWORD: {
    path: "/reset-password",
    requiresAuth: false,
    name: "resetPassword",
    showInNav: false,
  } as Route,
} as const;

// Menu groups configuration
export const MENU_GROUPS: MenuGroup[] = [
  {
    name: "reminders",
    label: "myReminders",
    requiresAuth: true,
    items: [
      ROUTES.REMINDERS,
      ROUTES.REMINDERS_CREATE,
      ROUTES.DONT_FORGET_ME,
      ROUTES.DAY_PLANNER,
    ],
  },
  {
    name: "resources",
    label: "resources",
    requiresAuth: false,
    items: [ROUTES.CHANGELOG, ROUTES.GITHUB, ROUTES.INVITE_BOT],
  },
  {
    name: "documentation",
    label: "documentation",
    requiresAuth: false,
    items: [ROUTES.SELFHOST, ROUTES.BOT_HELP],
  },
  {
    name: "help",
    label: "help",
    requiresAuth: false,
    items: [ROUTES.CONTACT, ROUTES.SUPPORT_SERVER, ROUTES.STATUS],
  },
  {
    name: "settings",
    label: "settings",
    requiresAuth: true,
    items: [ROUTES.ACCOUNT, ROUTES.API_KEYS],
  },
];

export const ROUTES_ARRAY: Route[] = Object.values(ROUTES);
