import { useTranslation } from "react-i18next";
import { Globe } from "lucide-react";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { useDisplayTimezone } from "@/hooks/useDisplayTimezone";
import { formatTimeInTimezone } from "@/lib/timezone";

/**
 * Shown once, the first time a signed-in user's device timezone is found to
 * differ from their account's. Picking either option (or dismissing it)
 * stores the choice, so it never comes back.
 */
export function DisplayTimezoneNotice() {
  const { t } = useTranslation();
  const {
    needsChoice,
    accountTimezone,
    deviceTimezone,
    setUseDevice,
  } = useDisplayTimezone();

  const now = new Date();

  return (
    <Dialog
      open={needsChoice}
      onOpenChange={(open) => {
        // Dismissing without picking keeps the account timezone, and counts as an answer
        if (!open) setUseDevice(false);
      }}
    >
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <Globe className="w-5 h-5 text-accent" />
            {t("timezoneNotice.title")}
          </DialogTitle>
          <DialogDescription>
            {t("timezoneNotice.description", {
              deviceTimezone,
              accountTimezone,
            })}
          </DialogDescription>
        </DialogHeader>

        <div className="rounded-md border border-border bg-muted/30 px-3 py-2 text-sm">
          <p className="text-foreground">
            {t("timezoneNotice.example", {
              deviceTime: formatTimeInTimezone(now, deviceTimezone),
              accountTime: formatTimeInTimezone(now, accountTimezone),
            })}
          </p>
        </div>

        <p className="text-sm text-muted-foreground">
          {t("timezoneNotice.hint")}
        </p>

        <DialogFooter className="gap-2 sm:gap-2">
          <Button variant="outline" onClick={() => setUseDevice(false)}>
            {t("timezoneNotice.keepAccount", { timezone: accountTimezone })}
          </Button>
          <Button
            className="bg-accent hover:bg-accent/90 text-accent-foreground font-semibold"
            onClick={() => setUseDevice(true)}
          >
            {t("timezoneNotice.useDevice", { timezone: deviceTimezone })}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
