import { redirect } from "react-router";

export function clientLoader() {
  return redirect("/tasks");
}

export default function HomeRedirect() {
  return null;
}
