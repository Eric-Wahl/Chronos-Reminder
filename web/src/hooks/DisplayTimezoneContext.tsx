import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from "react";
import { accountService } from "@/services";
import { getDeviceTimezone, timezonesDiffer } from "@/lib/timezone";
import { useAuth } from "./useAuth";

const STORAGE_KEY = "chronos.displayTimezone";

type DisplayTimezoneMode = "account" | "device";

interface DisplayTimezoneValue {
  /** Timezone every time on the site is rendered in */
  displayTimezone: string;
  accountTimezone: string;
  deviceTimezone: string;
  /** False until the account timezone has been fetched */
  isReady: boolean;
  /** Whether the device and account timezones actually differ, i.e. whether a choice makes sense */
  canSwitch: boolean;
  /** The timezones differ and the user hasn't picked one yet: time to explain the choice */
  needsChoice: boolean;
  useDevice: boolean;
  setUseDevice: (value: boolean) => void;
  /** Keeps the provider in sync after the user changes their account timezone */
  setAccountTimezone: (timezone: string) => void;
}

const DisplayTimezoneContext = createContext<DisplayTimezoneValue | undefined>(
  undefined
);

function readStoredMode(): DisplayTimezoneMode | null {
  try {
    const stored = localStorage.getItem(STORAGE_KEY);
    return stored === "device" || stored === "account" ? stored : null;
  } catch {
    return null;
  }
}

/**
 * Which timezone the site renders times in. Display only: reminders are
 * always scheduled and delivered in the account's timezone, and the forms
 * that take a date/time keep interpreting it in the account's timezone. The
 * choice is only offered when the device's timezone actually differs from the
 * account's, and is remembered in localStorage.
 */
export function DisplayTimezoneProvider({ children }: { children: ReactNode }) {
  const { isAuthenticated } = useAuth();
  const deviceTimezone = useMemo(() => getDeviceTimezone(), []);
  const [accountTimezone, setAccountTimezone] = useState<string | null>(null);
  const [mode, setMode] = useState<DisplayTimezoneMode | null>(readStoredMode);

  useEffect(() => {
    if (!isAuthenticated) {
      setAccountTimezone(null);
      return;
    }

    let cancelled = false;
    accountService
      .getAccount()
      .then((account) => {
        if (!cancelled) setAccountTimezone(account?.timezone || "UTC");
      })
      .catch(() => {
        // Same fallback the pages used when the account couldn't be loaded
        if (!cancelled) setAccountTimezone("UTC");
      });
    return () => {
      cancelled = true;
    };
  }, [isAuthenticated]);

  const setUseDevice = useCallback((value: boolean) => {
    const next: DisplayTimezoneMode = value ? "device" : "account";
    setMode(next);
    try {
      localStorage.setItem(STORAGE_KEY, next);
    } catch {
      // Storage unavailable (private mode, blocked): the choice just won't persist
    }
  }, []);

  const value = useMemo<DisplayTimezoneValue>(() => {
    const effectiveAccount = accountTimezone ?? deviceTimezone;
    const canSwitch =
      accountTimezone !== null &&
      timezonesDiffer(accountTimezone, deviceTimezone);
    const useDevice = canSwitch && mode === "device";
    return {
      displayTimezone: useDevice ? deviceTimezone : effectiveAccount,
      accountTimezone: effectiveAccount,
      deviceTimezone,
      isReady: !isAuthenticated || accountTimezone !== null,
      canSwitch,
      needsChoice: canSwitch && mode === null,
      useDevice,
      setUseDevice,
      setAccountTimezone,
    };
  }, [accountTimezone, deviceTimezone, isAuthenticated, mode, setUseDevice]);

  return (
    <DisplayTimezoneContext.Provider value={value}>
      {children}
    </DisplayTimezoneContext.Provider>
  );
}

// eslint-disable-next-line react-refresh/only-export-components
export function useDisplayTimezone(): DisplayTimezoneValue {
  const context = useContext(DisplayTimezoneContext);
  if (!context) {
    throw new Error(
      "useDisplayTimezone must be used within a DisplayTimezoneProvider"
    );
  }
  return context;
}
