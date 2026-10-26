import { useState, useEffect } from "react";
import { X, Clock, Link2, Eye, Trash2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { useTranslation } from "react-i18next";
import { useNavigate } from "react-router-dom";
import { remindersService } from "@/services";
import type { Reminder } from "@/services";
import { getRecurrenceTypeI18nKeyFromString } from "@/lib/recurrenceUtils";
import { formatDateInTimezone, formatTimeInTimezone } from "@/lib/timezone";

interface ReminderDetailModalProps {
  isOpen: boolean;
  date: Date | null;
  reminders: Reminder[];
  /** IANA timezone the reminder times are displayed in */
  timeZone: string;
  onClose: () => void;
  onReminderDeleted?: (reminderId: string) => void;
  onAddNew?: (date: Date) => void;
}

export function ReminderDetailModal({
  isOpen,
  date,
  reminders,
  timeZone,
  onClose,
  onReminderDeleted,
  onAddNew,
}: ReminderDetailModalProps) {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const [deletingId, setDeletingId] = useState<string | null>(null);
  const [confirmDeleteId, setConfirmDeleteId] = useState<string | null>(null);
  const [displayedReminders, setDisplayedReminders] = useState(reminders);

  // Sync displayedReminders with prop changes when modal opens
  useEffect(() => {
    setDisplayedReminders(reminders);
  }, [reminders, isOpen]);

  const handleDeleteReminder = async (reminderId: string) => {
    setDeletingId(reminderId);
    try {
      const success = await remindersService.deleteReminder(reminderId);
      if (success) {
        setDisplayedReminders(
          displayedReminders.filter((r) => r.id !== reminderId)
        );
        onReminderDeleted?.(reminderId);
      }
    } catch (error) {
      console.error("Delete error:", error);
    } finally {
      setDeletingId(null);
      setConfirmDeleteId(null);
    }
  };

  if (!isOpen || !date) return null;

  const dayName = date.toLocaleDateString("en-US", { weekday: "long" });
  const dateString = date.toLocaleDateString("en-US", {
    year: "numeric",
    month: "long",
    day: "numeric",
  });

  return (
    <>
      {/* Backdrop */}
      <div
        className="fixed inset-0 bg-black/50 backdrop-blur-sm z-40"
        onClick={onClose}
      />

      {/* Modal */}
      <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
        <Card className="w-full max-w-md border-border bg-card/95 backdrop-blur overflow-hidden shadow-2xl">
          {/* Header */}
          <div className="flex items-center justify-between p-6 border-b border-border">
            <div>
              <h2 className="text-xl font-bold text-foreground">{dayName}</h2>
              <p className="text-sm text-muted-foreground">{dateString}</p>
            </div>
            <Button
              onClick={onClose}
              variant="ghost"
              size="sm"
              className="hover:bg-accent/10"
            >
              <X className="w-5 h-5" />
            </Button>
          </div>

          {/* Content */}
          <div className="p-6 max-h-96 overflow-y-auto">
            {displayedReminders.length > 0 ? (
              <div className="space-y-3">
                {displayedReminders.map((reminder) => {
                  const timeString = formatTimeInTimezone(
                    new Date(reminder.remind_at_utc),
                    timeZone
                  );

                  return (
                    <div
                      key={reminder.id}
                      className="rounded-lg border border-accent bg-white/50 dark:bg-neutral-900/50 p-4 hover:border-accent/80 hover:bg-white/70 dark:hover:bg-neutral-800/70 transition-all"
                    >
                      {/* Time */}
                      <div className="flex items-center gap-2 mb-2">
                        <Clock className="w-4 h-4 text-accent flex-shrink-0" />
                        <span className="text-sm font-semibold text-accent">
                          {timeString}
                        </span>
                      </div>

                      {/* Message */}
                      <p className="text-sm text-foreground mb-3 break-words">
                        {reminder.message}
                      </p>

                      {/* Metadata */}
                      <div className="space-y-2 text-xs text-muted-foreground">
                        {/* Recurrence */}
                        <div className="flex items-center gap-2">
                          <span className="inline-block px-2 py-1 rounded bg-accent/10 text-accent">
                            {t(
                              getRecurrenceTypeI18nKeyFromString(
                                reminder.recurrence_type
                              )
                            )}
                          </span>
                          {reminder.is_paused && (
                            <span className="inline-block px-2 py-1 rounded bg-yellow-500/10 text-yellow-600 dark:text-yellow-400">
                              Paused
                            </span>
                          )}
                        </div>

                        {/* Destinations */}
                        {reminder.destinations &&
                          reminder.destinations.length > 0 && (
                            <div className="flex items-center gap-2 flex-wrap">
                              <Link2 className="w-3 h-3 flex-shrink-0" />
                              <span>
                                {reminder.destinations.length}{" "}
                                {reminder.destinations.length === 1
                                  ? "destination"
                                  : "destinations"}
                              </span>
                            </div>
                          )}

                        {/* Created date */}
                        <div className="text-xs">
                          Created:{" "}
                          {formatDateInTimezone(
                            new Date(reminder.created_at),
                            timeZone
                          )}
                        </div>
                      </div>

                      {/* Snoozed indicator */}
                      {reminder.snoozed_at_utc && (
                        <div className="mt-3 pt-3 border-t border-accent/20">
                          <span className="inline-block px-2 py-1 rounded text-xs bg-yellow-500/20 text-yellow-600 dark:text-yellow-400">
                            Snoozed until{" "}
                            {formatTimeInTimezone(
                              new Date(reminder.snoozed_at_utc),
                              timeZone
                            )}
                          </span>
                        </div>
                      )}

                      {/* Action Buttons */}
                      <div className="mt-4 pt-4 border-t border-accent/20 flex gap-2">
                        <Button
                          size="sm"
                          variant="outline"
                          className="flex-1 text-xs border-accent/50 text-accent hover:bg-accent/10"
                          disabled={deletingId === reminder.id}
                          onClick={() => navigate(`/reminders/${reminder.id}`)}
                        >
                          <Eye className="w-3 h-3 mr-1.5" />
                          View
                        </Button>
                        <Button
                          size="sm"
                          variant="outline"
                          className="flex-1 text-xs border-red-500/50 text-red-600 dark:text-red-400 hover:bg-red-500/10"
                          onClick={() => setConfirmDeleteId(reminder.id)}
                          disabled={
                            deletingId === reminder.id ||
                            confirmDeleteId === reminder.id
                          }
                        >
                          <Trash2 className="w-3 h-3 mr-1.5" />
                          {deletingId === reminder.id
                            ? "Deleting..."
                            : "Delete"}
                        </Button>
                      </div>
                    </div>
                  );
                })}
              </div>
            ) : (
              <div className="text-center py-8">
                <Clock className="w-12 h-12 text-muted-foreground mx-auto mb-3 opacity-50" />
                <p className="text-muted-foreground">
                  {t("calendar.noReminders")}
                </p>
              </div>
            )}
          </div>

          {/* Footer */}
          <div className="border-t border-border p-4 bg-secondary/20 flex gap-2">
            {displayedReminders.length === 0 ? (
              <>
                <Button
                  onClick={onClose}
                  variant="outline"
                  className="flex-1 border-border text-foreground hover:bg-secondary/50"
                >
                  Close
                </Button>
                <Button
                  onClick={() => date && onAddNew?.(date)}
                  className="flex-1 bg-accent hover:bg-accent/90 text-accent-foreground font-semibold"
                >
                  + Add New
                </Button>
              </>
            ) : (
              <Button
                onClick={onClose}
                className="w-full bg-accent hover:bg-accent/90 text-accent-foreground font-semibold"
              >
                Close
              </Button>
            )}
          </div>
        </Card>

        {/* Confirmation Dialog */}
        {confirmDeleteId && (
          <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
            <Card className="w-full max-w-sm border-border bg-card/95 backdrop-blur overflow-hidden shadow-2xl">
              {/* Header */}
              <div className="p-6 border-b border-border">
                <h2 className="text-lg font-bold text-foreground">
                  Delete Reminder?
                </h2>
                <p className="text-sm text-muted-foreground mt-1">
                  This action cannot be undone.
                </p>
              </div>

              {/* Content */}
              <div className="p-6">
                <p className="text-sm text-foreground">
                  Are you sure you want to delete this reminder? This will
                  permanently remove it from your account.
                </p>
              </div>

              {/* Footer */}
              <div className="border-t border-border p-4 bg-secondary/20 flex gap-2">
                <Button
                  onClick={() => setConfirmDeleteId(null)}
                  variant="outline"
                  className="flex-1 border-border text-foreground hover:bg-secondary/50"
                  disabled={deletingId === confirmDeleteId}
                >
                  Cancel
                </Button>
                <Button
                  onClick={() => handleDeleteReminder(confirmDeleteId)}
                  className="flex-1 bg-red-600 hover:bg-red-700 text-white font-semibold"
                  disabled={deletingId === confirmDeleteId}
                >
                  {deletingId === confirmDeleteId ? "Deleting..." : "Delete"}
                </Button>
              </div>
            </Card>
          </div>
        )}
      </div>
    </>
  );
}
