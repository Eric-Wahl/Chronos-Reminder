import {
  Plus,
  Clock,
  Bell,
  Search,
  Filter,
  ChevronLeft,
  ChevronRight,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { useTranslation } from "react-i18next";
import { useNavigate } from "react-router-dom";
import { useState, useMemo } from "react";
import type { Reminder } from "@/services";
import {
  formatShortDateInTimezone,
  formatTimeInTimezone,
  getDateKeyInTimezone,
} from "@/lib/timezone";

interface RemindersListProps {
  reminders: Reminder[];
  /** IANA timezone the reminder dates and times are displayed in */
  timeZone: string;
  onAddReminder: () => void;
}

const ITEMS_PER_PAGE = 5;

export function RemindersList({
  reminders,
  timeZone,
  onAddReminder,
}: RemindersListProps) {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const [searchQuery, setSearchQuery] = useState("");
  const [selectedYear, setSelectedYear] = useState<string>("all");
  const [selectedDestination, setSelectedDestination] = useState<string>("all");
  const [currentPage, setCurrentPage] = useState(1);

  // Extract unique years from reminders
  const years = useMemo(() => {
    const yearSet = new Set(
      reminders.map((r) =>
        getDateKeyInTimezone(new Date(r.remind_at_utc), timeZone).slice(0, 4)
      )
    );
    return Array.from(yearSet).sort((a, b) => parseInt(b) - parseInt(a));
  }, [reminders, timeZone]);

  // Extract unique destinations
  const destinations = useMemo(() => {
    const destSet = new Set<string>();
    reminders.forEach((r) => {
      if (r.destinations) {
        r.destinations.forEach((d) => destSet.add(d.type));
      }
    });
    return Array.from(destSet).sort();
  }, [reminders]);

  // Filter reminders
  const filteredReminders = useMemo(() => {
    return reminders.filter((reminder) => {
      const year = getDateKeyInTimezone(
        new Date(reminder.remind_at_utc),
        timeZone
      ).slice(0, 4);

      // Search filter
      const matchesSearch =
        reminder.message.toLowerCase().includes(searchQuery.toLowerCase()) ||
        reminder.id.toLowerCase().includes(searchQuery.toLowerCase());

      // Year filter
      const matchesYear = selectedYear === "all" || year === selectedYear;

      // Destination filter
      let matchesDestination = true;
      if (selectedDestination !== "all") {
        matchesDestination =
          (reminder.destinations &&
            reminder.destinations.some(
              (d) => d.type === selectedDestination
            )) ||
          false;
      }

      return matchesSearch && matchesYear && matchesDestination;
    });
  }, [reminders, searchQuery, selectedYear, selectedDestination, timeZone]);

  // Calculate pagination
  const totalPages = Math.ceil(filteredReminders.length / ITEMS_PER_PAGE);
  const startIndex = (currentPage - 1) * ITEMS_PER_PAGE;
  const endIndex = startIndex + ITEMS_PER_PAGE;
  const paginatedReminders = filteredReminders.slice(startIndex, endIndex);

  // Reset to first page when filters change
  const handleFilterChange = (callback: () => void) => {
    setCurrentPage(1);
    callback();
  };

  // Use destinations to avoid unused variable warning
  void destinations;

  if (reminders.length === 0) {
    return (
      <div className="space-y-4">
        <div>
          <h3 className="text-2xl font-bold text-foreground mb-1">
            {t("reminders.title")}
          </h3>
          <p className="text-muted-foreground text-sm mb-9">
            {t("reminders.subtitle")}
          </p>
        </div>
        <Card className="border-border bg-card/60 backdrop-blur text-center py-12">
          <Bell className="w-12 h-12 text-muted-foreground mx-auto mb-4" />
          <h3 className="text-lg font-semibold text-foreground mb-2">
            {t("calendar.noReminders")}
          </h3>
          <p className="text-muted-foreground mb-6">{t("calendar.subtitle")}</p>
          <Button
            onClick={onAddReminder}
            className="bg-accent hover:bg-accent/90 text-accent-foreground font-semibold gap-2"
          >
            <Plus className="w-4 h-4" />
            {t("welcome.newReminder")}
          </Button>
        </Card>
      </div>
    );
  }

  return (
    <div className="space-y-4">
      <div>
        <h3 className="text-2xl font-bold text-foreground mb-1">
          {t("reminders.title")}
        </h3>
        <p className="text-muted-foreground text-sm mb-4">
          {t("reminders.subtitle")}
        </p>
      </div>

      {/* Search and Filter Bar */}
      <div className="space-y-3">
        {/* Search Bar */}
        <div className="relative">
          <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 w-4 h-4 text-muted-foreground" />
          <input
            type="text"
            placeholder={t("reminders.search")}
            value={searchQuery}
            onChange={(e) =>
              handleFilterChange(() => setSearchQuery(e.target.value))
            }
            className="w-full pl-10 pr-4 py-2 bg-card/60 border border-border rounded-lg text-foreground placeholder-muted-foreground focus:outline-none focus:border-accent/50 transition-colors"
          />
        </div>

        {/* Filter Controls */}
        <div className="flex flex-col sm:flex-row gap-3">
          {/* Year Filter */}
          <div className="flex-1 flex items-center gap-2">
            <Filter className="w-4 h-4 text-muted-foreground flex-shrink-0" />
            <select
              value={selectedYear}
              onChange={(e) =>
                handleFilterChange(() => setSelectedYear(e.target.value))
              }
              className="flex-1 px-3 py-2 bg-card/60 border border-border rounded-lg text-foreground text-sm focus:outline-none focus:border-accent/50 transition-colors"
            >
              <option value="all">{t("reminders.filterYear")}</option>
              {years.map((year) => (
                <option key={year} value={year}>
                  {year}
                </option>
              ))}
            </select>
          </div>

          {/* Destination Filter */}
          <div className="flex-1 flex items-center gap-2">
            <select
              value={selectedDestination}
              onChange={(e) =>
                handleFilterChange(() => setSelectedDestination(e.target.value))
              }
              className="flex-1 px-3 py-2 bg-card/60 border border-border rounded-lg text-foreground text-sm focus:outline-none focus:border-accent/50 transition-colors"
            >
              <option value="all">{t("reminders.filterDestination")}</option>
              <option value="discord_dm">{t("reminders.discordDM")}</option>
              <option value="discord_channel">
                {t("reminders.discordChannel")}
              </option>
              <option value="webhook">{t("reminders.webhook")}</option>
              <option value="email">{t("reminders.email")}</option>
            </select>
          </div>
        </div>

        {/* Results Counter */}
        <p className="text-xs text-muted-foreground">
          {filteredReminders.length} {t("reminders.resultsCounter")}{" "}
          {reminders.length}
        </p>
      </div>

      {/* Reminders List */}
      <div className="space-y-3">
        {filteredReminders.length > 0 ? (
          paginatedReminders.map((reminder) => {
            const reminderDate = new Date(reminder.remind_at_utc);
            const isUpcoming = reminderDate > new Date();
            const isPaused = reminder.is_paused;

            return (
              <Card
                key={reminder.id}
                className="border-border bg-card/60 backdrop-blur hover:bg-card/80 hover:border-accent/50 transition-all cursor-pointer group"
                onClick={() => navigate(`/reminders/${reminder.id}`)}
              >
                <CardContent className="p-4">
                  <div className="space-y-3">
                    {/* Header: Message and Status */}
                    <div className="flex items-start justify-between gap-2">
                      <div className="flex-1 min-w-0">
                        <p className="text-sm font-semibold text-foreground truncate group-hover:text-accent transition-colors">
                          {reminder.message}
                        </p>
                      </div>
                      <div className="flex gap-2 flex-shrink-0">
                        {isPaused && (
                          <span className="inline-flex items-center px-2 py-1 rounded-full text-xs font-medium bg-yellow-500/20 text-yellow-600 dark:text-yellow-400 border border-yellow-500/30">
                            {t("reminderDetails.paused")}
                          </span>
                        )}
                        {isUpcoming && !isPaused && (
                          <span className="inline-flex items-center px-2 py-1 rounded-full text-xs font-medium bg-green-500/20 text-green-600 dark:text-green-400 border border-green-500/30">
                            {t("reminderDetails.active")}
                          </span>
                        )}
                      </div>
                    </div>

                    {/* Date and Time */}
                    <div className="flex items-center gap-2 text-xs text-muted-foreground">
                      <Clock className="w-3 h-3" />
                      <span>
                        {formatShortDateInTimezone(reminderDate, timeZone)} at{" "}
                        {formatTimeInTimezone(reminderDate, timeZone)}
                      </span>
                    </div>

                    {/* Destinations */}
                    {reminder.destinations &&
                      reminder.destinations.length > 0 && (
                        <div className="flex items-center gap-2 flex-wrap">
                          {reminder.destinations.map((dest, idx) => (
                            <div
                              key={idx}
                              className="inline-flex items-center gap-1 px-2 py-1 rounded-md bg-secondary/40 border border-border/50"
                            >
                              {dest.type === "discord_dm" && (
                                <>
                                  <svg
                                    className="w-3 h-3 text-indigo-500"
                                    fill="currentColor"
                                    viewBox="0 0 24 24"
                                  >
                                    <path d="M20.317 4.37a19.791 19.791 0 00-4.885-1.515.074.074 0 00-.079.037c-.21.375-.444.864-.607 1.25a18.27 18.27 0 00-5.487 0c-.163-.386-.397-.875-.61-1.25a.077.077 0 00-.079-.037A19.736 19.736 0 003.677 4.37a.07.07 0 00-.032.027C.533 9.046-.32 13.58.099 18.057a.082.082 0 00.031.056 19.9 19.9 0 005.993 3.03.078.078 0 00.084-.028c.462-.63.874-1.295 1.226-1.994a.076.076 0 00-.042-.106 13.107 13.107 0 01-1.872-.892.077.077 0 00-.008-.128 10.713 10.713 0 00.372-.294.075.075 0 00.03-.066c.329-.246.648-.5.954-.76a.07.07 0 00.076-.01 13.697 13.697 0 0011.086 0 .07.07 0 00.076.009c.305.26.625.514.954.759a.077.077 0 00.03.067c.12.088.246.177.371.294a.077.077 0 00-.006.127 13.227 13.227 0 01-1.873.892.076.076 0 00-.041.107c.352.699.764 1.364 1.225 1.994a.076.076 0 00.084.028 19.963 19.963 0 006.002-3.03.077.077 0 00.032-.054c.5-4.817-.838-9.033-3.55-12.765a.061.061 0 00-.031-.03zM8.02 15.33c-1.183 0-2.157-.969-2.157-2.156 0-1.193.974-2.157 2.157-2.157 1.193 0 2.156.964 2.156 2.157 0 1.187-.963 2.156-2.156 2.156zm7.975 0c-1.183 0-2.157-.969-2.157-2.156 0-1.193.974-2.157 2.157-2.157 1.193 0 2.157.964 2.157 2.157 0 1.187-.964 2.156-2.157 2.156z" />
                                  </svg>
                                  <span className="text-xs font-medium text-foreground">
                                    DM
                                  </span>
                                </>
                              )}
                              {dest.type === "discord_channel" && (
                                <>
                                  <svg
                                    className="w-3 h-3 text-indigo-500"
                                    fill="currentColor"
                                    viewBox="0 0 24 24"
                                  >
                                    <path d="M20.317 4.37a19.791 19.791 0 00-4.885-1.515.074.074 0 00-.079.037c-.21.375-.444.864-.607 1.25a18.27 18.27 0 00-5.487 0c-.163-.386-.397-.875-.61-1.25a.077.077 0 00-.079-.037A19.736 19.736 0 003.677 4.37a.07.07 0 00-.032.027C.533 9.046-.32 13.58.099 18.057a.082.082 0 00.031.056 19.9 19.9 0 005.993 3.03.078.078 0 00.084-.028c.462-.63.874-1.295 1.226-1.994a.076.076 0 00-.042-.106 13.107 13.107 0 01-1.872-.892.077.077 0 00-.008-.128 10.713 10.713 0 00.372-.294.075.075 0 00.03-.066c.329-.246.648-.5.954-.76a.07.07 0 00.076-.01 13.697 13.697 0 0011.086 0 .07.07 0 00.076.009c.305.26.625.514.954.759a.077.077 0 00.03.067c.12.088.246.177.371.294a.077.077 0 00-.006.127 13.227 13.227 0 01-1.873.892.076.076 0 00-.041.107c.352.699.764 1.364 1.225 1.994a.076.076 0 00.084.028 19.963 19.963 0 006.002-3.03.077.077 0 00.032-.054c.5-4.817-.838-9.033-3.55-12.765a.061.061 0 00-.031-.03zM8.02 15.33c-1.183 0-2.157-.969-2.157-2.156 0-1.193.974-2.157 2.157-2.157 1.193 0 2.156.964 2.156 2.157 0 1.187-.963 2.156-2.156 2.156zm7.975 0c-1.183 0-2.157-.969-2.157-2.156 0-1.193.974-2.157 2.157-2.157 1.193 0 2.157.964 2.157 2.157 0 1.187-.964 2.156-2.157 2.156z" />
                                  </svg>
                                  <span className="text-xs font-medium text-foreground">
                                    Channel
                                  </span>
                                </>
                              )}
                              {dest.type === "webhook" && (
                                <>
                                  <svg
                                    className="w-3 h-3 text-orange-500"
                                    fill="currentColor"
                                    viewBox="0 0 24 24"
                                  >
                                    <path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm-2 15l-5-5 1.41-1.41L10 14.17l7.59-7.59L19 8l-9 9z" />
                                  </svg>
                                  <span className="text-xs font-medium text-foreground">
                                    Webhook
                                  </span>
                                </>
                              )}
                              {dest.type === "email" && (
                                <>
                                  <svg
                                    className="w-3 h-3 text-accent"
                                    fill="none"
                                    stroke="currentColor"
                                    strokeWidth="2"
                                    viewBox="0 0 24 24"
                                  >
                                    <path strokeLinecap="round" strokeLinejoin="round" d="M3 8l7.89 5.26a2 2 0 002.22 0L21 8M5 19h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z" />
                                  </svg>
                                  <span className="text-xs font-medium text-foreground">
                                    Email
                                  </span>
                                </>
                              )}
                              {dest.type === "android_push" && (
                                <>
                                  <svg
                                    className="w-3 h-3 text-emerald-500"
                                    fill="none"
                                    stroke="currentColor"
                                    strokeWidth="2"
                                    viewBox="0 0 24 24"
                                  >
                                    <rect x="5" y="2" width="14" height="20" rx="2" ry="2" />
                                    <line x1="12" y1="18" x2="12.01" y2="18" />
                                  </svg>
                                  <span className="text-xs font-medium text-foreground">
                                    Push
                                  </span>
                                </>
                              )}
                            </div>
                          ))}
                        </div>
                      )}
                  </div>
                </CardContent>
              </Card>
            );
          })
        ) : (
          <Card className="border-border bg-card/60 backdrop-blur text-center py-8">
            <p className="text-muted-foreground">{t("reminders.noResults")}</p>
          </Card>
        )}
      </div>

      {/* Pagination Controls */}
      {filteredReminders.length > 0 && totalPages > 1 && (
        <div className="flex items-center justify-between pt-4 border-t border-border/50">
          <p className="text-xs text-muted-foreground">
            {t("reminders.pagination", {
              current: currentPage,
              total: totalPages,
            })}
          </p>
          <div className="flex gap-2">
            <Button
              onClick={() => setCurrentPage((p) => Math.max(1, p - 1))}
              disabled={currentPage === 1}
              variant="outline"
              size="sm"
              className="gap-1"
            >
              <ChevronLeft className="w-4 h-4" />
              {t("reminders.previous")}
            </Button>
            <Button
              onClick={() => setCurrentPage((p) => Math.min(totalPages, p + 1))}
              disabled={currentPage === totalPages}
              variant="outline"
              size="sm"
              className="gap-1"
            >
              {t("reminders.next")}
              <ChevronRight className="w-4 h-4" />
            </Button>
          </div>
        </div>
      )}
    </div>
  );
}
