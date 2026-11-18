import { Link } from "react-router";

import { buttonClass } from "~/shared/components/Button";

export const meta = () => [{ title: "Not found · Opus" }];

export default function NotFound() {
  return (
    <div className="flex flex-1 flex-col items-center justify-center gap-3 p-8 text-center">
      <p className="text-5xl font-semibold text-primary">404</p>
      <p className="text-muted">The page you are looking for does not exist.</p>
      <Link to="/tasks" className={buttonClass("primary")}>
        Back to tasks
      </Link>
    </div>
  );
}
