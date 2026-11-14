import { Icon } from "~/shared/components/Icon";
import { formatDeadline, todayISO } from "~/shared/util/dates";
import type { Task } from "../api/types";

export function DeadlineChip({ task }: { task: Task }) {
  if (!task.dueDate) return null;
  const overdue = task.status !== "done" && task.dueDate < todayISO();

  return (
    <span
      className={`inline-flex items-center gap-1 text-xs ${
        overdue ? "font-medium text-danger-ink" : "text-muted"
      }`}
      title={overdue ? "Overdue" : "Deadline"}
    >
      <Icon name="clock" size={13} />
      {formatDeadline(task.dueDate)}
    </span>
  );
}
