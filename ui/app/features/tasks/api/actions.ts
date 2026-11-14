import { useMutation, useQueryClient } from "@tanstack/react-query";

import { api, unwrap } from "~/shared/api/client";
import { TASK_ERRORS } from "./errors";
import { taskKeys } from "./keys";
import type {
  Activity,
  ActivityKind,
  CreateActivityRequest,
  CreateTaskRequest,
  Task,
  TaskStatus,
  UpdateTaskRequest,
} from "./types";

export type AssigneePreset = {
  id: string;
  name: string;
};

export const ASSIGNEE_PRESETS: AssigneePreset[] = [
  { id: "planner-agent", name: "planner-agent" },
  { id: "coder-agent", name: "coder-agent" },
  { id: "reviewer-agent", name: "reviewer-agent" },
];

export function useCreateTask() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (req: CreateTaskRequest): Promise<Task> =>
      unwrap(
        api.POST(
          "/tasks" as never,
          {
            body: req,
          } as never,
        ),
        TASK_ERRORS,
      ),
    onSuccess: (newTask) => {
      queryClient.setQueryData<Task[]>(taskKeys.list(), (old) =>
        old ? [...old, newTask] : [newTask],
      );
      queryClient.setQueryData(taskKeys.detail(newTask.id), newTask);
      queryClient.invalidateQueries({ queryKey: taskKeys.all });
    },
  });
}

export function useUpdateTask() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async ({
      id,
      ...patch
    }: UpdateTaskRequest & { id: string }): Promise<Task> =>
      unwrap(
        api.PATCH(
          "/tasks/{id}" as never,
          {
            params: { path: { id } },
            body: patch,
          } as never,
        ),
        TASK_ERRORS,
      ),
    onSuccess: (updated) => {
      queryClient.setQueryData<Task[]>(taskKeys.list(), (old) =>
        old ? old.map((t) => (t.id === updated.id ? updated : t)) : [updated],
      );
      queryClient.setQueryData(taskKeys.detail(updated.id), updated);
      queryClient.invalidateQueries({ queryKey: taskKeys.all });
    },
  });
}

export function useMoveTask() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async ({
      id,
      status,
    }: {
      id: string;
      status: TaskStatus;
    }): Promise<Task> =>
      unwrap(
        api.PATCH(
          "/tasks/{id}" as never,
          {
            params: { path: { id } },
            body: { status },
          } as never,
        ),
        TASK_ERRORS,
      ),
    onSuccess: (updated) => {
      queryClient.setQueryData<Task[]>(taskKeys.list(), (old) =>
        old ? old.map((t) => (t.id === updated.id ? updated : t)) : [updated],
      );
      queryClient.setQueryData(taskKeys.detail(updated.id), updated);
      queryClient.invalidateQueries({ queryKey: taskKeys.all });
    },
  });
}

export function useAddTaskActivity() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async ({
      id,
      ...req
    }: CreateActivityRequest & { id: string }): Promise<Activity> =>
      unwrap(
        api.POST(
          "/tasks/{id}/activity" as never,
          {
            params: { path: { id } },
            body: req,
          } as never,
        ),
        TASK_ERRORS,
      ),
    onSuccess: (activity, variables) => {
      queryClient.setQueryData<Activity[]>(
        taskKeys.activity(variables.id),
        (old) => (old ? [...old, activity] : [activity]),
      );
      queryClient.invalidateQueries({
        queryKey: taskKeys.activity(variables.id),
      });
      queryClient.invalidateQueries({
        queryKey: taskKeys.detail(variables.id),
      });
    },
  });
}

/** Legacy adapter providing synchronous-like hooks if needed */
export function useTaskActions() {
  const createTask = useCreateTask();
  const updateTask = useUpdateTask();
  const moveTask = useMoveTask();
  const addActivity = useAddTaskActivity();

  return {
    createTask: (input: CreateTaskRequest) => createTask.mutate(input),
    updateTask: (id: string, patch: UpdateTaskRequest) =>
      updateTask.mutate({ id, ...patch }),
    moveTask: (id: string, status: TaskStatus) =>
      moveTask.mutate({ id, status }),
    addMessage: (
      id: string,
      kind: ActivityKind,
      body: string,
      author = "You",
    ) => addActivity.mutate({ id, kind, body, author }),
  };
}
