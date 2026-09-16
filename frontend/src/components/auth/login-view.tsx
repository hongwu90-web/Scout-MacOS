import { useState } from "react";
import { Loader2, Lock, User, ShieldAlert } from "lucide-react";
import { sessionAPI } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";

interface LoginViewProps {
  onSuccess: () => void;
}

export function LoginView({ onSuccess }: LoginViewProps) {
  const [username, setUsername] = useState("alberto");
  const [password, setPassword] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!password.trim()) {
      setError("Please enter your password");
      return;
    }

    setLoading(true);
    setError(null);

    try {
      await sessionAPI.login({
        username: username.trim() || "alberto",
        password: password.trim(),
      });
      onSuccess();
    } catch (err: any) {
      setError(err?.message || "Invalid username or password");
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="min-h-screen w-full flex items-center justify-center bg-[#FAF9F6] dark:bg-[#121212] px-4">
      <div className="w-full max-w-sm">
        {/* Logo / Header */}
        <div className="text-center mb-8">
          <div className="inline-flex items-center justify-center w-12 h-12 rounded-xl bg-[#111111] dark:bg-white text-white dark:text-[#111111] mb-3 shadow-sm">
            <Lock className="w-5 h-5" />
          </div>
          <h1 className="text-2xl font-serif font-semibold tracking-tight text-[#111111] dark:text-[#F3F3F3]">
            Scout
          </h1>
          <p className="text-xs text-muted-foreground mt-1">
            Sign in to access your intelligence feeds
          </p>
        </div>

        {/* Login Box */}
        <div className="bg-card border border-border/80 rounded-2xl p-6 shadow-sm">
          <form onSubmit={handleSubmit} className="space-y-4">
            {error && (
              <div className="flex items-center gap-2 p-3 text-xs rounded-lg bg-rose-50 dark:bg-rose-950/30 text-rose-800 dark:text-rose-200 border border-rose-200 dark:border-rose-900/50">
                <ShieldAlert className="w-4 h-4 shrink-0 text-rose-600 dark:text-rose-400" />
                <span>{error}</span>
              </div>
            )}

            <div className="space-y-1.5">
              <label className="text-xs font-medium text-muted-foreground flex items-center gap-1.5">
                <User className="w-3.5 h-3.5" />
                Username
              </label>
              <Input
                type="text"
                value={username}
                onChange={(e) => setUsername(e.target.value)}
                placeholder="Username"
                autoComplete="username"
                disabled={loading}
                className="bg-background"
              />
            </div>

            <div className="space-y-1.5">
              <label className="text-xs font-medium text-muted-foreground flex items-center gap-1.5">
                <Lock className="w-3.5 h-3.5" />
                Password
              </label>
              <Input
                type="password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                placeholder="Enter password"
                autoComplete="current-password"
                autoFocus
                disabled={loading}
                className="bg-background"
              />
            </div>

            <Button
              type="submit"
              disabled={loading}
              className="w-full mt-2 font-medium bg-[#111111] hover:bg-[#222222] dark:bg-white dark:hover:bg-neutral-200 text-white dark:text-[#111111]"
            >
              {loading ? (
                <>
                  <Loader2 className="w-4 h-4 mr-2 animate-spin" />
                  Signing in...
                </>
              ) : (
                "Sign In"
              )}
            </Button>
          </form>
        </div>

        {/* Footer info */}
        <p className="text-center text-[11px] text-muted-foreground/70 mt-6 font-mono">
          Private Deployment • Scout Standalone
        </p>
      </div>
    </div>
  );
}
