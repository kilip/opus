import { useCallback, useEffect, useState } from "react";

export type ThemePref = "light" | "dark" | "system";

const KEY = "opus.theme";
const QUERY = "(prefers-color-scheme: dark)";

/** Runs inline in <head> before first paint so the page never flashes. */
export const themeBootstrapScript = `(function(){try{var p=localStorage.getItem("${KEY}")||"system";var d=p==="dark"||(p==="system"&&matchMedia("${QUERY}").matches);document.documentElement.dataset.theme=d?"dark":"light"}catch(e){}})()`;

export function readThemePref(): ThemePref {
  try {
    const v = localStorage.getItem(KEY);
    if (v === "light" || v === "dark" || v === "system") return v;
  } catch {
    // storage unavailable (private mode, blocked): fall back to system
  }
  return "system";
}

export function applyTheme(pref: ThemePref) {
  const dark =
    pref === "dark" || (pref === "system" && matchMedia(QUERY).matches);
  document.documentElement.dataset.theme = dark ? "dark" : "light";
}

export function useTheme() {
  const [pref, setPrefState] = useState<ThemePref>("system");

  useEffect(() => {
    setPrefState(readThemePref());
  }, []);

  // Follow OS changes while the preference is "system".
  useEffect(() => {
    if (pref !== "system") return;
    const mq = matchMedia(QUERY);
    const onChange = () => applyTheme("system");
    mq.addEventListener("change", onChange);
    return () => mq.removeEventListener("change", onChange);
  }, [pref]);

  const setPref = useCallback((next: ThemePref) => {
    setPrefState(next);
    try {
      localStorage.setItem(KEY, next);
    } catch {
      // ignore: preference just won't persist
    }
    applyTheme(next);
  }, []);

  return { pref, setPref };
}
