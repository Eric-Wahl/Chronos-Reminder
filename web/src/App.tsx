import { BrowserRouter, Routes, Route, Navigate } from "react-router-dom";
import { Toaster } from "sonner";
import { ThemeProvider } from "./components/common/theme-provider";
import { AuthProvider } from "./hooks/AuthContext";
import { HomePage } from "./pages/HomePage";
import { LoginPage } from "./pages/LoginPage";
import { VerificationPage } from "./pages/VerificationPage";
import { ForgotPasswordPage } from "./pages/ForgotPasswordPage";
import { ResetPasswordPage } from "./pages/ResetPasswordPage";
import { RemindersPage } from "./pages/RemindersPage";
import { CreateReminderPage } from "./pages/CreateReminderPage";
import { ReminderDetailsPage } from "./pages/ReminderDetailsPage";
import { DontForgetMePage } from "./pages/DontForgetMePage";
import { DayPlannerPage } from "./pages/DayPlannerPage";
import { AccountPage } from "./pages/AccountPage";
import { APIKeysPage } from "./pages/APIKeysPage";
import { OAuthCallbackPage } from "./pages/OAuthCallbackPage";
import { ChangelogPage } from "./pages/ChangelogPage";
import { ContactPage } from "./pages/ContactPage";
import { SelfHostPage } from "./pages/SelfHostPage";
import { TermsPage } from "./pages/TermsPage";
import { useAuth } from "./hooks/useAuth";
import { ROUTES } from "./config/routes";
import "./i18n/config";

function AppRoutes() {
  const { isAuthenticated, isCheckingAuth } = useAuth();

  // Don't render routes while checking initial auth status
  // isCheckingAuth is only true during the initial mount auth check
  if (isCheckingAuth) {
    return (
      <div className="flex items-center justify-center min-h-screen">
        <div className="animate-pulse">Loading...</div>
      </div>
    );
  }

  return (
    <Routes>
      {/* Email Verification route: Public route for email verification */}
      <Route path={ROUTES.VERIFY_EMAIL.path} element={<VerificationPage />} />

      {/* Forgot Password route: Public route for password reset request */}
      <Route
        path={ROUTES.FORGOT_PASSWORD.path}
        element={<ForgotPasswordPage />}
      />

      {/* Reset Password route: Public route for password reset with token */}
      <Route
        path={ROUTES.RESET_PASSWORD.path}
        element={<ResetPasswordPage />}
      />

      {/* OAuth Callback route: Public route for Discord OAuth callback */}
      <Route
        path={ROUTES.AUTH_CALLBACK_DISCORD.path}
        element={<OAuthCallbackPage />}
      />

      {/* Changelog route: Public route to view changelog */}
      <Route path={ROUTES.CHANGELOG.path} element={<ChangelogPage />} />

      {/* Self-Host Guide route: Public route for self-hosting documentation */}
      <Route path={ROUTES.SELFHOST.path} element={<SelfHostPage />} />

      {/* Contact route: Public route for contact form */}
      <Route path={ROUTES.CONTACT.path} element={<ContactPage />} />

      {/* Terms & Privacy route: Public route for terms and privacy policy */}
      <Route path={ROUTES.TERMS.path} element={<TermsPage />} />

      {/* Vitrine route: Public route at root for all users */}
      <Route path={ROUTES.HOME.path} element={<HomePage />} />

      {/* Protected route: Dashboard page requires authentication */}
      <Route
        path={ROUTES.REMINDERS.path}
        element={
          isAuthenticated ? (
            <RemindersPage />
          ) : (
            <Navigate to={ROUTES.HOME.path} replace />
          )
        }
      />

      {/* Protected route: Create Reminder page requires authentication */}
      <Route
        path={ROUTES.REMINDERS_CREATE.path}
        element={
          isAuthenticated ? (
            <CreateReminderPage />
          ) : (
            <Navigate to={ROUTES.HOME.path} replace />
          )
        }
      />

      {/* Protected route: Reminder Details page requires authentication */}
      <Route
        path={ROUTES.REMINDER_DETAILS.path}
        element={
          isAuthenticated ? (
            <ReminderDetailsPage />
          ) : (
            <Navigate to={ROUTES.HOME.path} replace />
          )
        }
      />

      {/* Protected route: Don't Forget Me page requires authentication */}
      <Route
        path={ROUTES.DONT_FORGET_ME.path}
        element={
          isAuthenticated ? (
            <DontForgetMePage />
          ) : (
            <Navigate to={ROUTES.HOME.path} replace />
          )
        }
      />

      {/* Protected route: Day Planner page requires authentication */}
      <Route
        path={ROUTES.DAY_PLANNER.path}
        element={
          isAuthenticated ? (
            <DayPlannerPage />
          ) : (
            <Navigate to={ROUTES.HOME.path} replace />
          )
        }
      />

      {/* Protected route: Account page requires authentication */}
      <Route
        path={ROUTES.ACCOUNT.path}
        element={
          isAuthenticated ? (
            <AccountPage />
          ) : (
            <Navigate to={ROUTES.HOME.path} replace />
          )
        }
      />

      {/* Protected route: API Keys page requires authentication */}
      <Route
        path={ROUTES.API_KEYS.path}
        element={
          isAuthenticated ? (
            <APIKeysPage />
          ) : (
            <Navigate to={ROUTES.HOME.path} replace />
          )
        }
      />

      {/* Login route: Redirect to dashboard if already authenticated */}
      <Route
        path={ROUTES.LOGIN.path}
        element={
          isAuthenticated ? (
            <Navigate to={ROUTES.REMINDERS.path} replace />
          ) : (
            <LoginPage />
          )
        }
      />

      {/* Catch all: Redirect to vitrine by default */}
      <Route
        path="*"
        element={
          <Navigate
            to={isAuthenticated ? ROUTES.REMINDERS.path : ROUTES.HOME.path}
            replace
          />
        }
      />
    </Routes>
  );
}

function App() {
  return (
    <ThemeProvider defaultTheme="dark" storageKey="vite-ui-theme">
      <AuthProvider>
        <BrowserRouter>
          <AppRoutes />
          <Toaster richColors theme="dark" position="top-right" expand />
        </BrowserRouter>
      </AuthProvider>
    </ThemeProvider>
  );
}

export default App;
