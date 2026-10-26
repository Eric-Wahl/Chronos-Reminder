/**
 * Timezone-aware date/time helpers for the reminder forms.
 *
 * The browser's local Date methods (getHours, setHours, toISOString, ...)
 * always operate in the device's timezone. Reminders are scheduled against
 * the account's configured IANA timezone, which can differ from the
 * device's, so these helpers go through Intl.DateTimeFormat instead of
 * relying on the device clock.
 */

const partsFormatterCache = new Map<string, Intl.DateTimeFormat>();

function getPartsFormatter(timeZone: string): Intl.DateTimeFormat {
  let formatter = partsFormatterCache.get(timeZone);
  if (!formatter) {
    formatter = new Intl.DateTimeFormat("en-CA", {
      timeZone,
      year: "numeric",
      month: "2-digit",
      day: "2-digit",
      hour: "2-digit",
      minute: "2-digit",
      hourCycle: "h23",
    });
    partsFormatterCache.set(timeZone, formatter);
  }
  return formatter;
}

/** Formats an absolute instant as YYYY-MM-DD / HH:mm strings in the given IANA timezone. */
export function formatPartsInTimezone(
  date: Date,
  timeZone: string
): { dateStr: string; timeStr: string } {
  const formatter = getPartsFormatter(timeZone);
  const parts = formatter.formatToParts(date).reduce((acc, part) => {
    acc[part.type] = part.value;
    return acc;
  }, {} as Record<string, string>);

  return {
    dateStr: `${parts.year}-${parts.month}-${parts.day}`,
    timeStr: `${parts.hour}:${parts.minute}`,
  };
}

/** Returns the date/time strings for "now + minutesAhead", as seen in the given timezone. */
export function getDefaultDateTime(
  timeZone: string,
  minutesAhead = 10
): { dateStr: string; timeStr: string } {
  const future = new Date(Date.now() + minutesAhead * 60 * 1000);
  return formatPartsInTimezone(future, timeZone);
}

/**
 * Checks whether a date+time (interpreted in the given timezone) is in the
 * past, by comparing wall-clock strings against "now" in that same timezone.
 * This avoids building a Date from the strings, which would otherwise be
 * interpreted in the browser's local timezone instead of the account's.
 */
export function isDateTimeInPast(
  dateStr: string,
  timeStr: string,
  timeZone: string
): boolean {
  if (!dateStr || !timeStr) return false;

  const { dateStr: nowDate, timeStr: nowTime } = formatPartsInTimezone(
    new Date(),
    timeZone
  );

  if (dateStr !== nowDate) return dateStr < nowDate;
  return timeStr <= nowTime;
}

/**
 * Parses a YYYY-MM-DD string into a Date using local calendar components.
 * Use this instead of `new Date(dateStr)` for display formatting: the native
 * parser treats date-only ISO strings as UTC midnight, which can roll over
 * to the previous/next day once re-displayed in the device's timezone.
 */
export function parseDateStrLocal(dateStr: string): Date {
  const [year, month, day] = dateStr.split("-").map(Number);
  return new Date(year, (month ?? 1) - 1, day ?? 1);
}

/**
 * Converts a wall-clock date+time (interpreted in the given IANA timezone)
 * into an absolute UTC Date instant. Used when a preview/computation needs
 * a real instant to compare against "now" or display, rather than just the
 * wall-clock strings handled by isDateTimeInPast/formatPartsInTimezone.
 */
export function zonedTimeToUtc(
  dateStr: string,
  timeStr: string,
  timeZone: string
): Date {
  const [year, month, day] = dateStr.split("-").map(Number);
  const [hour, minute] = timeStr.split(":").map(Number);

  // First guess: treat the wall-clock values as if they were UTC.
  const guess = new Date(
    Date.UTC(year, (month ?? 1) - 1, day ?? 1, hour ?? 0, minute ?? 0)
  );

  // Find what wall-clock time that UTC instant actually represents in the
  // target timezone, then correct the guess by the difference.
  const { dateStr: tzDateStr, timeStr: tzTimeStr } = formatPartsInTimezone(
    guess,
    timeZone
  );
  const [tzYear, tzMonth, tzDay] = tzDateStr.split("-").map(Number);
  const [tzHour, tzMinute] = tzTimeStr.split(":").map(Number);
  const tzAsUtc = Date.UTC(tzYear, tzMonth - 1, tzDay, tzHour, tzMinute);

  const offset = guess.getTime() - tzAsUtc;
  return new Date(guess.getTime() + offset);
}

/** The device's own IANA timezone. */
export function getDeviceTimezone(): string {
  return Intl.DateTimeFormat().resolvedOptions().timeZone;
}

function getOffsetMinutes(date: Date, timeZone: string): number {
  const { dateStr, timeStr } = formatPartsInTimezone(date, timeZone);
  const [year, month, day] = dateStr.split("-").map(Number);
  const [hour, minute] = timeStr.split(":").map(Number);
  const wallClockAsUtc = Date.UTC(year, month - 1, day, hour, minute);
  const instantToMinute = Math.floor(date.getTime() / 60000) * 60000;
  return Math.round((wallClockAsUtc - instantToMinute) / 60000);
}

/**
 * True when two timezones would show a different wall-clock time for at
 * least one of the sampled instants (now, mid-winter, mid-summer). Compares
 * offsets rather than names so aliases (Asia/Calcutta vs Asia/Kolkata) or
 * zones that currently agree (Europe/Paris vs Europe/Berlin) don't count.
 */
export function timezonesDiffer(a: string, b: string): boolean {
  if (a === b) return false;
  const year = new Date().getUTCFullYear();
  const samples = [
    new Date(),
    new Date(Date.UTC(year, 0, 1)),
    new Date(Date.UTC(year, 6, 1)),
  ];
  return samples.some((s) => getOffsetMinutes(s, a) !== getOffsetMinutes(s, b));
}

/** YYYY-MM-DD of an absolute instant as seen in the given timezone. */
export function getDateKeyInTimezone(date: Date, timeZone: string): string {
  return formatPartsInTimezone(date, timeZone).dateStr;
}

/** YYYY-MM-DD of a calendar date built from local components (new Date(y, m, d)). */
export function getLocalDateKey(date: Date): string {
  const month = String(date.getMonth() + 1).padStart(2, "0");
  const day = String(date.getDate()).padStart(2, "0");
  return `${date.getFullYear()}-${month}-${day}`;
}

/** Today's calendar date in the given timezone, as a local-component Date. */
export function getTodayAsCalendarDate(timeZone: string): Date {
  return parseDateStrLocal(getDateKeyInTimezone(new Date(), timeZone));
}

/** HH:mm (locale style) of an instant as seen in the given timezone. */
export function formatTimeInTimezone(date: Date, timeZone: string): string {
  return date.toLocaleTimeString([], {
    hour: "2-digit",
    minute: "2-digit",
    timeZone,
  });
}

/** Short "Mon D" date of an instant as seen in the given timezone. */
export function formatShortDateInTimezone(date: Date, timeZone: string): string {
  return date.toLocaleDateString([], {
    month: "short",
    day: "numeric",
    timeZone,
  });
}

/** Locale date of an instant as seen in the given timezone. */
export function formatDateInTimezone(date: Date, timeZone: string): string {
  return date.toLocaleDateString([], { timeZone });
}
