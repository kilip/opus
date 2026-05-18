import { createFileRoute } from '@tanstack/react-router';
import { LoginForm } from '@/features/auth/components/LoginForm';
import { AuthLayout } from '@/shared/components/layout';

/**
 * Route definition for the login page.
 * Uses a separate AuthLayout to provide a clean, focused experience.
 */
export const Route = createFileRoute('/login')({
  component: LoginPage,
});

function LoginPage() {
  return (
    <AuthLayout>
      <LoginForm />
    </AuthLayout>
  );
}
