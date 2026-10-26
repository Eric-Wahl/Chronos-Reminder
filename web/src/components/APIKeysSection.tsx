import { useState, useEffect } from "react";
import { useTranslation } from "react-i18next";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Alert } from "@/components/ui/alert";
import {
  Copy,
  Eye,
  EyeOff,
  Key,
  Loader2,
  Plus,
  Trash2,
  AlertCircle,
  CheckCircle2,
} from "lucide-react";
import { apiKeyService, type APIKey } from "@/services";
import { useToast } from "@/hooks/useToast";
import { useDisplayTimezone } from "@/hooks/useDisplayTimezone";
import { formatDateInTimezone } from "@/lib/timezone";

interface APIKeyWithPlain extends APIKey {
  plainKey?: string;
  showPlain?: boolean;
}

export function APIKeysSection() {
  const { t } = useTranslation();
  const { displayTimezone } = useDisplayTimezone();
  const toast = useToast();

  // API Keys state
  const [apiKeys, setApiKeys] = useState<APIKeyWithPlain[]>([]);
  const [isLoadingKeys, setIsLoadingKeys] = useState(true);
  const [keysError, setKeysError] = useState<string | null>(null);

  // Create new key state
  const [newKeyName, setNewKeyName] = useState("");
  const [isCreatingKey, setIsCreatingKey] = useState(false);
  const [createError, setCreateError] = useState<string | null>(null);
  const [createdKey, setCreatedKey] = useState<APIKeyWithPlain | null>(null);
  const [copiedKeyId, setCopiedKeyId] = useState<string | null>(null);

  // Delete state
  const [deletingKeyId, setDeletingKeyId] = useState<string | null>(null);
  const [deleteError, setDeleteError] = useState<string | null>(null);

  // Fetch API keys
  useEffect(() => {
    const fetchKeys = async () => {
      try {
        setIsLoadingKeys(true);
        setKeysError(null);
        const keys = await apiKeyService.listAPIKeys();
        setApiKeys(keys);
      } catch (err) {
        const errorMsg =
          err instanceof Error ? err.message : "Failed to load API keys";
        setKeysError(errorMsg);
        console.error("Failed to fetch API keys:", err);
      } finally {
        setIsLoadingKeys(false);
      }
    };

    fetchKeys();
  }, []);

  // Handle create API key
  const handleCreateKey = async (e: React.FormEvent) => {
    e.preventDefault();

    if (!newKeyName.trim()) {
      setCreateError(t("apiKeys.nameRequired") || "Key name is required");
      return;
    }

    try {
      setIsCreatingKey(true);
      setCreateError(null);

      const response = await apiKeyService.createAPIKey(newKeyName);

      // Add to list
      const keyWithPlain: APIKeyWithPlain = {
        ...response,
        plainKey: response.key,
        showPlain: true,
      };

      setApiKeys([keyWithPlain, ...apiKeys]);
      setCreatedKey(keyWithPlain);
      setNewKeyName("");

      toast.success(t("apiKeys.keyCreated") || "API Key created", {
        description:
          t("apiKeys.keyCreatedDesc") ||
          "Keep it safe, you won't see it again!",
      });
    } catch (err) {
      const errorMsg =
        err instanceof Error ? err.message : "Failed to create API key";
      setCreateError(errorMsg);
      toast.error(t("apiKeys.error") || "Error", {
        description: errorMsg,
      });
    } finally {
      setIsCreatingKey(false);
    }
  };

  // Handle copy key
  const handleCopyKey = async (keyId: string, plainKey?: string) => {
    if (!plainKey) return;

    try {
      await navigator.clipboard.writeText(plainKey);
      setCopiedKeyId(keyId);

      toast.success(t("common.copied") || "Copied", {
        description: t("apiKeys.keyCopied") || "API key copied to clipboard",
      });

      setTimeout(() => setCopiedKeyId(null), 2000);
    } catch (err) {
      console.error("Failed to copy:", err);
      toast.error(t("common.error") || "Error", {
        description: t("common.copyFailed") || "Failed to copy to clipboard",
      });
    }
  };

  // Handle toggle show/hide key
  const handleToggleShowKey = (keyId: string) => {
    setApiKeys(
      apiKeys.map((key) =>
        key.id === keyId ? { ...key, showPlain: !key.showPlain } : key
      )
    );
  };

  // Handle revoke key
  const handleRevokeKey = async (keyId: string) => {
    try {
      setDeletingKeyId(keyId);
      setDeleteError(null);

      await apiKeyService.revokeAPIKey(keyId);

      // Remove from list
      setApiKeys(apiKeys.filter((key) => key.id !== keyId));
      if (createdKey?.id === keyId) {
        setCreatedKey(null);
      }

      toast.success(t("apiKeys.keyRevoked") || "API Key revoked", {
        description:
          t("apiKeys.keyRevokedDesc") || "The API key has been deleted",
      });
    } catch (err) {
      const errorMsg =
        err instanceof Error ? err.message : "Failed to revoke API key";
      setDeleteError(errorMsg);
      toast.error(t("apiKeys.error") || "Error", {
        description: errorMsg,
      });
    } finally {
      setDeletingKeyId(null);
    }
  };

  return (
    <div className="space-y-6">
      {/* Create New API Key Card */}
      <Card className="border-border bg-card/95 backdrop-blur">
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Key className="w-5 h-5 text-accent" />
            {t("apiKeys.createNew") || "Create New API Key"}
          </CardTitle>
          <CardDescription>
            {t("apiKeys.createDesc") ||
              "Generate a new API key for third-party integrations"}
          </CardDescription>
        </CardHeader>
        <CardContent>
          <form onSubmit={handleCreateKey} className="space-y-4">
            {createError && (
              <Alert className="border-red-500/50 bg-red-500/10">
                <AlertCircle className="h-4 w-4 text-red-600 dark:text-red-400" />
                <div className="ml-3">
                  <p className="text-red-600 dark:text-red-400">
                    {createError}
                  </p>
                </div>
              </Alert>
            )}

            <div>
              <Label htmlFor="key-name" className="text-foreground">
                {t("apiKeys.keyName") || "Key Name"}
              </Label>
              <div className="mt-2 flex gap-2">
                <Input
                  id="key-name"
                  placeholder={
                    t("apiKeys.keyNamePlaceholder") || "e.g., Production Server"
                  }
                  value={newKeyName}
                  onChange={(e) => setNewKeyName(e.target.value)}
                  disabled={isCreatingKey || apiKeys.length >= 5}
                  className="flex-1"
                />
                <Button
                  type="submit"
                  disabled={isCreatingKey || apiKeys.length >= 5}
                  className="bg-accent hover:bg-accent/90 text-accent-foreground font-semibold gap-2"
                >
                  {isCreatingKey ? (
                    <>
                      <Loader2 className="w-4 h-4 animate-spin" />
                      {t("common.creating") || "Creating..."}
                    </>
                  ) : (
                    <>
                      <Plus className="w-4 h-4" />
                      {t("apiKeys.createButton") || "Create Key"}
                    </>
                  )}
                </Button>
              </div>
              {apiKeys.length >= 5 && (
                <p className="text-xs text-amber-600 dark:text-amber-400 mt-2">
                  {t("apiKeys.maxKeysReached") ||
                    "Maximum of 5 API keys per account"}
                </p>
              )}
            </div>
          </form>
        </CardContent>
      </Card>

      {/* Recently Created Key (Shown Once) */}
      {createdKey && (
        <Card className="border-green-500/30 bg-green-500/5 backdrop-blur">
          <CardHeader>
            <CardTitle className="flex items-center gap-2 text-green-600 dark:text-green-400">
              <CheckCircle2 className="w-5 h-5" />
              {t("apiKeys.newKeyCreated") || "New API Key Created"}
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="p-4 bg-green-500/10 rounded-lg border border-green-500/20">
              <p className="text-sm text-muted-foreground mb-3">
                {t("apiKeys.copyWarning") ||
                  "Copy your API key now. You won't be able to see it again!"}
              </p>
              <div className="flex items-center gap-2">
                <div className="flex-1 p-2 bg-background rounded border border-border font-mono text-sm break-all">
                  {createdKey.plainKey}
                </div>
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  onClick={() =>
                    handleCopyKey(createdKey.id, createdKey.plainKey)
                  }
                  className="flex-shrink-0"
                >
                  {copiedKeyId === createdKey.id ? (
                    <CheckCircle2 className="w-4 h-4 text-green-600" />
                  ) : (
                    <Copy className="w-4 h-4" />
                  )}
                </Button>
              </div>
            </div>
          </CardContent>
        </Card>
      )}

      {/* API Keys List */}
      <Card className="border-border bg-card/95 backdrop-blur">
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Key className="w-5 h-5 text-accent" />
            {t("apiKeys.yourKeys") || "Your API Keys"}
          </CardTitle>
          <CardDescription>
            {t("apiKeys.keysCount", { count: apiKeys.length }) ||
              `${apiKeys.length} active key(s)`}
          </CardDescription>
        </CardHeader>
        <CardContent>
          {keysError && (
            <Alert className="border-red-500/50 bg-red-500/10 mb-4">
              <AlertCircle className="h-4 w-4 text-red-600 dark:text-red-400" />
              <div className="ml-3">
                <p className="text-red-600 dark:text-red-400">{keysError}</p>
              </div>
            </Alert>
          )}

          {deleteError && (
            <Alert className="border-red-500/50 bg-red-500/10 mb-4">
              <AlertCircle className="h-4 w-4 text-red-600 dark:text-red-400" />
              <div className="ml-3">
                <p className="text-red-600 dark:text-red-400">{deleteError}</p>
              </div>
            </Alert>
          )}

          {isLoadingKeys ? (
            <div className="flex items-center justify-center py-8">
              <Loader2 className="w-6 h-6 text-muted-foreground animate-spin" />
              <p className="text-muted-foreground ml-3">
                {t("common.loading") || "Loading..."}
              </p>
            </div>
          ) : apiKeys.length === 0 ? (
            <div className="py-8 text-center">
              <Key className="w-12 h-12 text-muted-foreground/30 mx-auto mb-3" />
              <p className="text-muted-foreground">
                {t("apiKeys.noKeys") || "No API keys created yet"}
              </p>
              <p className="text-sm text-muted-foreground/75">
                {t("apiKeys.noKeysDesc") ||
                  "Create one to get started with the API"}
              </p>
            </div>
          ) : (
            <div className="space-y-3">
              {apiKeys.map((key) => (
                <div
                  key={key.id}
                  className="p-4 bg-secondary/20 rounded-lg border border-border hover:bg-secondary/30 transition-colors"
                >
                  <div className="flex items-center gap-3 mb-3">
                    <div className="flex-1 min-w-0">
                      <p className="font-semibold text-foreground truncate">
                        {key.name}
                      </p>
                      <p className="text-xs text-muted-foreground">
                        {t("apiKeys.created")}{" "}
                        {formatDateInTimezone(new Date(key.created_at), displayTimezone)}
                      </p>
                    </div>
                    <span className="inline-flex items-center px-2 py-1 rounded-full text-xs font-medium bg-accent/20 text-accent whitespace-nowrap">
                      {key.scopes}
                    </span>
                  </div>

                  {/* Key Display with Copy/Reveal */}
                  {key.plainKey ? (
                    <div className="flex items-center gap-2 mb-3">
                      <div className="flex-1 p-2 bg-background/50 rounded border border-border font-mono text-xs break-all">
                        {key.showPlain
                          ? key.plainKey
                          : apiKeyService.maskAPIKey(key.plainKey)}
                      </div>
                      <Button
                        type="button"
                        variant="ghost"
                        size="sm"
                        onClick={() => handleToggleShowKey(key.id)}
                        className="flex-shrink-0 h-8 w-8 p-0"
                      >
                        {key.showPlain ? (
                          <EyeOff className="w-4 h-4" />
                        ) : (
                          <Eye className="w-4 h-4" />
                        )}
                      </Button>
                      <Button
                        type="button"
                        variant="ghost"
                        size="sm"
                        onClick={() => handleCopyKey(key.id, key.plainKey)}
                        className="flex-shrink-0 h-8 w-8 p-0"
                      >
                        {copiedKeyId === key.id ? (
                          <CheckCircle2 className="w-4 h-4 text-green-600" />
                        ) : (
                          <Copy className="w-4 h-4" />
                        )}
                      </Button>
                    </div>
                  ) : null}

                  {/* Actions */}
                  <div className="flex items-center justify-end gap-2 pt-3 border-t border-border">
                    <Button
                      type="button"
                      variant="destructive"
                      size="sm"
                      onClick={() => handleRevokeKey(key.id)}
                      disabled={deletingKeyId === key.id}
                      className="gap-2"
                    >
                      {deletingKeyId === key.id ? (
                        <>
                          <Loader2 className="w-4 h-4 animate-spin" />
                          {t("apiKeys.revoking") || "Revoking..."}
                        </>
                      ) : (
                        <>
                          <Trash2 className="w-4 h-4" />
                          {t("apiKeys.revoke") || "Revoke"}
                        </>
                      )}
                    </Button>
                  </div>
                </div>
              ))}
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
