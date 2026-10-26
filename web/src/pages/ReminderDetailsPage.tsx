import { useState, useEffect } from "react";
import {
  ChevronLeft,
  Edit2,
  Save,
  X,
  Calendar,
  Clock,
  MessageSquare,
  Zap,
  Pause,
  Play,
  Trash2,
  Copy,
  CheckCircle2,
  Settings,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Header } from "@/components/common/header";
import { DisplayTimezoneNotice } from "@/components/DisplayTimezoneNotice";
import { DeleteConfirmModal } from "@/components/DeleteConfirmModal";
import {
  DestinationPicker,
  type ReminderDestination as PickerDestination,
} from "@/components/reminder-wizard/DestinationPicker";
import { useNavigate, useParams } from "react-router-dom";
import { useTranslation } from "react-i18next";
import { toast } from "sonner";
import { getRecurrenceTypeI18nKeyFromString } from "@/lib/recurrenceUtils";
import {
  formatDateInTimezone,
  formatPartsInTimezone,
  isDateTimeInPast,
  parseDateStrLocal,
} from "@/lib/timezone";
import { useDisplayTimezone } from "@/hooks/useDisplayTimezone";
import type { Reminder } from "@/services";
import { remindersService } from "@/services/reminders";
import { accountService } from "@/services/account";
import { Footer } from "@/components/common/footer";

interface EditableReminderData {
  message: string;
  date: string; // Store as YYYY-MM-DD string
  time: string; // Store as HH:mm string
  recurrence: string; // Store as uppercase string (e.g., "DAILY")
  destinations: PickerDestination[];
}

export function ReminderDetailsPage() {
  const navigate = useNavigate();
  const { reminderId } = useParams();
  const { t } = useTranslation();

  const [reminder, setReminder] = useState<Reminder | null>(null);
  const [userTimezone, setUserTimezone] = useState<string>("UTC");
  const { displayTimezone } = useDisplayTimezone();
  const [isLoading, setIsLoading] = useState(true);
  const [isEditing, setIsEditing] = useState(false);
  const [isPaused, setIsPaused] = useState(false);
  const [isDeleting, setIsDeleting] = useState(false);
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false);
  const [editData, setEditData] = useState<EditableReminderData>({
    message: "",
    date: "",
    time: "10:00",
    recurrence: "ONCE",
    destinations: [],
  });

  // Load reminder data on mount
  useEffect(() => {
    if (!reminderId) {
      navigate("/home");
      return;
    }

    const loadReminder = async () => {
      try {
        // Fetch user account to get timezone
        const account = await accountService.getAccount();
        if (account) {
          setUserTimezone(account.timezone);
        }

        const data = await remindersService.getReminder(reminderId);
        if (!data) {
          toast.error("Reminder not found");
          navigate("/home");
          return;
        }

        setReminder(data);
        setIsPaused(data.is_paused);

        const convertToPickerDestinations = (
          destinations: typeof data.destinations
        ): PickerDestination[] => {
          return (destinations || []).map((d) => ({
            type: d.type,
            metadata: d.metadata,
          }));
        };

        // Convert UTC time to user's timezone for display
        const utcDate = new Date(data.remind_at_utc);
        const timezone = account?.timezone || "UTC";

        // Get the time in the user's timezone using Intl API
        const formatter = new Intl.DateTimeFormat("en-US", {
          timeZone: timezone,
          year: "numeric",
          month: "2-digit",
          day: "2-digit",
          hour: "2-digit",
          minute: "2-digit",
          second: "2-digit",
          hour12: false,
        });

        const parts = formatter.formatToParts(utcDate);
        const timeParts = parts.reduce((acc, part) => {
          acc[part.type] = part.value;
          return acc;
        }, {} as Record<string, string>);

        const userTime = `${timeParts.hour}:${timeParts.minute}`;
        const userDateString = `${timeParts.year}-${timeParts.month}-${timeParts.day}`;

        setEditData({
          message: data.message,
          date: userDateString,
          time: userTime,
          recurrence: data.recurrence_type,
          destinations: convertToPickerDestinations(data.destinations),
        });
      } catch (error) {
        console.error("Failed to load reminder:", error);
        toast.error("Failed to load reminder");
        navigate("/home");
      } finally {
        setIsLoading(false);
      }
    };

    loadReminder();
  }, [reminderId, navigate]);

  const handleEditToggle = () => {
    if (!isEditing && reminder) {
      // Entering edit mode - initialize editData with current reminder data
      const convertToPickerDestinations = (
        destinations: typeof reminder.destinations
      ): PickerDestination[] => {
        return (destinations || []).map((d) => ({
          type: d.type,
          metadata: d.metadata,
        }));
      };

      // Convert UTC time to user's timezone for display
      const utcDate = new Date(reminder.remind_at_utc);

      // Get the time in the user's timezone using Intl API
      const formatter = new Intl.DateTimeFormat("en-US", {
        timeZone: userTimezone,
        year: "numeric",
        month: "2-digit",
        day: "2-digit",
        hour: "2-digit",
        minute: "2-digit",
        second: "2-digit",
        hour12: false,
      });

      const parts = formatter.formatToParts(utcDate);
      const timeParts = parts.reduce((acc, part) => {
        acc[part.type] = part.value;
        return acc;
      }, {} as Record<string, string>);

      const userTime = `${timeParts.hour}:${timeParts.minute}`;
      const userDateString = `${timeParts.year}-${timeParts.month}-${timeParts.day}`;

      setEditData({
        message: reminder.message,
        date: userDateString,
        time: userTime,
        recurrence: reminder.recurrence_type,
        destinations: convertToPickerDestinations(reminder.destinations),
      });
    }
    setIsEditing(!isEditing);
  };

  const handleSave = async () => {
    if (!reminder || !reminderId) return;

    if (
      editData.date &&
      isDateTimeInPast(editData.date, editData.time, userTimezone)
    ) {
      toast.error(t("reminderDetails.dateTimeInPast"));
      return;
    }

    try {
      await remindersService.updateReminder(reminderId, {
        message: editData.message,
        date: editData.date,
        time: editData.time,
        recurrence: editData.recurrence,
        destinations: editData.destinations.map((d) => ({
          type: d.type as "discord_dm" | "discord_channel" | "webhook" | "email" | "android_push",
          metadata: d.metadata,
        })),
      });

      toast.success(t("reminderDetails.updatedSuccessfully"));
      setIsEditing(false);

      // Reload reminder data
      const updated = await remindersService.getReminder(reminderId);
      if (updated) {
        setReminder(updated);
      }
    } catch (err) {
      console.error("Failed to update reminder:", err);
      toast.error(t("reminderDetails.updateFailed"));
    }
  };

  const handleTogglePause = async () => {
    if (!reminder || !reminderId) return;

    try {
      if (isPaused) {
        await remindersService.resumeReminder(reminderId);
        toast.success(t("reminderDetails.resumedSuccessfully"));
      } else {
        await remindersService.pauseReminder(reminderId);
        toast.success(t("reminderDetails.pausedSuccessfully"));
      }

      setIsPaused(!isPaused);

      // Reload reminder data
      const updated = await remindersService.getReminder(reminderId);
      if (updated) {
        setReminder(updated);
      }
    } catch (err) {
      console.error("Failed to toggle pause:", err);
      toast.error(
        isPaused
          ? t("reminderDetails.resumeFailed")
          : t("reminderDetails.pauseFailed")
      );
    }
  };

  const handleDeleteClick = () => {
    setShowDeleteConfirm(true);
  };

  const handleConfirmDelete = async () => {
    if (!reminder || !reminderId) return;

    setIsDeleting(true);
    try {
      await remindersService.deleteReminder(reminderId);
      toast.success(t("reminderDetails.deletedSuccessfully"));
      navigate("/home");
    } catch (err) {
      console.error("Failed to delete reminder:", err);
      toast.error(t("reminderDetails.deleteFailed"));
      setIsDeleting(false);
      setShowDeleteConfirm(false);
    }
  };

  const handleCancelDelete = () => {
    setShowDeleteConfirm(false);
  };

  const handleDuplicate = async () => {
    if (!reminder || !reminderId) return;

    try {
      const duplicated = await remindersService.duplicateReminder(reminderId);
      if (duplicated) {
        toast.success(t("reminderDetails.duplicatedSuccessfully"));
        navigate(`/reminders/${duplicated.id}`);
      }
    } catch (err) {
      console.error("Failed to duplicate reminder:", err);
      toast.error(t("reminderDetails.duplicateFailed"));
    }
  };

  const formatDisplayDate = (date: Date) => {
    return date.toLocaleDateString("en-US", {
      weekday: "short",
      year: "numeric",
      month: "short",
      day: "numeric",
    });
  };

  if (isLoading) {
    return (
      <div className="min-h-screen bg-background-main dark:bg-background-main flex items-center justify-center">
        <div className="text-foreground text-lg">
          {t("common.loading") || "Loading..."}
        </div>
      </div>
    );
  }

  if (!reminder) {
    return (
      <div className="min-h-screen bg-background-main dark:bg-background-main flex items-center justify-center">
        <div className="text-foreground text-lg">
          {t("reminderDetails.notFound") || "Reminder not found"}
        </div>
      </div>
    );
  }

  const isUpcoming = new Date(reminder.remind_at_utc) > new Date();
  const displayParts = formatPartsInTimezone(
    new Date(reminder.remind_at_utc),
    displayTimezone
  );

  return (
    <div className="min-h-screen bg-background-main dark:bg-background-main relative overflow-hidden">
      {/* Subtle Settings Icon Pattern Background */}
      <div className="absolute inset-0 opacity-8 pointer-events-none overflow-hidden">
        {/* MAIN CLUSTER - Bottom Right */}
        <div
          className="absolute bottom-20 right-4 text-foreground"
          style={{ transform: "rotate(15deg)" }}
        >
          <Settings size={300} strokeWidth={0.8} className="opacity-50" />
        </div>

        <div
          className="absolute bottom-50 right-64 text-foreground"
          style={{ transform: "rotate(-60deg)" }}
        >
          <Settings size={200} strokeWidth={0.8} className="opacity-45" />
        </div>

        <div
          className="absolute bottom-82 right-36 text-foreground"
          style={{ transform: "rotate(45deg)" }}
        >
          <Settings size={130} strokeWidth={0.8} className="opacity-42" />
        </div>

        {/* TOP RIGHT CLUSTER */}
        <div className="absolute top-24 right-16 text-foreground">
          <Settings size={200} strokeWidth={0.8} className="opacity-44" />
        </div>

        <div
          className="absolute top-70 right-16 text-foreground"
          style={{ transform: "rotate(-40deg)" }}
        >
          <Settings size={150} strokeWidth={0.8} className="opacity-46" />
        </div>
      </div>

      <Header />
      <DisplayTimezoneNotice />

      <main className="max-w-4xl mx-auto px-4 sm:px-6 lg:px-8 py-12 pt-24 relative z-10">
        {/* Back Button */}
        <Button
          onClick={() => navigate("/home")}
          variant="ghost"
          className="mb-8 text-foreground hover:bg-secondary/50 gap-2"
        >
          <ChevronLeft className="w-4 h-4" />
          {t("reminderDetails.backToDashboard")}
        </Button>

        {/* Header with Status Badge */}
        <div className="mb-8 flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
          <div>
            <h1 className="text-4xl font-bold text-foreground mb-2">
              {isEditing
                ? t("reminderDetails.editingTitle")
                : t("reminderDetails.title")}
            </h1>
            <div className="flex items-center gap-3 flex-wrap">
              <p className="text-muted-foreground text-sm">
                {t("reminderDetails.id")}: {reminder.id.substring(0, 8)}
              </p>
              {isPaused && (
                <span className="px-3 py-1 rounded-full bg-yellow-500/10 border border-yellow-500/30 text-yellow-600 dark:text-yellow-400 text-xs font-medium">
                  {t("reminderDetails.paused")}
                </span>
              )}
              {isUpcoming && !isPaused && (
                <span className="px-3 py-1 rounded-full bg-green-500/10 border border-green-500/30 text-green-600 dark:text-green-400 text-xs font-medium flex items-center gap-1">
                  <CheckCircle2 className="w-3 h-3" />
                  {t("reminderDetails.active")}
                </span>
              )}
            </div>
          </div>

          {/* Action Buttons */}
          <div className="flex gap-2 flex-wrap">
            {!isEditing && (
              <>
                <Button
                  onClick={handleTogglePause}
                  variant="outline"
                  className="border-border text-foreground hover:bg-secondary/50 gap-2"
                >
                  {isPaused ? (
                    <>
                      <Play className="w-4 h-4" />
                      {t("reminderDetails.resume")}
                    </>
                  ) : (
                    <>
                      <Pause className="w-4 h-4" />
                      {t("reminderDetails.pause")}
                    </>
                  )}
                </Button>
                <Button
                  onClick={handleDuplicate}
                  variant="outline"
                  className="border-border text-foreground hover:bg-secondary/50 gap-2"
                >
                  <Copy className="w-4 h-4" />
                  {t("reminderDetails.duplicate")}
                </Button>
                <Button
                  onClick={handleEditToggle}
                  className="bg-accent hover:bg-accent/90 text-accent-foreground gap-2"
                >
                  <Edit2 className="w-4 h-4" />
                  {t("reminderDetails.edit")}
                </Button>
              </>
            )}
            {isEditing && (
              <>
                <Button
                  onClick={handleEditToggle}
                  variant="outline"
                  className="border-border text-foreground hover:bg-secondary/50 gap-2"
                >
                  <X className="w-4 h-4" />
                  {t("reminderDetails.cancel")}
                </Button>
                <Button
                  onClick={handleSave}
                  className="bg-accent hover:bg-accent/90 text-accent-foreground gap-2"
                >
                  <Save className="w-4 h-4" />
                  {t("reminderDetails.save")}
                </Button>
              </>
            )}
          </div>
        </div>

        {/* Main Content */}
        <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
          {/* Left Column - Reminder Details */}
          <div className="lg:col-span-2 space-y-6">
            {/* Message Section */}
            <Card className="border-border bg-card/95 backdrop-blur overflow-hidden">
              <CardContent className="pt-6">
                <div className="flex items-start gap-3 mb-4">
                  <div className="w-10 h-10 rounded-lg bg-accent/10 flex items-center justify-center flex-shrink-0">
                    <MessageSquare className="w-5 h-5 text-accent" />
                  </div>
                  <div>
                    <p className="text-sm font-semibold text-muted-foreground">
                      {t("reminderDetails.message")}
                    </p>
                  </div>
                </div>

                {isEditing ? (
                  <textarea
                    value={editData.message}
                    onChange={(e) =>
                      setEditData({ ...editData, message: e.target.value })
                    }
                    className="w-full px-4 py-3 rounded-lg border border-border bg-background text-foreground placeholder-muted-foreground resize-none h-32"
                    placeholder={t("reminderDetails.messagePlaceholder")}
                  />
                ) : (
                  <p className="text-lg text-foreground leading-relaxed bg-secondary/20 rounded-lg p-4">
                    {reminder.message}
                  </p>
                )}
              </CardContent>
            </Card>

            {/* Date & Time Section */}
            <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
              {/* Date */}
              <Card className="border-border bg-card/95 backdrop-blur">
                <CardContent className="pt-6">
                  <div className="flex items-start gap-3 mb-4">
                    <div className="w-10 h-10 rounded-lg bg-blue-500/10 flex items-center justify-center flex-shrink-0">
                      <Calendar className="w-5 h-5 text-blue-500" />
                    </div>
                    <div>
                      <p className="text-sm font-semibold text-muted-foreground">
                        {t("reminderDetails.date")}
                      </p>
                    </div>
                  </div>

                  {isEditing ? (
                    <input
                      type="date"
                      value={editData.date}
                      onChange={(e) =>
                        setEditData({
                          ...editData,
                          date: e.target.value,
                        })
                      }
                      className="w-full px-3 py-2 rounded border border-border bg-background text-foreground"
                    />
                  ) : (
                    <p className="text-lg font-semibold text-foreground">
                      {formatDisplayDate(
                        parseDateStrLocal(displayParts.dateStr)
                      )}
                    </p>
                  )}
                </CardContent>
              </Card>

              {/* Time */}
              <Card className="border-border bg-card/95 backdrop-blur">
                <CardContent className="pt-6">
                  <div className="flex items-start gap-3 mb-4">
                    <div className="w-10 h-10 rounded-lg bg-purple-500/10 flex items-center justify-center flex-shrink-0">
                      <Clock className="w-5 h-5 text-purple-500" />
                    </div>
                    <div>
                      <p className="text-sm font-semibold text-muted-foreground">
                        {t("reminderDetails.time")}
                      </p>
                    </div>
                  </div>

                  {isEditing ? (
                    <input
                      type="time"
                      value={editData.time}
                      onChange={(e) =>
                        setEditData({ ...editData, time: e.target.value })
                      }
                      className="w-full px-3 py-2 rounded border border-border bg-background text-foreground"
                    />
                  ) : (
                    <div className="space-y-2">
                      <p className="text-lg font-semibold text-foreground">
                        {displayParts.timeStr}
                      </p>
                      <p className="text-xs text-muted-foreground">
                        {displayTimezone}
                      </p>
                    </div>
                  )}
                </CardContent>
              </Card>
            </div>

            {/* Recurrence Section */}
            <Card className="border-border bg-card/95 backdrop-blur">
              <CardContent className="pt-6">
                <div className="flex items-start gap-3 mb-4">
                  <div className="w-10 h-10 rounded-lg bg-green-500/10 flex items-center justify-center flex-shrink-0">
                    <Zap className="w-5 h-5 text-green-500" />
                  </div>
                  <div>
                    <p className="text-sm font-semibold text-muted-foreground">
                      {t("reminderDetails.recurrence")}
                    </p>
                  </div>
                </div>

                {isEditing ? (
                  <div className="grid grid-cols-2 gap-2">
                    <select
                      value={editData.recurrence}
                      onChange={(e) =>
                        setEditData({
                          ...editData,
                          recurrence: e.target.value,
                        })
                      }
                      className="col-span-2 px-3 py-2 rounded border border-border bg-background text-foreground"
                    >
                      <option value="ONCE">Once</option>
                      <option value="YEARLY">Yearly</option>
                      <option value="MONTHLY">Monthly</option>
                      <option value="WEEKLY">Weekly</option>
                      <option value="DAILY">Daily</option>
                      <option value="HOURLY">Hourly</option>
                      <option value="WORKDAYS">Workdays</option>
                      <option value="WEEKEND">Weekend</option>
                    </select>
                  </div>
                ) : (
                  <p className="text-lg font-semibold text-foreground">
                    {t(getRecurrenceTypeI18nKeyFromString(editData.recurrence))}
                  </p>
                )}
              </CardContent>
            </Card>

            {/* Destinations Section */}
            <Card className="border-border bg-card/95 backdrop-blur">
              <CardContent className="pt-6">
                {isEditing ? (
                  <DestinationPicker
                    destinations={editData.destinations}
                    onDestinationsChange={(destinations) =>
                      setEditData({ ...editData, destinations })
                    }
                    showTitle={true}
                    showAddOptions={true}
                    compact={false}
                  />
                ) : (
                  <div className="space-y-4">
                    <p className="text-sm font-semibold text-muted-foreground">
                      {t("reminderDetails.destinations")}
                    </p>
                    {reminder.destinations &&
                    reminder.destinations.length > 0 ? (
                      <div className="space-y-2">
                        {reminder.destinations.map((dest, idx) => (
                          <div
                            key={idx}
                            className="flex items-center gap-3 p-3 rounded-lg border border-border bg-secondary/20"
                          >
                            {dest.type === "discord_dm" && (
                              <>
                                <div className="w-8 h-8 rounded-lg bg-indigo-500/10 flex items-center justify-center flex-shrink-0">
                                  <svg
                                    className="w-4 h-4 text-indigo-500"
                                    fill="currentColor"
                                    viewBox="0 0 24 24"
                                  >
                                    <path d="M20.317 4.37a19.791 19.791 0 00-4.885-1.515.074.074 0 00-.079.037c-.21.375-.444.864-.607 1.25a18.27 18.27 0 00-5.487 0c-.163-.386-.397-.875-.61-1.25a.077.077 0 00-.079-.037A19.736 19.736 0 003.677 4.37a.07.07 0 00-.032.027C.533 9.046-.32 13.58.099 18.057a.082.082 0 00.031.056 19.9 19.9 0 005.993 3.03.078.078 0 00.084-.028c.462-.63.874-1.295 1.226-1.994a.076.076 0 00-.042-.106 13.107 13.107 0 01-1.872-.892.077.077 0 00-.008-.128 10.713 10.713 0 00.372-.294.075.075 0 00.03-.066c.329-.246.648-.5.954-.76a.07.07 0 00.076-.01 13.697 13.697 0 0011.086 0 .07.07 0 00.076.009c.305.26.625.514.954.759a.077.077 0 00.03.067c.12.088.246.177.371.294a.077.077 0 00-.006.127 13.227 13.227 0 01-1.873.892.076.076 0 00-.041.107c.352.699.764 1.364 1.225 1.994a.076.076 0 00.084.028 19.963 19.963 0 006.002-3.03.077.077 0 00.032-.054c.5-4.817-.838-9.033-3.55-12.765a.061.061 0 00-.031-.03zM8.02 15.33c-1.183 0-2.157-.969-2.157-2.156 0-1.193.974-2.157 2.157-2.157 1.193 0 2.156.964 2.156 2.157 0 1.187-.963 2.156-2.156 2.156zm7.975 0c-1.183 0-2.157-.969-2.157-2.156 0-1.193.974-2.157 2.157-2.157 1.193 0 2.157.964 2.157 2.157 0 1.187-.964 2.156-2.157 2.156z" />
                                  </svg>
                                </div>
                                <p className="text-sm font-medium text-foreground">
                                  Discord Direct Message
                                </p>
                              </>
                            )}
                            {dest.type === "discord_channel" && (
                              <>
                                <div className="w-8 h-8 rounded-lg bg-indigo-500/10 flex items-center justify-center flex-shrink-0">
                                  <svg
                                    className="w-4 h-4 text-indigo-500"
                                    fill="currentColor"
                                    viewBox="0 0 24 24"
                                  >
                                    <path d="M20.317 4.37a19.791 19.791 0 00-4.885-1.515.074.074 0 00-.079.037c-.21.375-.444.864-.607 1.25a18.27 18.27 0 00-5.487 0c-.163-.386-.397-.875-.61-1.25a.077.077 0 00-.079-.037A19.736 19.736 0 003.677 4.37a.07.07 0 00-.032.027C.533 9.046-.32 13.58.099 18.057a.082.082 0 00.031.056 19.9 19.9 0 005.993 3.03.078.078 0 00.084-.028c.462-.63.874-1.295 1.226-1.994a.076.076 0 00-.042-.106 13.107 13.107 0 01-1.872-.892.077.077 0 00-.008-.128 10.713 10.713 0 00.372-.294.075.075 0 00.03-.066c.329-.246.648-.5.954-.76a.07.07 0 00.076-.01 13.697 13.697 0 0011.086 0 .07.07 0 00.076.009c.305.26.625.514.954.759a.077.077 0 00.03.067c.12.088.246.177.371.294a.077.077 0 00-.006.127 13.227 13.227 0 01-1.873.892.076.076 0 00-.041.107c.352.699.764 1.364 1.225 1.994a.076.076 0 00.084.028 19.963 19.963 0 006.002-3.03.077.077 0 00.032-.054c.5-4.817-.838-9.033-3.55-12.765a.061.061 0 00-.031-.03zM8.02 15.33c-1.183 0-2.157-.969-2.157-2.156 0-1.193.974-2.157 2.157-2.157 1.193 0 2.156.964 2.156 2.157 0 1.187-.963 2.156-2.156 2.156zm7.975 0c-1.183 0-2.157-.969-2.157-2.156 0-1.193.974-2.157 2.157-2.157 1.193 0 2.157.964 2.157 2.157 0 1.187-.964 2.156-2.157 2.156z" />
                                  </svg>
                                </div>
                                <p className="text-sm font-medium text-foreground">
                                  Discord Channel (
                                  {(dest.metadata.channel_id as string) ||
                                    "N/A"}
                                  )
                                </p>
                              </>
                            )}
                            {dest.type === "webhook" && (
                              <>
                                <div className="w-8 h-8 rounded-lg bg-orange-500/10 flex items-center justify-center flex-shrink-0">
                                  <svg
                                    className="w-4 h-4 text-orange-500"
                                    fill="currentColor"
                                    viewBox="0 0 24 24"
                                  >
                                    <path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm-2 15l-5-5 1.41-1.41L10 14.17l7.59-7.59L19 8l-9 9z" />
                                  </svg>
                                </div>
                                <p className="text-sm font-medium text-foreground truncate">
                                  Webhook
                                </p>
                                <p className="text-xs text-muted-foreground ml-auto truncate">
                                  {(dest.metadata.url as string) || "N/A"}
                                </p>
                              </>
                            )}
                            {dest.type === "email" && (
                              <>
                                <div className="w-8 h-8 rounded-lg bg-accent/10 flex items-center justify-center flex-shrink-0">
                                  <svg
                                    className="w-4 h-4 text-accent"
                                    fill="none"
                                    stroke="currentColor"
                                    strokeWidth="2"
                                    viewBox="0 0 24 24"
                                  >
                                    <path strokeLinecap="round" strokeLinejoin="round" d="M3 8l7.89 5.26a2 2 0 002.22 0L21 8M5 19h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z" />
                                  </svg>
                                </div>
                                <p className="text-sm font-medium text-foreground">
                                  {t("reminderCreation.destinations.email")}
                                </p>
                                <p className="text-xs text-muted-foreground ml-auto truncate">
                                  {(dest.metadata.email as string) || "N/A"}
                                </p>
                              </>
                            )}
                            {dest.type === "android_push" && (
                              <>
                                <div className="w-8 h-8 rounded-lg bg-emerald-500/10 flex items-center justify-center flex-shrink-0">
                                  <svg
                                    className="w-4 h-4 text-emerald-500"
                                    fill="none"
                                    stroke="currentColor"
                                    strokeWidth="2"
                                    viewBox="0 0 24 24"
                                  >
                                    <rect x="5" y="2" width="14" height="20" rx="2" ry="2" />
                                    <line x1="12" y1="18" x2="12.01" y2="18" />
                                  </svg>
                                </div>
                                <p className="text-sm font-medium text-foreground">
                                  {t("reminderCreation.destinations.androidPush")}
                                </p>
                              </>
                            )}
                          </div>
                        ))}
                      </div>
                    ) : (
                      <p className="text-sm text-muted-foreground italic">
                        {t("reminderDetails.noDestinations")}
                      </p>
                    )}
                  </div>
                )}
              </CardContent>
            </Card>
          </div>

          {/* Right Column - Info & Actions */}
          <div className="space-y-6">
            {/* Info Card */}
            <Card className="border-border bg-card/95 backdrop-blur">
              <CardContent className="pt-6">
                <h3 className="text-sm font-semibold text-foreground mb-4">
                  {t("reminderDetails.info")}
                </h3>
                <div className="space-y-3 text-sm">
                  <div>
                    <p className="text-muted-foreground text-xs mb-1">
                      {t("reminderDetails.created")}
                    </p>
                    <p className="text-foreground font-medium">
                      {formatDateInTimezone(
                        new Date(reminder.created_at),
                        displayTimezone
                      )}
                    </p>
                  </div>
                  <div>
                    <p className="text-muted-foreground text-xs mb-1">
                      {t("reminderDetails.status")}
                    </p>
                    <p className="text-foreground font-medium">
                      {isPaused
                        ? t("reminderDetails.paused")
                        : isUpcoming
                        ? t("reminderDetails.active")
                        : t("reminderDetails.completed")}
                    </p>
                  </div>
                  <div>
                    <p className="text-muted-foreground text-xs mb-1">
                      Reminder ID
                    </p>
                    <p className="text-foreground font-mono text-xs break-all">
                      {reminder.id}
                    </p>
                  </div>
                </div>
              </CardContent>
            </Card>

            {/* Danger Zone */}
            {!isEditing && (
              <Card className="border-red-500/30 bg-red-500/5 backdrop-blur">
                <CardContent className="pt-6">
                  <h3 className="text-sm font-semibold text-red-600 dark:text-red-400 mb-4">
                    {t("reminderDetails.dangerZone")}
                  </h3>
                  <Button
                    onClick={handleDeleteClick}
                    variant="outline"
                    className="w-full border-red-500/50 text-red-600 dark:text-red-400 hover:bg-red-500/10 gap-2"
                  >
                    <Trash2 className="w-4 h-4" />
                    {t("reminderDetails.delete")}
                  </Button>
                </CardContent>
              </Card>
            )}
          </div>
        </div>
      </main>

      <DeleteConfirmModal
        isOpen={showDeleteConfirm}
        title={t("reminderDetails.deleteConfirmTitle") || "Delete Reminder?"}
        description={
          t("reminderDetails.deleteConfirm") ||
          "This action cannot be undone. Are you sure you want to delete this reminder?"
        }
        onConfirm={handleConfirmDelete}
        onCancel={handleCancelDelete}
        isLoading={isDeleting}
      />

      <Footer />
    </div>
  );
}
