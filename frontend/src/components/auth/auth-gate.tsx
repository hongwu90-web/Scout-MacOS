import { useEffect, useState } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { sessionAPI, setUnauthorizedCallback, APIError } from "@/lib/api";
import { LoginView } from "./login-view";
import { Loader2 } from "lucide-react";

interface AuthGateProps {
  children: React.ReactNode;
}

export function AuthGate({ children }: AuthGateProps) {
  const [authState, setAuthState] = useState<"checking" | "authenticated" | "unauthenticated">("checking");
  const queryClient = useQueryClient();

  useEffect(() => {
    // Register global 401 interceptor so any expired session shows login
    setUnauthorizedCallback(() => {
      setAuthState("unauthenticated");
    });

    let isMounted = true;

    async function checkAuth() {
      try {
        await sessionAPI.getMe();
        if (isMounted) {
          setAuthState("authenticated");
        }
      } catch (err: any) {
        if (isMounted) {
          // If 401, definitely unauthenticated
          if (err instanceof APIError && err.status === 401) {
            setAuthState("unauthenticated");
          } else {
            // For other errors or public mode without auth enabled
            setAuthState("unauthenticated");
          }
        }
      }
    }

    checkAuth();

    return () => {
      isMounted = false;
    };
  }, []);

  const handleLoginSuccess = () => {
    setAuthState("authenticated");
    void queryClient.invalidateQueries();
  };

  if (authState === "checking") {
    return (
      <div className="min-h-screen w-full flex items-center justify-center bg-[#FAF9F6] dark:bg-[#121212]">
        <div className="flex flex-col items-center gap-3">
          <Loader2 className="w-6 h-6 animate-spin text-muted-foreground" />
          <span className="text-xs font-mono text-muted-foreground">Connecting to Scout...</span>
        </div>
      </div>
    );
  }

  if (authState === "unauthenticated") {
    return <LoginView onSuccess={handleLoginSuccess} />;
  }

  return <>{children}</>;
}
