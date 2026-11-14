import { Link } from "react-router";

import { Avatar } from "~/shared/components/Avatar";
import { Priority } from "~/shared/components/Priority";
import { Tag, toneFor } from "~/shared/components/Tag";
import type { Task } from "../api/types";
import { priorityLevel, STAGES } from "../model";
import { DeadlineChip } from "./DeadlineChip";

export function TaskList({ tasks }: { tasks: Task[] }) {
  if (tasks.length === 0) {
    return <p className="p-8 text-center text-muted">No tasks match.</p>;
  }

  return (
    <div className="min-h-0 flex-1 overflow-auto p-4">
      <table className="w-full min-w-[720px] border-collapse rounded-lg bg-surface text-left">
        <thead>
          <tr className="border-b border-line text-xs tracking-wide text-muted uppercase">
            <th className="px-3 py-2 font-semibold">Ref</th>
            <th className="px-3 py-2 font-semibold">Task</th>
            <th className="px-3 py-2 font-semibold">Assignee</th>
            <th className="px-3 py-2 font-semibold">Stage</th>
            <th className="px-3 py-2 font-semibold">Priority</th>
            <th className="px-3 py-2 font-semibold">Deadline</th>
          </tr>
        </thead>
        <tbody>
          {tasks.map((task) => {
            const stage = STAGES.find((s) => s.id === task.status);
            return (
              <tr
                key={task.id}
                className="relative border-b border-line last:border-b-0 hover:bg-surface-2"
              >
                <td className="px-3 py-2 text-muted">{task.key}</td>
                <td className="px-3 py-2">
                  <Link
                    to={`/tasks/${task.id}`}
                    className="font-medium after:absolute after:inset-0 after:content-[''] hover:text-primary"
                  >
                    {task.title}
                  </Link>
                  {task.labels.length > 0 && (
                    <span className="ml-2 inline-flex gap-1 align-middle">
                      {task.labels.map((label) => (
                        <Tag key={label} tone={toneFor(label)}>
                          {label}
                        </Tag>
                      ))}
                    </span>
                  )}
                </td>
                <td className="px-3 py-2">
                  {task.assignee ? (
                    <span className="inline-flex items-center gap-2">
                      <Avatar name={task.assignee.name} size={22} />
                      {task.assignee.name}
                    </span>
                  ) : (
                    <span className="text-muted">Unassigned</span>
                  )}
                </td>
                <td className="px-3 py-2">
                  {stage && <Tag tone={stage.tone}>{stage.name}</Tag>}
                </td>
                <td className="px-3 py-2">
                  <Priority value={priorityLevel(task.priority)} size={14} />
                </td>
                <td className="px-3 py-2">
                  <DeadlineChip task={task} />
                </td>
              </tr>
            );
          })}
        </tbody>
      </table>
    </div>
  );
}
