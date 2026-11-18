import { Link } from "react-router";

import { Avatar } from "~/shared/components/Avatar";
import { Priority } from "~/shared/components/Priority";
import { Tag, toneFor } from "~/shared/components/Tag";
import { useTaskActions } from "../api/actions";
import type { Task } from "../api/types";
import { priorityFromLevel, priorityLevel } from "../model";
import { DeadlineChip } from "./DeadlineChip";

export function TaskCard({
  task,
  dragging,
  onDragStart,
  onDragEnd,
}: {
  task: Task;
  dragging: boolean;
  onDragStart: () => void;
  onDragEnd: () => void;
}) {
  const { updateTask } = useTaskActions();

  return (
    <article
      draggable
      onDragStart={(e) => {
        e.dataTransfer.setData("text/plain", task.id);
        e.dataTransfer.effectAllowed = "move";
        onDragStart();
      }}
      onDragEnd={onDragEnd}
      className={`group relative cursor-grab rounded-md border border-line bg-surface p-3 shadow-card hover:border-line-strong active:cursor-grabbing ${
        dragging ? "opacity-40" : ""
      }`}
    >
      <p className="text-xs text-muted">{task.key}</p>
      {/* Stretched link: the whole card opens the task, stars stay clickable. */}
      <Link
        to={`/tasks/${task.id}`}
        draggable={false}
        className="block font-medium after:absolute after:inset-0 after:content-[''] hover:text-primary"
      >
        {task.title}
      </Link>

      {task.labels.length > 0 && (
        <ul className="mt-2 flex flex-wrap gap-1">
          {task.labels.map((label) => (
            <li key={label}>
              <Tag tone={toneFor(label)}>{label}</Tag>
            </li>
          ))}
        </ul>
      )}

      <div className="relative z-10 mt-3 flex items-center gap-3">
        <Priority
          value={priorityLevel(task.priority)}
          size={15}
          onChange={(level) =>
            updateTask(task.id, { priority: priorityFromLevel(level) })
          }
        />
        <DeadlineChip task={task} />
        <span className="ml-auto">
          {task.assignee && <Avatar name={task.assignee.name} size={24} />}
        </span>
      </div>
    </article>
  );
}
