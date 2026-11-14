import type { components } from "~/shared/api/schema";

/*
 * API types for the tasks feature.
 * Sourced from the generated OpenAPI schema components["schemas"].
 * Fallbacks are provided until `mise run api` regenerates schema.d.ts.
 */
type Schemas = components["schemas"];

export type Task = Schemas extends { "handler.Task": infer T }
  ? T
  : {
      id: string;
      key: string;
      source: TaskSource;
      externalId?: string;
      externalUrl?: string;
      title: string;
      description: string;
      status: TaskStatus;
      sourceStatus?: string;
      priority: TaskPriority;
      type: TaskType;
      assignee?: Assignee;
      dueDate?: string;
      labels: string[];
      createdAt: string;
      updatedAt: string;
    };

export type TaskStatus = Schemas extends { "handler.TaskStatus": infer T }
  ? T
  : "backlog" | "spec" | "inProgress" | "inReview" | "done";

export type TaskPriority = Schemas extends { "handler.TaskPriority": infer T }
  ? T
  : "none" | "low" | "medium" | "high";

export type TaskSource = Schemas extends { "handler.TaskSource": infer T }
  ? T
  : "github" | "jira" | "sentry" | "custom";

export type TaskType = Schemas extends { "handler.TaskType": infer T }
  ? T
  : "feature" | "bug" | "chore" | "incident";

export type Assignee = Schemas extends { "handler.Assignee": infer T }
  ? T
  : {
      id: string;
      name: string;
    };

export type Activity = Schemas extends { "handler.Activity": infer T }
  ? T
  : {
      id: string;
      kind: ActivityKind;
      author: string;
      body: string;
      createdAt: string;
    };

export type ActivityKind = Schemas extends { "handler.ActivityKind": infer T }
  ? T
  : "message" | "note" | "tracking";

export type CreateTaskRequest = Schemas extends {
  "handler.CreateTaskRequest": infer T;
}
  ? T
  : {
      title: string;
      description?: string;
      status?: TaskStatus;
      priority?: TaskPriority;
      type?: TaskType;
      assignee?: Assignee;
      dueDate?: string;
      labels?: string[];
    };

export type UpdateTaskRequest = Schemas extends {
  "handler.UpdateTaskRequest": infer T;
}
  ? T
  : {
      title?: string;
      description?: string;
      status?: TaskStatus;
      priority?: TaskPriority;
      type?: TaskType;
      assignee?: Assignee;
      clearAssignee?: boolean;
      dueDate?: string;
      labels?: string[];
    };

export type CreateActivityRequest = Schemas extends {
  "handler.CreateActivityRequest": infer T;
}
  ? T
  : {
      kind?: ActivityKind;
      author?: string;
      body: string;
    };
