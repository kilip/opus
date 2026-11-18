import { Tag } from "~/shared/components/Tag";
import type { RunState } from "../api/types";
import { STATE_TONES } from "../model";

export function RunStateBadge({ state }: { state?: RunState }) {
	const resolved: RunState = state ?? "queued";
	return <Tag tone={STATE_TONES[resolved]}>{resolved}</Tag>;
}
