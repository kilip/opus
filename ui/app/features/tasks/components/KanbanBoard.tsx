import { useState } from "react";

import { Icon } from "~/shared/components/Icon";
import { useTaskActions } from "../api/actions";
import type { Task } from "../api/types";
import { CURRENT_USER, STAGES } from "../model";
import { TaskCard } from "./TaskCard";

export function KanbanBoard({ tasks }: { tasks: Task[] }) {
  const { createTask, moveTask } = useTaskActions();
  const [dragId, setDragId] = useState<string | null>(null);
  const [overStage, setOverStage] = useState<string | null>(null);
  const [adding, setAdding] = useState<string | null>(null);

  function endDrag() {
    setDragId(null);
    setOverStage(null);
  }

  return (
    <div className="flex min-h-0 flex-1 gap-3 overflow-x-auto p-4">
      {STAGES.map((stage) => {
        const items = tasks.filter((t) => t.status === stage.id);
        const isOver = overStage === stage.id;

        return (
          <section
            key={stage.id}
            aria-label={stage.name}
            onDragOver={(e) => {
              if (!dragId) return;
              e.preventDefault();
              e.dataTransfer.dropEffect = "move";
              setOverStage(stage.id);
            }}
            onDragLeave={(e) => {
              if (!e.currentTarget.contains(e.relatedTarget as Node)) {
                setOverStage(null);
              }
            }}
            onDrop={(e) => {
              e.preventDefault();
              const id = e.dataTransfer.getData("text/plain") || dragId;
              if (id) moveTask(id, stage.id);
              endDrag();
            }}
            className={`flex w-72 shrink-0 flex-col rounded-lg bg-surface-2 ${
              isOver ? "ring-2 ring-primary" : ""
            }`}
          >
            <header className="flex items-center gap-2 px-3 pt-3 pb-2">
              <h2 className="font-semibold">{stage.name}</h2>
              <span className="rounded-full bg-surface px-2 text-xs text-muted">
                {items.length}
              </span>
              <button
                type="button"
                aria-label={`Quick add to ${stage.name}`}
                onClick={() => setAdding(adding === stage.id ? null : stage.id)}
                className="ml-auto rounded p-1 text-muted hover:bg-surface hover:text-fg"
              >
                <Icon name="plus" size={16} />
              </button>
            </header>

            <div className="flex min-h-0 flex-1 flex-col gap-2 overflow-y-auto px-2 pb-2">
              {adding === stage.id && (
                <QuickAdd
                  onSubmit={(title) => {
                    createTask({
                      title,
                      status: stage.id,
                      priority: "none",
                      assignee: CURRENT_USER,
                      dueDate: null,
                      labels: [],
                      description: "",
                    });
                  }}
                  onClose={() => setAdding(null)}
                />
              )}
              {items.map((task) => (
                <TaskCard
                  key={task.id}
                  task={task}
                  dragging={dragId === task.id}
                  onDragStart={() => setDragId(task.id)}
                  onDragEnd={endDrag}
                />
              ))}
              {items.length === 0 && adding !== stage.id && (
                <p className="rounded-md border border-dashed border-line-strong px-3 py-6 text-center text-muted">
                  Drop tasks here
                </p>
              )}
            </div>
          </section>
        );
      })}
    </div>
  );
}

function QuickAdd({
  onSubmit,
  onClose,
}: {
  onSubmit: (title: string) => void;
  onClose: () => void;
}) {
  const [title, setTitle] = useState("");

  return (
    <form
      onSubmit={(e) => {
        e.preventDefault();
        const t = title.trim();
        if (!t) return;
        onSubmit(t);
        setTitle("");
      }}
    >
      <input
        // biome-ignore lint/a11y/noAutofocus: the field is opened on demand by the user
        autoFocus
        aria-label="Task title"
        className="field"
        placeholder="Task title, then Enter"
        value={title}
        onChange={(e) => setTitle(e.target.value)}
        onKeyDown={(e) => {
          if (e.key === "Escape") onClose();
        }}
      />
    </form>
  );
}
