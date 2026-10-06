// features/notes/components/noteColors.ts
export const NOTE_COLORS = [
  {
    pill: "border-blue-300 bg-blue-50 text-blue-700 hover:bg-blue-100 dark:border-blue-800 dark:bg-blue-950 dark:text-blue-300",
    mark: "bg-blue-200/70 dark:bg-blue-500/30",
  },
  {
    pill: "border-purple-300 bg-purple-50 text-purple-700 hover:bg-purple-100 dark:border-purple-800 dark:bg-purple-950 dark:text-purple-300",
    mark: "bg-purple-200/70 dark:bg-purple-500/30",
  },
  {
    pill: "border-amber-300 bg-amber-50 text-amber-700 hover:bg-amber-100 dark:border-amber-800 dark:bg-amber-950 dark:text-amber-300",
    mark: "bg-amber-200/70 dark:bg-amber-500/30",
  },
];

export function noteColor(index: number) {
  return NOTE_COLORS[index % NOTE_COLORS.length];
}