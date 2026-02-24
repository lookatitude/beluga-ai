import { useState, useCallback, useEffect, useRef } from "react";

interface UseKeyboardNavigationOptions {
  itemCount: number;
  onSelect: (index: number) => void;
  onEscape: () => void;
  enabled?: boolean;
}

export function useKeyboardNavigation({
  itemCount,
  onSelect,
  onEscape,
  enabled = true,
}: UseKeyboardNavigationOptions) {
  const [selectedIndex, setSelectedIndex] = useState(-1);
  const containerRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    setSelectedIndex(-1);
  }, [itemCount]);

  const handleKeyDown = useCallback(
    (e: KeyboardEvent) => {
      if (!enabled || itemCount === 0) return;

      switch (e.key) {
        case "ArrowDown":
          e.preventDefault();
          setSelectedIndex((prev) =>
            prev < itemCount - 1 ? prev + 1 : 0
          );
          break;
        case "ArrowUp":
          e.preventDefault();
          setSelectedIndex((prev) =>
            prev > 0 ? prev - 1 : itemCount - 1
          );
          break;
        case "Enter":
          e.preventDefault();
          if (selectedIndex >= 0) {
            onSelect(selectedIndex);
          }
          break;
        case "Escape":
          e.preventDefault();
          onEscape();
          break;
      }
    },
    [enabled, itemCount, selectedIndex, onSelect, onEscape]
  );

  useEffect(() => {
    document.addEventListener("keydown", handleKeyDown);
    return () => document.removeEventListener("keydown", handleKeyDown);
  }, [handleKeyDown]);

  useEffect(() => {
    if (selectedIndex < 0 || !containerRef.current) return;
    const items = containerRef.current.querySelectorAll('[role="option"]');
    items[selectedIndex]?.scrollIntoView({ block: "nearest" });
  }, [selectedIndex]);

  return { selectedIndex, setSelectedIndex, containerRef };
}
