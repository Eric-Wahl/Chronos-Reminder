import {
  Plus,
  Trash2,
  GripVertical,
  Link2,
  Sun,
  Moon,
  ClipboardList,
} from "lucide-react";
import {
  DndContext,
  DragOverlay,
  closestCenter,
  PointerSensor,
  TouchSensor,
  useSensor,
  useSensors,
  type DragEndEvent,
  type DragStartEvent,
  type DragOverEvent,
} from "@dnd-kit/core";
import {
  SortableContext,
  verticalListSortingStrategy,
  useSortable,
  arrayMove,
} from "@dnd-kit/sortable";
import { CSS } from "@dnd-kit/utilities";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
  CardDescription,
} from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Header } from "@/components/common/header";
import { Footer } from "@/components/common/footer";
import { DeleteConfirmModal } from "@/components/DeleteConfirmModal";
import { useTranslation } from "react-i18next";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { toast } from "sonner";
import {
  plannerService,
  dfmService,
  type PlannerItem,
  type PlannerPeriod,
  type DFMItem,
} from "@/services";

const PERIODS: PlannerPeriod[] = ["morning", "afternoon"];

export function DayPlannerPage() {
  const { t } = useTranslation();

  const [items, setItems] = useState<PlannerItem[]>([]);
  const [dfmItems, setDfmItems] = useState<DFMItem[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [activeId, setActiveId] = useState<string | null>(null);
  const [showClearConfirm, setShowClearConfirm] = useState(false);
  const [isClearing, setIsClearing] = useState(false);

  const sensors = useSensors(
    useSensor(PointerSensor, { activationConstraint: { distance: 4 } }),
    useSensor(TouchSensor, {
      activationConstraint: { delay: 150, tolerance: 5 },
    })
  );

  const loadData = useCallback(async () => {
    const [plannerItems, note] = await Promise.all([
      plannerService.getItems(),
      dfmService.getNote(),
    ]);
    setItems(plannerItems);
    setDfmItems(note?.items ?? []);
  }, []);

  useEffect(() => {
    setIsLoading(true);
    loadData().finally(() => setIsLoading(false));
  }, [loadData]);

  const columns = useMemo(() => {
    const grouped: Record<PlannerPeriod, PlannerItem[]> = {
      morning: [],
      afternoon: [],
    };
    for (const item of items) {
      grouped[item.period].push(item);
    }
    for (const period of PERIODS) {
      grouped[period].sort((a, b) => a.position - b.position);
    }
    return grouped;
  }, [items]);

  const activeItem = items.find((i) => i.id === activeId) ?? null;

  const handleAddItem = async (
    content: string,
    period: PlannerPeriod,
    dfmItemId?: string
  ) => {
    try {
      const created = await plannerService.addItem({
        content,
        period,
        dfm_item_id: dfmItemId,
      });
      setItems((prev) => [...prev, created]);
    } catch (err) {
      const message =
        err instanceof Error ? err.message : t("dayPlanner.addFailed");
      toast.error(message);
    }
  };

  const handleToggleChecked = async (item: PlannerItem) => {
    const nextChecked = !item.checked;
    setItems((prev) =>
      prev.map((i) => (i.id === item.id ? { ...i, checked: nextChecked } : i))
    );
    if (item.dfm_item_id) {
      setDfmItems((prev) =>
        prev.map((d) =>
          d.id === item.dfm_item_id ? { ...d, checked: nextChecked } : d
        )
      );
    }

    const updated = await plannerService.updateItem(item.id, {
      checked: nextChecked,
    });
    if (!updated) {
      // Revert on failure
      setItems((prev) =>
        prev.map((i) =>
          i.id === item.id ? { ...i, checked: item.checked } : i
        )
      );
      toast.error(t("dayPlanner.updateFailed"));
    }
  };

  const handleDeleteItem = async (item: PlannerItem) => {
    const previous = items;
    setItems((prev) => prev.filter((i) => i.id !== item.id));
    const success = await plannerService.deleteItem(item.id);
    if (!success) {
      setItems(previous);
      toast.error(t("dayPlanner.deleteFailed"));
    }
  };

  const handleClearAll = async () => {
    setIsClearing(true);
    const success = await plannerService.clearAll();
    setIsClearing(false);
    setShowClearConfirm(false);
    if (success) {
      setItems([]);
      toast.success(t("dayPlanner.cleared"));
    } else {
      toast.error(t("dayPlanner.clearFailed"));
    }
  };

  const persistOrder = useCallback((allItems: PlannerItem[]) => {
    const payload = PERIODS.flatMap((period) =>
      allItems
        .filter((i) => i.period === period)
        .map((i, index) => ({ id: i.id, position: index, period }))
    );
    plannerService.reorder(payload).catch(() => {
      toast.error(t("dayPlanner.reorderFailed"));
    });
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const findPeriod = (id: string): PlannerPeriod | null => {
    if (PERIODS.includes(id as PlannerPeriod)) return id as PlannerPeriod;
    return items.find((i) => i.id === id)?.period ?? null;
  };

  const handleDragStart = (event: DragStartEvent) => {
    setActiveId(String(event.active.id));
  };

  const handleDragOver = (event: DragOverEvent) => {
    const { active, over } = event;
    if (!over) return;

    const activeItemData = items.find((i) => i.id === active.id);
    const overPeriod = findPeriod(String(over.id));
    if (!activeItemData || !overPeriod) return;
    if (activeItemData.period === overPeriod) return;

    // Moving into a different column: move it there immediately (at the end
    // of that column) so the column the pointer is over reflects the drag
    // live. handleDragEnd finalizes the exact drop position.
    setItems((prev) => {
      const maxPosition = Math.max(
        0,
        ...prev.filter((i) => i.period === overPeriod).map((i) => i.position)
      );
      return prev.map((i) =>
        i.id === active.id
          ? { ...i, period: overPeriod, position: maxPosition + 1 }
          : i
      );
    });
  };

  const handleDragEnd = (event: DragEndEvent) => {
    setActiveId(null);
    const { active, over } = event;
    if (!over) return;

    const activePeriod = findPeriod(String(active.id));
    const overPeriod = findPeriod(String(over.id));
    if (!activePeriod || !overPeriod) return;

    setItems((prev) => {
      const columnIds = prev
        .filter((i) => i.period === overPeriod)
        .sort((a, b) => a.position - b.position)
        .map((i) => i.id);

      const oldIndex = columnIds.indexOf(String(active.id));
      const overIndex = columnIds.indexOf(String(over.id));

      let reorderedColumnIds = columnIds;
      if (oldIndex !== -1 && overIndex !== -1 && oldIndex !== overIndex) {
        reorderedColumnIds = arrayMove(columnIds, oldIndex, overIndex);
      }

      const otherItems = prev.filter((i) => i.period !== overPeriod);
      const reorderedColumn = reorderedColumnIds
        .map((id) => prev.find((i) => i.id === id))
        .filter((i): i is PlannerItem => !!i)
        .map((i, index) => ({ ...i, period: overPeriod, position: index }));

      const next = [...otherItems, ...reorderedColumn];
      persistOrder(next);
      return next;
    });
  };

  return (
    <div className="min-h-screen bg-background-main dark:bg-background-main">
      <Header />

      <main className="max-w-6xl mx-auto px-4 sm:px-6 lg:px-8 py-12 pt-24">
        <div className="mb-8 flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
          <div>
            <h2 className="text-3xl sm:text-4xl font-bold text-foreground flex items-center gap-3">
              <ClipboardList className="w-8 h-8 text-accent" />
              {t("dayPlanner.title")}
            </h2>
            <p className="text-muted-foreground text-base sm:text-lg mt-2">
              {t("dayPlanner.subtitle")}
            </p>
          </div>
          {items.length > 0 && (
            <Button
              variant="outline"
              onClick={() => setShowClearConfirm(true)}
              className="text-red-500 hover:text-red-600 border-red-500/30 hover:bg-red-500/10 gap-2 w-fit"
            >
              <Trash2 className="w-4 h-4" />
              {t("dayPlanner.clearAll")}
            </Button>
          )}
        </div>

        {isLoading ? (
          <Card className="border-border bg-card/95 backdrop-blur text-center py-12">
            <p className="text-muted-foreground">{t("dayPlanner.loading")}</p>
          </Card>
        ) : (
          <DndContext
            sensors={sensors}
            collisionDetection={closestCenter}
            onDragStart={handleDragStart}
            onDragOver={handleDragOver}
            onDragEnd={handleDragEnd}
          >
            <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
              <PlannerColumn
                period="morning"
                icon={<Sun className="w-5 h-5 text-amber-500" />}
                items={columns.morning}
                dfmItems={dfmItems}
                onAdd={handleAddItem}
                onToggle={handleToggleChecked}
                onDelete={handleDeleteItem}
              />
              <PlannerColumn
                period="afternoon"
                icon={<Moon className="w-5 h-5 text-indigo-400" />}
                items={columns.afternoon}
                dfmItems={dfmItems}
                onAdd={handleAddItem}
                onToggle={handleToggleChecked}
                onDelete={handleDeleteItem}
              />
            </div>

            <DragOverlay>
              {activeItem ? (
                <div className="rounded-md border border-accent/50 bg-card px-3 py-2 shadow-lg text-sm text-foreground">
                  {activeItem.content}
                </div>
              ) : null}
            </DragOverlay>
          </DndContext>
        )}
      </main>

      <DeleteConfirmModal
        isOpen={showClearConfirm}
        title={t("dayPlanner.clearAllTitle")}
        description={t("dayPlanner.clearAllDescription")}
        onConfirm={handleClearAll}
        onCancel={() => setShowClearConfirm(false)}
        isLoading={isClearing}
      />

      <Footer />
    </div>
  );
}

interface PlannerColumnProps {
  period: PlannerPeriod;
  icon: React.ReactNode;
  items: PlannerItem[];
  dfmItems: DFMItem[];
  onAdd: (content: string, period: PlannerPeriod, dfmItemId?: string) => void;
  onToggle: (item: PlannerItem) => void;
  onDelete: (item: PlannerItem) => void;
}

function PlannerColumn({
  period,
  icon,
  items,
  dfmItems,
  onAdd,
  onToggle,
  onDelete,
}: PlannerColumnProps) {
  const { t } = useTranslation();
  const [inputValue, setInputValue] = useState("");
  const [selectedDfmId, setSelectedDfmId] = useState<string | undefined>();
  const [showSuggestions, setShowSuggestions] = useState(false);
  const inputRef = useRef<HTMLInputElement>(null);

  const suggestions = useMemo(() => {
    const query = inputValue.trim().toLowerCase();
    if (!query) return [];
    return dfmItems
      .filter((d) => d.content.toLowerCase().includes(query))
      .slice(0, 5);
  }, [inputValue, dfmItems]);

  const handleSubmit = () => {
    const content = inputValue.trim();
    if (!content) return;
    onAdd(content, period, selectedDfmId);
    setInputValue("");
    setSelectedDfmId(undefined);
    setShowSuggestions(false);
  };

  const handleSelectSuggestion = (item: DFMItem) => {
    setInputValue(item.content);
    setSelectedDfmId(item.id);
    setShowSuggestions(false);
    inputRef.current?.focus();
  };

  return (
    <Card className="border-border bg-card/95 backdrop-blur">
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          {icon}
          {t(`dayPlanner.${period}`)}
        </CardTitle>
        <CardDescription>
          {items.length === 0
            ? t("dayPlanner.emptyColumn")
            : t("dayPlanner.itemCount", { count: items.length })}
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        <div className="relative">
          <div className="flex gap-2">
            <Input
              ref={inputRef}
              value={inputValue}
              onChange={(e) => {
                setInputValue(e.target.value);
                setSelectedDfmId(undefined);
                setShowSuggestions(true);
              }}
              onFocus={() => setShowSuggestions(true)}
              onBlur={() =>
                setTimeout(() => setShowSuggestions(false), 150)
              }
              onKeyDown={(e) => {
                if (e.key === "Enter") handleSubmit();
              }}
              placeholder={t("dayPlanner.addPlaceholder")}
              maxLength={200}
            />
            <Button
              onClick={handleSubmit}
              disabled={!inputValue.trim()}
              size="icon"
              className="bg-accent hover:bg-accent/90 text-accent-foreground shrink-0"
              aria-label={t("dayPlanner.addItem")}
            >
              <Plus className="w-4 h-4" />
            </Button>
          </div>

          {showSuggestions && suggestions.length > 0 && (
            <div className="absolute z-10 mt-1 w-full rounded-md border border-border bg-popover shadow-lg overflow-hidden">
              {suggestions.map((s) => (
                <button
                  key={s.id}
                  type="button"
                  onMouseDown={(e) => e.preventDefault()}
                  onClick={() => handleSelectSuggestion(s)}
                  className="flex w-full items-center gap-2 px-3 py-2 text-sm text-left text-foreground hover:bg-secondary/50 transition-colors"
                >
                  <Link2 className="w-3.5 h-3.5 text-accent shrink-0" />
                  <span className="truncate">{s.content}</span>
                </button>
              ))}
            </div>
          )}

          {selectedDfmId && (
            <p className="text-xs text-accent mt-1.5 flex items-center gap-1">
              <Link2 className="w-3 h-3" />
              {t("dayPlanner.willLink")}
            </p>
          )}
        </div>

        <SortableContext
          items={items.map((i) => i.id)}
          strategy={verticalListSortingStrategy}
        >
          <ul className="space-y-2 min-h-[2rem]" id={period}>
            {items.map((item) => (
              <PlannerRow
                key={item.id}
                item={item}
                onToggle={onToggle}
                onDelete={onDelete}
              />
            ))}
          </ul>
        </SortableContext>
      </CardContent>
    </Card>
  );
}

interface PlannerRowProps {
  item: PlannerItem;
  onToggle: (item: PlannerItem) => void;
  onDelete: (item: PlannerItem) => void;
}

function PlannerRow({ item, onToggle, onDelete }: PlannerRowProps) {
  const { t } = useTranslation();
  const { attributes, listeners, setNodeRef, transform, transition, isDragging } =
    useSortable({ id: item.id });

  const style = {
    transform: CSS.Transform.toString(transform),
    transition,
    opacity: isDragging ? 0.4 : 1,
  };

  return (
    <li
      ref={setNodeRef}
      style={style}
      className="flex items-center gap-2 rounded-md border border-border bg-background-main/40 px-2 py-2 group"
    >
      <button
        type="button"
        className="cursor-grab active:cursor-grabbing text-muted-foreground shrink-0 touch-none"
        {...attributes}
        {...listeners}
        aria-label={t("dayPlanner.dragHandle")}
      >
        <GripVertical className="w-4 h-4" />
      </button>

      <input
        type="checkbox"
        checked={item.checked}
        onChange={() => onToggle(item)}
        className="w-4 h-4 accent-accent cursor-pointer shrink-0"
        aria-label={t("dayPlanner.toggleItem")}
      />

      {item.dfm_item_id && (
        <Link2
          className="w-3.5 h-3.5 text-accent shrink-0"
          aria-label={t("dayPlanner.linkedToDFM")}
        />
      )}

      <span
        className={`flex-1 text-sm truncate ${
          item.checked
            ? "line-through text-muted-foreground"
            : "text-foreground"
        }`}
      >
        {item.content}
      </span>

      <Button
        size="icon"
        variant="ghost"
        className="h-7 w-7 shrink-0 md:opacity-0 md:group-hover:opacity-100 transition-opacity text-red-500 hover:text-red-600"
        onClick={() => onDelete(item)}
        aria-label={t("dayPlanner.deleteItem")}
      >
        <Trash2 className="w-3.5 h-3.5" />
      </Button>
    </li>
  );
}
