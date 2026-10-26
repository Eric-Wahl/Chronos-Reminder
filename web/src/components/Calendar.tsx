import { useState, useMemo, useEffect } from "react";
import { useTranslation } from "react-i18next";
import { ChevronLeft, ChevronRight, Clock } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { ReminderDetailModal } from "@/components/ReminderDetailModal";
import { cn } from "@/lib/utils";
import {
  formatTimeInTimezone,
  getDateKeyInTimezone,
  getLocalDateKey,
  getTodayAsCalendarDate,
} from "@/lib/timezone";
import type { Reminder } from "@/services";

interface CalendarProps {
  reminders?: Reminder[];
  /** IANA timezone used to decide which day a reminder falls on, and to format its time */
  timeZone: string;
  onAddReminder?: (date: Date) => void;
}

export function Calendar({
  reminders = [],
  timeZone,
  onAddReminder,
}: CalendarProps) {
  const { t } = useTranslation();
  const [currentDate, setCurrentDate] = useState(() =>
    getTodayAsCalendarDate(timeZone)
  );
  const [selectedDate, setSelectedDate] = useState<Date | null>(null);
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [displayedReminders, setDisplayedReminders] = useState(reminders);
  const [isDatePickerOpen, setIsDatePickerOpen] = useState(false);
  const [datePickerYear, setDatePickerYear] = useState(() =>
    getTodayAsCalendarDate(timeZone).getFullYear()
  );
  const [datePickerMonth, setDatePickerMonth] = useState(() =>
    getTodayAsCalendarDate(timeZone).getMonth()
  );

  // Sync displayed reminders when props change
  useEffect(() => {
    setDisplayedReminders(reminders);
  }, [reminders]);

  // Month names and day names
  const monthNames = [
    t("calendar.january"),
    t("calendar.february"),
    t("calendar.march"),
    t("calendar.april"),
    t("calendar.may"),
    t("calendar.june"),
    t("calendar.july"),
    t("calendar.august"),
    t("calendar.september"),
    t("calendar.october"),
    t("calendar.november"),
    t("calendar.december"),
  ];

  const dayNames = [
    t("calendar.sunday"),
    t("calendar.monday"),
    t("calendar.tuesday"),
    t("calendar.wednesday"),
    t("calendar.thursday"),
    t("calendar.friday"),
    t("calendar.saturday"),
  ];

  // Reminders grouped by the YYYY-MM-DD they fall on in the display timezone
  const remindersByDay = useMemo(() => {
    const map = new Map<string, Reminder[]>();
    for (const reminder of displayedReminders) {
      const key = getDateKeyInTimezone(
        new Date(reminder.remind_at_utc),
        timeZone
      );
      const list = map.get(key);
      if (list) {
        list.push(reminder);
      } else {
        map.set(key, [reminder]);
      }
    }
    return map;
  }, [displayedReminders, timeZone]);

  // Find the next month with reminders
  const getNextReminderMonth = () => {
    let nextDate = new Date(
      currentDate.getFullYear(),
      currentDate.getMonth() + 1,
      1
    );
    let attempts = 0;
    const maxAttempts = 24; // Check up to 2 years ahead

    while (attempts < maxAttempts) {
      const monthPrefix = getLocalDateKey(nextDate).slice(0, 7);
      const hasReminders = Array.from(remindersByDay.keys()).some((key) =>
        key.startsWith(monthPrefix)
      );

      if (hasReminders) {
        return nextDate;
      }

      nextDate = new Date(nextDate.getFullYear(), nextDate.getMonth() + 1, 1);
      attempts++;
    }

    return null; // No future months with reminders
  };

  const goToNextReminderMonth = () => {
    const nextMonth = getNextReminderMonth();
    if (nextMonth) {
      setCurrentDate(nextMonth);
    }
  };

  // Get reminders for specific date
  const getRemindersForDate = (date: Date): Reminder[] => {
    return remindersByDay.get(getLocalDateKey(date)) ?? [];
  };

  // Get calendar days
  const calendarDays = useMemo(() => {
    const year = currentDate.getFullYear();
    const month = currentDate.getMonth();
    const firstDay = new Date(year, month, 1);
    const lastDay = new Date(year, month + 1, 0);
    const daysInMonth = lastDay.getDate();
    const startingDayOfWeek = firstDay.getDay();

    const days: (number | null)[] = [];

    // Fill in empty days before month starts
    for (let i = 0; i < startingDayOfWeek; i++) {
      days.push(null);
    }

    // Fill in days of the month
    for (let i = 1; i <= daysInMonth; i++) {
      days.push(i);
    }

    // Fill in empty days after month ends
    const remainingDays = 42 - days.length; // 6 rows * 7 days
    for (let i = 0; i < remainingDays; i++) {
      days.push(null);
    }

    return days;
  }, [currentDate]);

  const previousMonth = () => {
    setCurrentDate(
      new Date(currentDate.getFullYear(), currentDate.getMonth() - 1)
    );
  };

  const nextMonth = () => {
    setCurrentDate(
      new Date(currentDate.getFullYear(), currentDate.getMonth() + 1)
    );
  };

  const goToToday = () => {
    setCurrentDate(getTodayAsCalendarDate(timeZone));
  };

  const todayKey = getDateKeyInTimezone(new Date(), timeZone);
  const isToday = (day: number | null) => {
    if (!day) return false;
    return (
      getLocalDateKey(
        new Date(currentDate.getFullYear(), currentDate.getMonth(), day)
      ) === todayKey
    );
  };

  const isCurrentMonth = (day: number | null) => day !== null;

  const handleReminderDeleted = (reminderId: string) => {
    // Remove the deleted reminder from our local state
    setDisplayedReminders(
      displayedReminders.filter((r) => r.id !== reminderId)
    );
  };

  return (
    <div className="w-full space-y-6">
      {/* Header Section */}
      <div className="flex flex-col md:flex-row md:items-center md:justify-between gap-4">
        <div>
          <h2 className="text-3xl md:text-4xl font-bold text-foreground">
            {t("calendar.title")}
          </h2>
          <p className="text-muted-foreground text-sm md:text-base mt-1">
            {t("calendar.subtitle")}
          </p>
        </div>
        <div className="flex gap-2 w-full md:w-auto flex-col sm:flex-row">
          {displayedReminders.length > 0 && getNextReminderMonth() && (
            <Button
              onClick={goToNextReminderMonth}
              className="bg-secondary hover:bg-secondary/80 text-foreground font-semibold"
            >
              {t("calendar.nextReminder")}
            </Button>
          )}
          <Button
            onClick={goToToday}
            className="bg-accent hover:bg-accent/90 text-accent-foreground font-semibold"
          >
            {t("calendar.today")}
          </Button>
        </div>
      </div>

      {/* Calendar Card */}
      <Card className="border-border bg-card/95 backdrop-blur overflow-hidden">
        {/* Month Navigation */}
        <div className="flex items-center justify-between p-4 md:p-6 border-b border-border gap-2 relative">
          <Button
            onClick={previousMonth}
            variant="ghost"
            size="sm"
            className="text-foreground hover:text-accent hover:bg-accent/10"
          >
            <ChevronLeft className="w-5 h-5" />
            <span className="hidden sm:inline ml-1 text-sm">
              {t("calendar.previousMonth")}
            </span>
          </Button>

          <div className="flex-1 flex justify-center">
            <button
              onClick={() => {
                setIsDatePickerOpen(!isDatePickerOpen);
                setDatePickerYear(currentDate.getFullYear());
                setDatePickerMonth(currentDate.getMonth());
              }}
              className="text-xl md:text-2xl font-bold text-foreground text-center min-w-[200px] px-4 py-2 rounded-lg hover:bg-accent/10 transition-colors cursor-pointer"
            >
              {monthNames[currentDate.getMonth()]} {currentDate.getFullYear()}
            </button>
          </div>

          <Button
            onClick={nextMonth}
            variant="ghost"
            size="sm"
            className="text-foreground hover:text-accent hover:bg-accent/10"
          >
            <span className="hidden sm:inline mr-1 text-sm">
              {t("calendar.nextMonth")}
            </span>
            <ChevronRight className="w-5 h-5" />
          </Button>

          {/* Date Picker Popover */}
          {isDatePickerOpen && (
            <div className="absolute top-full left-1/2 transform -translate-x-1/2 mt-2 bg-card border-2 border-border rounded-lg shadow-lg p-4 z-50 backdrop-blur">
              <div className="w-80">
                {/* Year and Month Selection */}
                <div className="flex items-center justify-between gap-2 mb-4">
                  <Button
                    onClick={() => setDatePickerYear(datePickerYear - 1)}
                    variant="ghost"
                    size="sm"
                    className="text-foreground hover:text-accent hover:bg-accent/10"
                  >
                    <ChevronLeft className="w-4 h-4" />
                  </Button>
                  <span className="text-sm font-semibold text-foreground min-w-[60px] text-center">
                    {datePickerYear}
                  </span>
                  <Button
                    onClick={() => setDatePickerYear(datePickerYear + 1)}
                    variant="ghost"
                    size="sm"
                    className="text-foreground hover:text-accent hover:bg-accent/10"
                  >
                    <ChevronRight className="w-4 h-4" />
                  </Button>
                </div>

                {/* Month Grid */}
                <div className="grid grid-cols-3 gap-2 mb-4">
                  {monthNames.map((month, index) => (
                    <Button
                      key={month}
                      onClick={() => {
                        setCurrentDate(new Date(datePickerYear, index, 1));
                        setIsDatePickerOpen(false);
                      }}
                      variant="ghost"
                      size="sm"
                      className={cn(
                        "text-xs font-medium h-8 transition-colors",
                        datePickerMonth === index
                          ? "bg-accent text-accent-foreground hover:bg-accent/90"
                          : "text-foreground hover:text-accent hover:bg-accent/10"
                      )}
                    >
                      {month.slice(0, 3)}
                    </Button>
                  ))}
                </div>

                {/* Quick Actions */}
                <div className="flex gap-2">
                  <Button
                    onClick={() => {
                      setCurrentDate(getTodayAsCalendarDate(timeZone));
                      setIsDatePickerOpen(false);
                    }}
                    variant="ghost"
                    size="sm"
                    className="flex-1 text-xs text-foreground hover:text-accent hover:bg-accent/10"
                  >
                    {t("calendar.today")}
                  </Button>
                  <Button
                    onClick={() => setIsDatePickerOpen(false)}
                    variant="ghost"
                    size="sm"
                    className="flex-1 text-xs text-foreground hover:text-accent hover:bg-accent/10"
                  >
                    {t("common.close") || "Close"}
                  </Button>
                </div>
              </div>
            </div>
          )}
        </div>

        {/* Desktop Calendar Grid */}
        <div className="hidden md:block p-6">
          {/* Day Headers */}
          <div className="grid grid-cols-7 gap-2 mb-4">
            {dayNames.map((day) => (
              <div
                key={day}
                className="h-12 flex items-center justify-center font-semibold text-muted-foreground text-sm"
              >
                {day}
              </div>
            ))}
          </div>

          {/* Calendar Days */}
          <div className="grid grid-cols-7 gap-2">
            {calendarDays.map((day, index) => {
              const dayReminders = day
                ? getRemindersForDate(
                    new Date(
                      currentDate.getFullYear(),
                      currentDate.getMonth(),
                      day
                    )
                  )
                : [];
              const today = isToday(day);
              const inMonth = isCurrentMonth(day);

              const handleDateClick = () => {
                if (day && inMonth) {
                  setSelectedDate(
                    new Date(
                      currentDate.getFullYear(),
                      currentDate.getMonth(),
                      day
                    )
                  );
                  setIsModalOpen(true);
                }
              };

              return (
                <div
                  key={index}
                  onClick={handleDateClick}
                  className={cn(
                    "relative h-24 rounded-lg border-2 transition-all duration-200",
                    !inMonth && "bg-muted/30 border-transparent cursor-default",
                    inMonth &&
                      "bg-secondary/40 border-border hover:border-accent hover:bg-secondary/60 cursor-pointer",
                    today && "border-accent bg-accent/10"
                  )}
                >
                  {day && (
                    <div className="p-2 h-full flex flex-col">
                      <span
                        className={cn(
                          "text-sm font-semibold mb-1",
                          today ? "text-accent" : "text-foreground"
                        )}
                      >
                        {day}
                      </span>
                      {dayReminders.length > 0 && (
                        <div className="flex-1 overflow-hidden">
                          <div className="flex items-start gap-1">
                            <Clock className="w-3 h-3 text-accent flex-shrink-0 mt-0.5" />
                            <span className="text-xs text-accent font-semibold truncate">
                              {dayReminders.length}{" "}
                              {dayReminders.length === 1
                                ? t("calendar.remindersOnDate").slice(0, -1)
                                : t("calendar.remindersOnDate")}
                            </span>
                          </div>
                        </div>
                      )}
                    </div>
                  )}
                </div>
              );
            })}
          </div>
        </div>

        {/* Mobile Calendar List View */}
        <div className="md:hidden p-4 space-y-3">
          {calendarDays.map((day, index) => {
            if (!day || !isCurrentMonth(day)) return null;

            const dayReminders = getRemindersForDate(
              new Date(currentDate.getFullYear(), currentDate.getMonth(), day)
            );
            const today = isToday(day);
            const dayName =
              dayNames[
                new Date(
                  currentDate.getFullYear(),
                  currentDate.getMonth(),
                  day
                ).getDay()
              ];

            const handleDateClick = () => {
              setSelectedDate(
                new Date(currentDate.getFullYear(), currentDate.getMonth(), day)
              );
              setIsModalOpen(true);
            };

            return (
              <div
                key={index}
                onClick={handleDateClick}
                className={cn(
                  "rounded-lg border-2 p-3 transition-all duration-200 cursor-pointer",
                  today
                    ? "border-accent bg-accent/10"
                    : "border-border bg-secondary/40"
                )}
              >
                <div className="flex items-center justify-between mb-2">
                  <div className="flex items-center gap-2">
                    <span
                      className={cn(
                        "font-bold text-lg",
                        today ? "text-accent" : "text-foreground"
                      )}
                    >
                      {day}
                    </span>
                    <span className="text-xs text-muted-foreground">
                      {dayName}
                    </span>
                  </div>
                  {today && (
                    <span className="text-xs bg-accent text-accent-foreground px-2 py-1 rounded-full font-semibold">
                      {t("calendar.today")}
                    </span>
                  )}
                </div>

                {dayReminders.length > 0 ? (
                  <div className="space-y-2">
                    {dayReminders.map((reminder) => (
                      <div
                        key={reminder.id}
                        className="bg-accent/20 rounded px-2 py-1 border border-accent/30"
                      >
                        <div className="flex items-start gap-2">
                          <Clock className="w-3 h-3 text-accent flex-shrink-0 mt-1" />
                          <div className="flex-1 min-w-0">
                            <p className="text-xs text-foreground truncate">
                              {reminder.message}
                            </p>
                            <p className="text-xs text-muted-foreground">
                              {formatTimeInTimezone(
                                new Date(reminder.remind_at_utc),
                                timeZone
                              )}
                            </p>
                          </div>
                        </div>
                      </div>
                    ))}
                  </div>
                ) : (
                  <p className="text-xs text-muted-foreground italic">
                    {t("calendar.noReminders")}
                  </p>
                )}
              </div>
            );
          })}
        </div>
      </Card>

      {/* Reminder Detail Modal */}
      <ReminderDetailModal
        isOpen={isModalOpen}
        date={selectedDate}
        reminders={selectedDate ? getRemindersForDate(selectedDate) : []}
        timeZone={timeZone}
        onClose={() => {
          setIsModalOpen(false);
          setSelectedDate(null);
        }}
        onReminderDeleted={handleReminderDeleted}
        onAddNew={(date) => {
          setIsModalOpen(false);
          setSelectedDate(null);
          onAddReminder?.(date);
        }}
      />
    </div>
  );
}
