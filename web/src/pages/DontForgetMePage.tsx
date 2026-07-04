import {
  Plus,
  Trash2,
  BellOff,
  BellRing,
  Clock,
  Pencil,
  Check,
  X,
  Send,
  CalendarClock,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
  CardDescription,
} from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Badge } from "@/components/ui/badge";
import { Header } from "@/components/common/header";
import { Footer } from "@/components/common/footer";
import { useTranslation } from "react-i18next";
import { useCallback, useEffect, useState } from "react";
import { toast } from "sonner";
import {
  dfmService,
  accountService,
  type DFMNote,
  type DFMItem,
} from "@/services";
import {
  formatPartsInTimezone,
  parseDateStrLocal,
  zonedTimeToUtc,
} from "@/lib/timezone";
import { MAX_DFM_ITEMS_PER_NOTE } from "@/lib/limits";

const RECURRENCE_OPTIONS = [
  "DAILY",
  "WEEKLY",
  "MONTHLY",
  "YEARLY",
  "WORKDAYS",
  "WEEKEND",
];

/** Formats a Date's local calendar components back into a YYYY-MM-DD string. */
function toDateStr(date: Date): string {
  const year = date.getFullYear();
  const month = String(date.getMonth() + 1).padStart(2, "0");
  const day = String(date.getDate()).padStart(2, "0");
  return `${year}-${month}-${day}`;
}

/**
 * Advances a YYYY-MM-DD calendar date by one recurrence step. This is pure
 * calendar arithmetic (no time-of-day, no timezone offset involved), so it's
 * safe regardless of the account's configured timezone.
 */
function advanceDateStr(dateStr: string, recurrence: string): string {
  const next = parseDateStrLocal(dateStr);
  switch (recurrence) {
    case "WEEKLY":
      next.setDate(next.getDate() + 7);
      break;
    case "MONTHLY":
      next.setMonth(next.getMonth() + 1);
      break;
    case "YEARLY":
      next.setFullYear(next.getFullYear() + 1);
      break;
    case "WORKDAYS":
      do {
        next.setDate(next.getDate() + 1);
      } while (next.getDay() === 0 || next.getDay() === 6);
      break;
    case "WEEKEND":
      do {
        next.setDate(next.getDate() + 1);
      } while (next.getDay() !== 0 && next.getDay() !== 6);
      break;
    default:
      next.setDate(next.getDate() + 1);
  }
  return toDateStr(next);
}

/**
 * Client-side preview of the next delivery, based on the selected start date,
 * time and recurrence, interpreted in the account's configured timezone. The
 * backend does the authoritative computation; this mirrors it in the browser
 * so the user sees the result live.
 */
function computeNextDelivery(
  dateStr: string,
  timeStr: string,
  recurrence: string,
  timeZone: string,
): Date | null {
  const [hours, minutes] = timeStr.split(":").map(Number);
  if (Number.isNaN(hours) || Number.isNaN(minutes)) return null;

  let currentDateStr = dateStr || formatPartsInTimezone(new Date(), timeZone).dateStr;
  if (!/^\d{4}-\d{2}-\d{2}$/.test(currentDateStr)) return null;

  const now = new Date();
  let candidate = zonedTimeToUtc(currentDateStr, timeStr, timeZone);
  let guard = 0;
  while (candidate <= now && guard++ < 1000) {
    currentDateStr = advanceDateStr(currentDateStr, recurrence);
    candidate = zonedTimeToUtc(currentDateStr, timeStr, timeZone);
  }
  return candidate;
}

export function DontForgetMePage() {
  const { t } = useTranslation();
  const [note, setNote] = useState<DFMNote | null>(null);
  const [isLoading, setIsLoading] = useState(true);

  const [newItemContent, setNewItemContent] = useState("");
  const [isAddingItem, setIsAddingItem] = useState(false);

  const [editingItemId, setEditingItemId] = useState<string | null>(null);
  const [editingContent, setEditingContent] = useState("");

  const [reminderRecurrence, setReminderRecurrence] = useState("DAILY");
  const [reminderTime, setReminderTime] = useState("09:00");
  const [reminderDate, setReminderDate] = useState("");
  const [isSendingNow, setIsSendingNow] = useState(false);
  const [destDiscord, setDestDiscord] = useState(true);
  const [destEmail, setDestEmail] = useState(false);
  const [hasDiscord, setHasDiscord] = useState(true);
  const [hasEmail, setHasEmail] = useState(true);
  const [isSavingReminder, setIsSavingReminder] = useState(false);
  // Default to the browser's timezone until the account's configured
  // timezone has loaded.
  const [timeZone, setTimeZone] = useState<string>(
    () => Intl.DateTimeFormat().resolvedOptions().timeZone || "UTC",
  );

  const refreshNote = useCallback(async () => {
    const [fetchedNote, fetchedAccount] = await Promise.all([
      dfmService.getNote(),
      accountService.getAccount(),
    ]);
    const accountTimeZone =
      fetchedAccount?.timezone ||
      Intl.DateTimeFormat().resolvedOptions().timeZone ||
      "UTC";
    if (fetchedAccount?.timezone) {
      setTimeZone(fetchedAccount.timezone);
    }
    if (fetchedNote) {
      setNote(fetchedNote);
      if (fetchedNote.has_reminder) {
        if (fetchedNote.recurrence_type !== "ONCE") {
          setReminderRecurrence(fetchedNote.recurrence_type);
        }
        if (fetchedNote.remind_at_utc) {
          const { timeStr } = formatPartsInTimezone(
            new Date(fetchedNote.remind_at_utc),
            accountTimeZone,
          );
          setReminderTime(timeStr);
        }
        setDestDiscord(fetchedNote.destinations.includes("discord_dm"));
        setDestEmail(fetchedNote.destinations.includes("email"));
      }
    }
    if (fetchedAccount) {
      const identities = fetchedAccount.identities ?? [];
      const discordLinked = identities.some((i) => i.provider === "discord");
      const emailLinked = !!fetchedAccount.email;
      setHasDiscord(discordLinked);
      setHasEmail(emailLinked);
      // Default to a destination the user can actually receive
      if (!fetchedNote?.has_reminder) {
        setDestDiscord(discordLinked);
        setDestEmail(!discordLinked && emailLinked);
      }
    }
  }, []);

  useEffect(() => {
    const fetchData = async () => {
      setIsLoading(true);
      await refreshNote();
      setIsLoading(false);
    };
    fetchData();
  }, [refreshNote]);

  const handleAddItem = async () => {
    const content = newItemContent.trim();
    if (!content) return;

    if ((note?.items.length ?? 0) >= MAX_DFM_ITEMS_PER_NOTE) {
      toast.error(t("dfm.itemLimitReachedTitle"), {
        description: t("dfm.itemLimitReachedDesc", {
          max: MAX_DFM_ITEMS_PER_NOTE,
        }),
      });
      return;
    }

    setIsAddingItem(true);
    try {
      const item = await dfmService.addItem(content);
      if (item) {
        setNewItemContent("");
        setNote((prev) =>
          prev ? { ...prev, items: [...prev.items, item] } : prev,
        );
        toast.success(t("dfm.itemAdded"));
      } else {
        toast.error(t("dfm.itemAddFailed"));
      }
    } catch (err) {
      const errorMessage =
        err instanceof Error ? err.message : t("dfm.itemAddFailed");
      toast.error(errorMessage);
    } finally {
      setIsAddingItem(false);
    }
  };

  const handleToggleItem = async (item: DFMItem) => {
    const updated = await dfmService.updateItem(item.id, {
      checked: !item.checked,
    });
    if (updated) {
      setNote((prev) =>
        prev
          ? {
              ...prev,
              items: prev.items.map((i) => (i.id === item.id ? updated : i)),
            }
          : prev,
      );
    } else {
      toast.error(t("dfm.itemUpdateFailed"));
    }
  };

  const handleStartEdit = (item: DFMItem) => {
    setEditingItemId(item.id);
    setEditingContent(item.content);
  };

  const handleSaveEdit = async (item: DFMItem) => {
    const content = editingContent.trim();
    if (!content) return;

    const updated = await dfmService.updateItem(item.id, { content });
    if (updated) {
      setNote((prev) =>
        prev
          ? {
              ...prev,
              items: prev.items.map((i) => (i.id === item.id ? updated : i)),
            }
          : prev,
      );
      setEditingItemId(null);
      toast.success(t("dfm.itemUpdated"));
    } else {
      toast.error(t("dfm.itemUpdateFailed"));
    }
  };

  const handleDeleteItem = async (item: DFMItem) => {
    const success = await dfmService.deleteItem(item.id);
    if (success) {
      setNote((prev) =>
        prev
          ? { ...prev, items: prev.items.filter((i) => i.id !== item.id) }
          : prev,
      );
      toast.success(t("dfm.itemDeleted"));
    } else {
      toast.error(t("dfm.itemDeleteFailed"));
    }
  };

  const handleSaveReminder = async () => {
    const destinations: Array<"discord_dm" | "email"> = [
      ...(destDiscord ? (["discord_dm"] as const) : []),
      ...(destEmail ? (["email"] as const) : []),
    ];
    if (destinations.length === 0) {
      toast.error(t("dfm.destinationRequired"));
      return;
    }

    setIsSavingReminder(true);
    const updated = await dfmService.setReminder({
      ...(reminderDate ? { date: reminderDate } : {}),
      time: reminderTime,
      recurrence: reminderRecurrence,
      destinations,
    });
    setIsSavingReminder(false);

    if (updated) {
      setNote((prev) => (prev ? { ...prev, ...updated } : updated));
      toast.success(t("dfm.reminderSaved"));
    } else {
      toast.error(t("dfm.reminderSaveFailed"));
    }
  };

  const handleSendNow = async () => {
    setIsSendingNow(true);
    const success = await dfmService.sendNow();
    setIsSendingNow(false);

    if (success) {
      toast.success(t("dfm.sendNowSuccess"));
    } else {
      toast.error(t("dfm.sendNowFailed"));
    }
  };

  const handleRemoveReminder = async () => {
    const updated = await dfmService.removeReminder();
    if (updated) {
      setNote((prev) => (prev ? { ...prev, ...updated } : updated));
      toast.success(t("dfm.reminderRemoved"));
    } else {
      toast.error(t("dfm.reminderRemoveFailed"));
    }
  };

  const checkedCount = note?.items.filter((i) => i.checked).length ?? 0;
  const itemLimitReached =
    (note?.items.length ?? 0) >= MAX_DFM_ITEMS_PER_NOTE;

  const nextDeliveryPreview = computeNextDelivery(
    reminderDate,
    reminderTime,
    reminderRecurrence,
    timeZone,
  );

  return (
    <div className="min-h-screen bg-background-main dark:bg-background-main">
      <Header />

      <main className="max-w-5xl mx-auto px-4 sm:px-6 lg:px-8 py-12 pt-24">
        {/* Title Section */}
        <div className="mb-12">
          <h2 className="text-3xl sm:text-4xl font-bold text-foreground">
            {t("dfm.title")}
          </h2>
          <p className="text-muted-foreground text-base sm:text-lg mt-2">
            {t("dfm.subtitle")}
          </p>
        </div>

        {isLoading ? (
          <Card className="border-border bg-card/95 backdrop-blur text-center py-12">
            <Clock className="w-12 h-12 text-muted-foreground mx-auto mb-4 animate-spin" />
            <p className="text-muted-foreground">{t("dfm.loading")}</p>
          </Card>
        ) : (
          <div className="gap-8">
            {/* Note Section */}
            <div className="space-y-8">
              <Card className="border-border bg-card/95 backdrop-blur">
                <CardHeader>
                  <CardTitle className="flex items-center justify-between">
                    <span>{t("dfm.noteTitle")}</span>
                    {note && note.items.length > 0 && (
                      <Badge variant="secondary">
                        {t("dfm.itemsProgress", {
                          checked: checkedCount,
                          total: note.items.length,
                        })}
                      </Badge>
                    )}
                  </CardTitle>
                  <CardDescription>{t("dfm.noteDescription")}</CardDescription>
                </CardHeader>
                <CardContent className="space-y-4">
                  {/* Add item input */}
                  <div className="flex gap-2">
                    <Input
                      value={newItemContent}
                      onChange={(e) => setNewItemContent(e.target.value)}
                      onKeyDown={(e) => {
                        if (e.key === "Enter") handleAddItem();
                      }}
                      placeholder={t("dfm.addItemPlaceholder")}
                      maxLength={500}
                      disabled={itemLimitReached}
                    />
                    <Button
                      onClick={handleAddItem}
                      disabled={
                        isAddingItem ||
                        !newItemContent.trim() ||
                        itemLimitReached
                      }
                      className="bg-accent hover:bg-accent/90 text-accent-foreground gap-2"
                    >
                      <Plus className="w-4 h-4" />
                      {t("dfm.addItem")}
                    </Button>
                  </div>
                  {itemLimitReached && (
                    <p className="text-xs text-red-500">
                      {t("dfm.itemLimitReachedDesc", {
                        max: MAX_DFM_ITEMS_PER_NOTE,
                      })}
                    </p>
                  )}

                  {/* Items list */}
                  {note && note.items.length === 0 ? (
                    <p className="text-muted-foreground text-sm py-6 text-center">
                      {t("dfm.emptyNote")}
                    </p>
                  ) : (
                    <ul className="space-y-2">
                      {note?.items.map((item) => (
                        <li
                          key={item.id}
                          className="flex items-center gap-3 rounded-md border border-border px-3 py-2 group"
                        >
                          <input
                            type="checkbox"
                            checked={item.checked}
                            onChange={() => handleToggleItem(item)}
                            className="w-4 h-4 accent-accent cursor-pointer shrink-0"
                            aria-label={t("dfm.toggleItem")}
                          />

                          {editingItemId === item.id ? (
                            <>
                              <Input
                                value={editingContent}
                                onChange={(e) =>
                                  setEditingContent(e.target.value)
                                }
                                onKeyDown={(e) => {
                                  if (e.key === "Enter") handleSaveEdit(item);
                                  if (e.key === "Escape")
                                    setEditingItemId(null);
                                }}
                                maxLength={500}
                                autoFocus
                                className="h-8"
                              />
                              <Button
                                size="icon"
                                variant="ghost"
                                className="h-8 w-8 shrink-0"
                                onClick={() => handleSaveEdit(item)}
                                aria-label={t("dfm.saveItem")}
                              >
                                <Check className="w-4 h-4" />
                              </Button>
                              <Button
                                size="icon"
                                variant="ghost"
                                className="h-8 w-8 shrink-0"
                                onClick={() => setEditingItemId(null)}
                                aria-label={t("dfm.cancelEdit")}
                              >
                                <X className="w-4 h-4" />
                              </Button>
                            </>
                          ) : (
                            <>
                              <span
                                className={`flex-1 text-sm ${
                                  item.checked
                                    ? "line-through text-muted-foreground"
                                    : "text-foreground"
                                }`}
                              >
                                {item.content}
                              </span>
                              <Button
                                size="icon"
                                variant="ghost"
                                className="h-8 w-8 shrink-0 md:opacity-0 md:group-hover:opacity-100 transition-opacity"
                                onClick={() => handleStartEdit(item)}
                                aria-label={t("dfm.editItem")}
                              >
                                <Pencil className="w-4 h-4" />
                              </Button>
                              <Button
                                size="icon"
                                variant="ghost"
                                className="h-8 w-8 shrink-0 md:opacity-0 md:group-hover:opacity-100 transition-opacity text-red-500 hover:text-red-600"
                                onClick={() => handleDeleteItem(item)}
                                aria-label={t("dfm.deleteItem")}
                              >
                                <Trash2 className="w-4 h-4" />
                              </Button>
                            </>
                          )}
                        </li>
                      ))}
                    </ul>
                  )}
                </CardContent>
              </Card>

              <Card className="border-border bg-card/95 backdrop-blur">
                <CardHeader>
                  <CardTitle className="flex items-center gap-2">
                    {note?.has_reminder ? (
                      <BellRing className="w-5 h-5 text-accent" />
                    ) : (
                      <BellOff className="w-5 h-5" />
                    )}
                    {t("dfm.reminderTitle")}
                  </CardTitle>
                  <CardDescription>
                    {t("dfm.reminderDescription")}
                  </CardDescription>
                </CardHeader>
                <CardContent className="space-y-4">
                  {note?.has_reminder && note.next_fire_utc && (
                    <div className="rounded-md border border-accent/40 bg-accent/10 px-3 py-2 text-sm">
                      <p className="text-foreground font-medium">
                        {t(`recurrence.${note.recurrence_type.toLowerCase()}`, {
                          defaultValue: note.recurrence_type,
                        })}
                      </p>
                      <p className="text-muted-foreground mt-1">
                        {t("dfm.nextDelivery")}:{" "}
                        {new Date(note.next_fire_utc).toLocaleString(undefined, {
                          timeZone,
                        })}
                      </p>
                      <p className="text-muted-foreground mt-1">
                        {t("dfm.destinationLabel")}:{" "}
                        {note.destinations
                          .map((d) =>
                            d === "email"
                              ? t("dfm.destinationEmail")
                              : t("dfm.destinationDiscord"),
                          )
                          .join(", ")}
                      </p>
                    </div>
                  )}

                  <div className="space-y-2">
                    <label className="text-sm text-muted-foreground">
                      {t("dfm.recurrenceLabel")}
                    </label>
                    <Select
                      value={reminderRecurrence}
                      onValueChange={setReminderRecurrence}
                    >
                      <SelectTrigger className="w-full">
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        {RECURRENCE_OPTIONS.map((option) => (
                          <SelectItem key={option} value={option}>
                            {t(`recurrence.${option.toLowerCase()}`, {
                              defaultValue: option,
                            })}
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                  </div>

                  <div className="space-y-2">
                    <label className="text-sm text-muted-foreground">
                      {t("dfm.startDateLabel")}
                    </label>
                    <Input
                      type="date"
                      value={reminderDate}
                      min={formatPartsInTimezone(new Date(), timeZone).dateStr}
                      onChange={(e) => setReminderDate(e.target.value)}
                    />
                    <p className="text-xs text-muted-foreground">
                      {t("dfm.startDateHint")}
                    </p>
                  </div>

                  <div className="space-y-2">
                    <label className="text-sm text-muted-foreground">
                      {t("dfm.timeLabel")}
                    </label>
                    <Input
                      type="time"
                      value={reminderTime}
                      onChange={(e) => setReminderTime(e.target.value)}
                    />
                  </div>

                  <div className="space-y-2">
                    <label className="text-sm text-muted-foreground">
                      {t("dfm.destinationLabel")}
                    </label>
                    <div className="space-y-2">
                      <label
                        className={`flex items-center gap-2 rounded-md border border-border px-3 py-2 text-sm ${
                          hasDiscord
                            ? "cursor-pointer"
                            : "opacity-50 cursor-not-allowed"
                        }`}
                      >
                        <input
                          type="checkbox"
                          checked={destDiscord}
                          disabled={!hasDiscord}
                          onChange={(e) => setDestDiscord(e.target.checked)}
                          className="w-4 h-4 accent-accent"
                        />
                        <span className="text-foreground">
                          {t("dfm.destinationDiscord")}
                          {!hasDiscord
                            ? ` (${t("dfm.destinationNotLinked")})`
                            : ""}
                        </span>
                      </label>
                      <label
                        className={`flex items-center gap-2 rounded-md border border-border px-3 py-2 text-sm ${
                          hasEmail
                            ? "cursor-pointer"
                            : "opacity-50 cursor-not-allowed"
                        }`}
                      >
                        <input
                          type="checkbox"
                          checked={destEmail}
                          disabled={!hasEmail}
                          onChange={(e) => setDestEmail(e.target.checked)}
                          className="w-4 h-4 accent-accent"
                        />
                        <span className="text-foreground">
                          {t("dfm.destinationEmail")}
                          {!hasEmail
                            ? ` (${t("dfm.destinationNotLinked")})`
                            : ""}
                        </span>
                      </label>
                    </div>
                    <p className="text-xs text-muted-foreground">
                      {t("dfm.destinationHint")}
                    </p>
                  </div>

                  {nextDeliveryPreview && (
                    <div className="flex items-start gap-2 rounded-md border border-border bg-muted/30 px-3 py-2 text-sm">
                      <CalendarClock className="w-4 h-4 mt-0.5 shrink-0 text-accent" />
                      <p className="text-muted-foreground">
                        {t("dfm.nextPreview")}{" "}
                        <span className="text-foreground font-medium">
                          {nextDeliveryPreview.toLocaleString(undefined, {
                            timeZone,
                            weekday: "long",
                            year: "numeric",
                            month: "short",
                            day: "numeric",
                            hour: "2-digit",
                            minute: "2-digit",
                          })}
                        </span>
                      </p>
                    </div>
                  )}

                  <Button
                    onClick={handleSaveReminder}
                    disabled={isSavingReminder}
                    className="w-full bg-accent hover:bg-accent/90 text-accent-foreground"
                  >
                    {note?.has_reminder
                      ? t("dfm.updateReminder")
                      : t("dfm.setReminder")}
                  </Button>

                  {note?.has_reminder && (
                    <Button
                      onClick={handleRemoveReminder}
                      variant="outline"
                      className="w-full text-red-500 hover:text-red-600"
                    >
                      {t("dfm.removeReminder")}
                    </Button>
                  )}

                  <div className="border-t border-border pt-4">
                    <Button
                      onClick={handleSendNow}
                      disabled={
                        isSendingNow || !note || note.items.length === 0
                      }
                      variant="outline"
                      className="w-full gap-2"
                    >
                      <Send className="w-4 h-4" />
                      {t("dfm.sendNow")}
                    </Button>
                    <p className="text-xs text-muted-foreground mt-2 text-center">
                      {t("dfm.sendNowHint")}
                    </p>
                  </div>
                </CardContent>
              </Card>
            </div>
          </div>
        )}
      </main>

      <Footer />
    </div>
  );
}
