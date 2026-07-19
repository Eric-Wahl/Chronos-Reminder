import { useState } from "react";
import { useTranslation } from "react-i18next";
import { useNavigate } from "react-router-dom";
import { LanguageSwitcher } from "@/components/common/language-switcher";
import { ModeToggle } from "@/components/common/mode-toggle";
import { useToast } from "@/hooks/useToast";
import { authService } from "@/services";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";

export function ForgotPasswordPage() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const { success, error: showError } = useToast();

  const [email, setEmail] = useState("");
  const [isLoading, setIsLoading] = useState(false);
  const [submitted, setSubmitted] = useState(false);
  const [localError, setLocalError] = useState<string | null>(null);

  const handleSubmit = async (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    e.stopPropagation();
    setLocalError(null);

    if (!email.trim()) {
      setLocalError(t("common.emailRequired"));
      return;
    }

    setIsLoading(true);

    try {
      await authService.requestPasswordReset({ email });

      // Show success message
      success(t("forgotPassword.emailSent") as string);
      setSubmitted(true);

      // Redirect to login after 3 seconds
      setTimeout(() => {
        navigate("/login", { replace: true });
      }, 3000);
    } catch (err) {
      const errorMsg =
        err instanceof Error ? err.message : t("forgotPassword.error");
      setLocalError(errorMsg);
      showError(errorMsg);
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <div className="min-h-screen bg-gradient-to-br from-background-main to-background-secondary flex items-center justify-center p-4 relative">
      {/* Background decorative elements */}
      <div className="absolute inset-0 overflow-hidden pointer-events-none">
        <div className="absolute top-0 right-0 w-72 h-72 bg-accent/10 rounded-full blur-3xl dark:bg-accent/5"></div>
        <div className="absolute bottom-0 left-0 w-96 h-96 bg-accent/10 rounded-full blur-3xl dark:bg-accent/5"></div>
      </div>

      {/* Theme and Language Controls */}
      <div className="absolute top-4 right-4 z-50 flex gap-2">
        <LanguageSwitcher />
        <ModeToggle />
      </div>

      {/* Back to Login Button */}
      <button
        onClick={() => navigate("/login")}
        className="absolute top-4 left-4 z-50 px-4 py-2 rounded-md text-foreground dark:text-white hover:text-amber-600 dark:hover:text-amber-400 hover:bg-amber-400/10 transition-colors border border-border dark:border-white/10"
      >
        ← {t("common.back") || "Back"}
      </button>

      <div className="w-full max-w-md relative z-10">
        <div className="flex flex-col justify-center items-center space-y-8">
          {/* Header */}
          <div className="text-center">
            <div className="flex items-center justify-center gap-3 mb-6">
              <img
                src="/logo_chronos.png"
                alt="Chronos Logo"
                className="w-12 h-12 rounded-full"
              />
              <span className="text-3xl font-bold text-accent">Chronos</span>
            </div>
            <h1 className="text-3xl font-bold text-foreground mb-2">
              {t("forgotPassword.title")}
            </h1>
            <p className="text-muted-foreground">
              {t("forgotPassword.description")}
            </p>
          </div>

          {/* Form Card */}
          {!submitted ? (
            <Card className="w-full border-border bg-card/95 backdrop-blur">
              <CardHeader className="space-y-1">
                <CardTitle className="text-foreground">
                  {t("forgotPassword.enterEmail")}
                </CardTitle>
                <CardDescription>
                  {t("forgotPassword.weWillSendLink")}
                </CardDescription>
              </CardHeader>
              <CardContent>
                <form onSubmit={handleSubmit} className="space-y-4">
                  {/* Error Alert */}
                  {localError && (
                    <div className="bg-red-500/10 border border-red-500 rounded-md p-3 text-sm text-red-600">
                      {localError}
                    </div>
                  )}

                  {/* Email Field */}
                  <div className="space-y-2">
                    <label
                      htmlFor="email"
                      className="text-sm font-medium text-foreground"
                    >
                      {t("common.emailAddress")}
                    </label>
                    <Input
                      id="email"
                      type="email"
                      placeholder="timely@yours.com"
                      value={email}
                      onChange={(e) => setEmail(e.target.value)}
                      className="bg-secondary/50 border-border text-foreground placeholder:text-muted-foreground"
                      required
                      disabled={isLoading}
                    />
                  </div>

                  {/* Submit Button */}
                  <Button
                    type="submit"
                    disabled={isLoading}
                    className="w-full bg-accent hover:bg-accent/90 disabled:opacity-50 disabled:cursor-not-allowed text-accent-foreground font-semibold mt-6"
                  >
                    {isLoading
                      ? t("forgotPassword.sending")
                      : t("forgotPassword.sendResetLink")}
                  </Button>
                </form>

                {/* Back to Login Link */}
                <p className="text-center text-muted-foreground text-sm mt-6">
                  {t("forgotPassword.rememberPassword")}{" "}
                  <button
                    type="button"
                    onClick={() => navigate("/login")}
                    className="text-accent hover:text-accent/80 transition-colors font-medium"
                  >
                    {t("common.signIn")}
                  </button>
                </p>

                {/* Discord-only accounts hint */}
                <p className="text-center text-muted-foreground text-xs mt-4 border-t border-border pt-4">
                  {t("forgotPassword.discordHint")}
                </p>
              </CardContent>
            </Card>
          ) : (
            /* Success Message */
            <Card className="w-full border-border bg-card/95 backdrop-blur border-green-500/30">
              <CardContent className="pt-6">
                <div className="text-center space-y-4">
                  <div className="flex justify-center">
                    <div className="w-16 h-16 rounded-full bg-green-500/20 flex items-center justify-center">
                      <span className="text-3xl">✓</span>
                    </div>
                  </div>
                  <h2 className="text-lg font-semibold text-foreground">
                    {t("forgotPassword.checkYourEmail")}
                  </h2>
                  <p className="text-muted-foreground">
                    {t("forgotPassword.linkSentTo")} <strong>{email}</strong>
                  </p>
                  <p className="text-sm text-muted-foreground">
                    {t("forgotPassword.linkExpires")}
                  </p>
                  <Button
                    onClick={() => navigate("/login")}
                    className="w-full bg-accent hover:bg-accent/90 text-accent-foreground font-semibold mt-4"
                  >
                    {t("forgotPassword.backToLogin")}
                  </Button>
                </div>
              </CardContent>
            </Card>
          )}
        </div>
      </div>
    </div>
  );
}
