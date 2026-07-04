import { Plus, Bell, CheckCircle2, Clock } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Header } from "@/components/common/header";
import { Calendar } from "@/components/Calendar";
import { RemindersList } from "@/components/RemindersList";
import { useTranslation } from "react-i18next";
import { useNavigate } from "react-router-dom";
import { useEffect, useState } from "react";
import {
  remindersService,
  accountService,
  type Reminder,
  type Account,
} from "@/services";
import { Footer } from "@/components/common/footer";
import { useToast } from "@/hooks/useToast";
import { MAX_REMINDERS_PER_ACCOUNT } from "@/lib/limits";

export function RemindersPage() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const toast = useToast();
  const [reminders, setReminders] = useState<Reminder[]>([]);
  const [account, setAccount] = useState<Account | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Fetch reminders and account data
  useEffect(() => {
    const fetchData = async () => {
      try {
        setIsLoading(true);
        setError(null);

        // Fetch reminders and account in parallel
        const [fetchedReminders, fetchedAccount] = await Promise.all([
          remindersService.getReminders(),
          accountService.getAccount(),
        ]);

        setReminders(fetchedReminders);
        setAccount(fetchedAccount);
      } catch (err) {
        console.error("Failed to fetch data:", err);
        setError(err instanceof Error ? err.message : "Failed to fetch data");
      } finally {
        setIsLoading(false);
      }
    };

    fetchData();
  }, []);

  // Calculate statistics
  const totalReminders = reminders.length;
  const activeReminders = reminders.filter((r) => {
    const reminderDate = new Date(r.remind_at_utc);
    return reminderDate > new Date();
  }).length;

  const reminderLimitReached = totalReminders >= MAX_REMINDERS_PER_ACCOUNT;

  const handleAddReminder = () => {
    if (reminderLimitReached) {
      toast.error(t("reminderLimit.reachedTitle"), {
        description: t("reminderLimit.reachedDesc", {
          max: MAX_REMINDERS_PER_ACCOUNT,
        }),
      });
      return;
    }
    navigate("/reminders/create");
  };

  return (
    <div className="min-h-screen bg-background-main dark:bg-background-main">
      <Header />

      {/* Main Content */}
      <main className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-12 pt-24">
        {/* Welcome Section */}
        <div className="mb-12">
          <div className="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-4 mb-4">
            <div>
              <h2 className="text-3xl sm:text-4xl font-bold text-foreground">
                {t("welcome.title")}
              </h2>
              <p className="text-muted-foreground text-base sm:text-lg mt-2">
                {t("welcome.subtitle")}
              </p>
            </div>
            <Button
              onClick={handleAddReminder}
              disabled={reminderLimitReached}
              title={
                reminderLimitReached
                  ? t("reminderLimit.reachedDesc", {
                      max: MAX_REMINDERS_PER_ACCOUNT,
                    })
                  : undefined
              }
              className="bg-accent hover:bg-accent/90 text-accent-foreground font-semibold w-full sm:w-auto gap-2 disabled:opacity-50 disabled:cursor-not-allowed"
            >
              <Plus className="w-4 h-4" />
              {t("welcome.newReminder")}
            </Button>
          </div>
        </div>

        {/* Error State */}
        {error && (
          <Card className="border-red-500/50 bg-red-500/10 backdrop-blur mb-6">
            <CardContent className="pt-6">
              <p className="text-red-600 dark:text-red-400">{error}</p>
            </CardContent>
          </Card>
        )}

        {/* Loading State */}
        {isLoading ? (
          <Card className="border-border bg-card/95 backdrop-blur text-center py-12">
            <Clock className="w-12 h-12 text-muted-foreground mx-auto mb-4 animate-spin" />
            <p className="text-muted-foreground">Loading your reminders...</p>
          </Card>
        ) : (
          <>
            {/* Mobile Reminders List - shown on small screens before overview */}
            <div className="lg:hidden mb-12">
              <RemindersList
                reminders={reminders}
                onAddReminder={handleAddReminder}
              />
            </div>

            {/* Account Overview Cards */}
            {totalReminders > 0 && (
              <div className="grid grid-cols-1 md:grid-cols-3 gap-4 mb-12">
                {/* Total Reminders Card */}
                <Card className="border-border bg-card/95 backdrop-blur hover:border-accent/50 transition-colors">
                  <CardHeader className="pb-3">
                    <CardTitle className="text-sm font-medium text-muted-foreground flex items-center gap-2">
                      <Bell className="w-4 h-4" />
                      {t("overview.totalReminders")}
                    </CardTitle>
                  </CardHeader>
                  <CardContent>
                    <div className="text-3xl font-bold text-foreground">
                      {totalReminders}
                      {totalReminders >= MAX_REMINDERS_PER_ACCOUNT * 0.8 && (
                        <span className="text-sm font-normal text-muted-foreground">
                          {" "}
                          / {MAX_REMINDERS_PER_ACCOUNT}
                        </span>
                      )}
                    </div>
                    <p className="text-xs text-muted-foreground mt-1">
                      {t("overview.active")}: {activeReminders}
                    </p>
                  </CardContent>
                </Card>

                {/* Active Reminders Card */}
                <Card className="border-border bg-card/95 backdrop-blur hover:border-accent/50 transition-colors">
                  <CardHeader className="pb-3">
                    <CardTitle className="text-sm font-medium text-muted-foreground flex items-center gap-2">
                      <Clock className="w-4 h-4 text-accent" />
                      {t("overview.activeReminders")}
                    </CardTitle>
                  </CardHeader>
                  <CardContent>
                    <div className="text-3xl font-bold text-accent">
                      {activeReminders}
                    </div>
                    <p className="text-xs text-muted-foreground mt-1">
                      {t("overview.accountStatus")}:{" "}
                      <span className="text-accent font-semibold">
                        {t("overview.active")}
                      </span>
                    </p>
                  </CardContent>
                </Card>

                {/* Timezone Card */}
                <Card className="border-border bg-card/95 backdrop-blur hover:border-accent/50 transition-colors">
                  <CardHeader className="pb-3">
                    <CardTitle className="text-sm font-medium text-muted-foreground flex items-center gap-2">
                      <CheckCircle2 className="w-4 h-4" />
                      {t("overview.timezone")}
                    </CardTitle>
                  </CardHeader>
                  <CardContent>
                    <div className="text-3xl font-bold text-foreground">
                      {account?.timezone || "UTC"}
                    </div>
                    <p className="text-xs text-muted-foreground mt-1">
                      {new Date().toLocaleDateString()}
                    </p>
                  </CardContent>
                </Card>
              </div>
            )}

            {/* Layout: Calendar on left (desktop only), Reminders list on right (desktop only) */}
            <div className="grid grid-cols-1 lg:grid-cols-3 gap-8">
              {/* Calendar Section - Hidden on mobile */}
              <div className="lg:col-span-2 hidden lg:block">
                <Calendar
                  reminders={reminders}
                  onAddReminder={handleAddReminder}
                />
              </div>

              {/* Reminders List Section - Hidden on mobile */}
              <div className="lg:col-span-1 hidden lg:block">
                <RemindersList
                  reminders={reminders}
                  onAddReminder={handleAddReminder}
                />
              </div>
            </div>
          </>
        )}
      </main>

      {/* Footer */}
      <Footer />
    </div>
  );
}
