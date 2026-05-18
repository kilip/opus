import { useMutation, useQueryClient } from '@tanstack/react-query';
import { useNavigate } from '@tanstack/react-router';
import { ArrowRight, Lock, Mail } from 'lucide-react';
import { useState } from 'react';
import opusLogo from '@/assets/opus.svg';
import { GitHubIcon, GoogleIcon } from '@/shared/components/icons';
import { Button } from '@/shared/components/ui/button';
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from '@/shared/components/ui/card';
import { Field } from '@/shared/components/ui/field';
import { Input } from '@/shared/components/ui/input';
import { Separator } from '@/shared/components/ui/separator';
import { initiateOAuth, login } from '../api';
import type { LoginCredentials } from '../types';

/**
 * Branded login form component.
 * Reuses existing UI components (Card, Button, Field, Input, Separator)
 * to maintain consistency and reduce code duplication.
 */
export function LoginForm() {
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState<string | null>(null);

  const loginMutation = useMutation({
    mutationFn: (credentials: LoginCredentials) => login(credentials),
    onSuccess: (user) => {
      queryClient.setQueryData(['auth', 'me'], user);
      navigate({ to: '/agent' });
    },
    onError: (err: Error) => {
      setError(err.message || 'Login failed. Please check your credentials.');
    },
  });

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);
    loginMutation.mutate({ email, password });
  };

  const handleOAuth = (provider: 'google' | 'github') => {
    if (import.meta.env.DEV && !import.meta.env.VITE_API_URL) {
      localStorage.setItem('opus_mock_authed', 'true');
      navigate({ to: '/agent' });
      return;
    }
    initiateOAuth(provider);
  };

  return (
    <Card className="shadow-elevated border-brand-subtle bg-card/80 backdrop-blur-sm">
      <CardHeader className="text-center space-y-4">
        <div className="flex justify-center">
          <img src={opusLogo} alt="Opus Logo" className="h-16 w-16" />
        </div>
        <div className="space-y-1">
          <CardTitle>Welcome Back</CardTitle>
          <CardDescription className="italic">
            The agents have been expecting you.
          </CardDescription>
        </div>
      </CardHeader>
      <CardContent>
        <form onSubmit={handleSubmit} className="space-y-6">
          <Field label="Email Address" htmlFor="email" required>
            <div className="relative">
              <Mail className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted" />
              <Input
                id="email"
                type="email"
                placeholder="alice@example.com"
                className="pl-10"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                required
              />
            </div>
          </Field>

          <Field label="Password" htmlFor="password" required>
            <div className="relative">
              <Lock className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted" />
              <Input
                id="password"
                type="password"
                placeholder="••••••••"
                className="pl-10"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                required
              />
              <button
                type="button"
                className="absolute right-3 top-1/2 -translate-y-1/2 text-xs font-sans text-brand-primary hover:text-brand-secondary transition-colors"
              >
                Forgot?
              </button>
            </div>
          </Field>

          {error && (
            <p className="text-xs font-sans text-brand-primary bg-brand-primary/10 border border-brand-primary/20 rounded-badge px-3 py-2 text-center">
              {error}
            </p>
          )}

          <Button
            type="submit"
            className="w-full"
            disabled={loginMutation.isPending}
          >
            {loginMutation.isPending ? 'Authenticating...' : 'Sign In'}
            {!loginMutation.isPending && (
              <ArrowRight className="ml-2 h-4 w-4" />
            )}
          </Button>
        </form>

        <div className="relative my-8">
          <Separator />
          <div className="absolute inset-0 flex items-center justify-center">
            <span className="bg-card px-3 text-[10px] font-sans font-medium uppercase tracking-widest text-muted">
              Or continue with
            </span>
          </div>
        </div>

        <div className="grid grid-cols-2 gap-4">
          <Button variant="secondary" onClick={() => handleOAuth('github')}>
            <GitHubIcon className="mr-2 h-4 w-4" />
            GitHub
          </Button>
          <Button variant="secondary" onClick={() => handleOAuth('google')}>
            <GoogleIcon className="mr-2 h-4 w-4" />
            Google
          </Button>
        </div>
      </CardContent>
      <CardFooter className="justify-center border-t border-border mt-2 py-4">
        <p className="text-xs text-muted font-serif">
          Don't have an account?{' '}
          <button
            type="button"
            className="font-sans font-medium text-brand-primary hover:text-brand-secondary"
          >
            Request access
          </button>
        </p>
      </CardFooter>
    </Card>
  );
}
