export type StageDef<Id extends string = string> = { id: Id; name: string };

/** Chevron-shaped stage selector (form view status bar). */
export function Statusbar<Id extends string>({
  stages,
  current,
  onSelect,
}: {
  stages: StageDef<Id>[];
  current: Id;
  onSelect: (id: Id) => void;
}) {
  return (
    <fieldset aria-label="Stage" className="statusbar">
      {stages.map((stage) => (
        <button
          key={stage.id}
          type="button"
          className="statusbar-step"
          aria-pressed={stage.id === current}
          onClick={() => onSelect(stage.id)}
        >
          {stage.name}
        </button>
      ))}
    </fieldset>
  );
}
