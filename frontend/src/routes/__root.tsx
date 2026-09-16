import { createRootRoute, Outlet } from "@tanstack/react-router";
import { AuthGate } from "@/components/auth/auth-gate";

export const Route = createRootRoute({
  component: RootLayout,
});

function RootLayout() {
  return (
    <AuthGate>
      <Outlet />
    </AuthGate>
  );
}
