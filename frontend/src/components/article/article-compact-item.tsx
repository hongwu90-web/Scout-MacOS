import { useMemo } from "react";
import { cn, formatDate } from "@/lib/utils";
import type { Item } from "@/lib/api";
import { Star, Check } from "lucide-react";
import { Checkbox } from "@/components/ui/checkbox";

interface ArticleCompactItemProps {
  article: Item;
  selectedArticleId: number | null;
  onSelectArticle: (articleId: number | null) => void;
  onToggleRead: (article: Item) => Promise<void>;
  onToggleStar: (article: Item) => Promise<void>;
  canToggleRead: boolean;
  isStarred: boolean;
  isStarredMode?: boolean;
  feedName: string;
  isMultiSelected?: boolean;
  isSelectionMode?: boolean;
  matchedKeywords?: string[];
  onToggleSelect?: (e: React.MouseEvent) => void;
}

export function ArticleCompactItem({
  article,
  selectedArticleId,
  onSelectArticle,
  onToggleRead,
  onToggleStar,
  canToggleRead,
  isStarred,
  isStarredMode = false,
  feedName,
  isMultiSelected = false,
  isSelectionMode = false,
  matchedKeywords,
  onToggleSelect,
}: ArticleCompactItemProps) {
  const isSelected = selectedArticleId === article.id;
  const hasTranslation = Boolean(article.translated_title && article.translated_title.trim() !== "");
  const displayTitle = hasTranslation ? article.translated_title : article.title;

  const matchedPill = useMemo(() => {
    if (!matchedKeywords || matchedKeywords.length === 0) return null;
    const title = (article.title || "").toLowerCase();
    const trans = (article.translated_title || "").toLowerCase();
    const content = (article.content || "").toLowerCase();
    for (const kw of matchedKeywords) {
      const lower = kw.toLowerCase().trim();
      if (lower && (title.includes(lower) || trans.includes(lower) || content.includes(lower))) {
        return kw.trim();
      }
    }
    return null;
  }, [matchedKeywords, article.title, article.translated_title, article.content]);

  const handleToggleRead = async (e: React.MouseEvent) => {
    e.stopPropagation();
    if (!canToggleRead) return;
    try {
      await onToggleRead(article);
    } catch (error) {
      console.error("Failed to toggle read status:", error);
    }
  };

  const handleToggleStar = async (e: React.MouseEvent) => {
    e.stopPropagation();
    try {
      await onToggleStar(article);
    } catch (error) {
      console.error("Failed to toggle star:", error);
    }
  };

  return (
    <div
      onClick={() => onSelectArticle(article.id)}
      className={cn(
        "group relative flex w-full items-center justify-between gap-3 px-3 py-2 border-b border-border/60 text-left transition-colors cursor-pointer bg-card hover:bg-accent/30",
        isSelected
          ? "bg-accent/60 border-primary ring-1 ring-primary/40 z-10"
          : "hover:border-foreground/30",
        isMultiSelected && "bg-primary/10 border-primary/70 ring-1 ring-primary/40",
      )}
    >
      {/* Left side: Checkbox, Status dot, Favicon, Source, Title */}
      <div className="flex items-center gap-2 min-w-0 flex-1">
        {/* Selection Checkbox */}
        <div
          onClick={(e) => {
            e.stopPropagation();
            onToggleSelect?.(e);
          }}
          className={cn(
            "shrink-0 p-1 -m-1 transition-opacity cursor-pointer flex items-center justify-center",
            isSelectionMode || isMultiSelected
              ? "opacity-100"
              : "opacity-0 group-hover:opacity-100",
          )}
        >
          <Checkbox
            checked={isMultiSelected}
            onClick={(e) => onToggleSelect?.(e)}
            className="h-3.5 w-3.5"
            aria-label="Select article"
          />
        </div>

        {/* Unread indicator dot */}
        <span
          className={cn(
            "h-1.5 w-1.5 rounded-full shrink-0 transition-colors",
            article.unread ? "bg-primary" : "bg-transparent",
            (isSelectionMode || isMultiSelected) && "hidden",
          )}
          title={article.unread ? "Unread" : "Read"}
        />

        {/* Source Feed Name */}
        <span className="font-mono text-[10px] uppercase font-bold text-muted-foreground/80 truncate max-w-[130px] shrink-0">
          [{feedName}]
        </span>

        {matchedPill && (
          <span className="font-mono text-[9px] uppercase font-bold text-primary bg-primary/10 border border-primary/20 px-1 py-0.5 rounded shrink-0">
            [{matchedPill}]
          </span>
        )}

        {/* Article Title */}
        <span
          className={cn(
            "text-xs truncate transition-colors font-sans",
            isStarredMode || article.unread
              ? "text-foreground font-semibold"
              : "text-muted-foreground font-normal",
            isSelected && "text-foreground",
          )}
        >
          {displayTitle}
        </span>

        {hasTranslation && (
          <span className="font-mono text-[9px] font-bold text-primary/80 shrink-0 bg-primary/10 px-1 py-0.5 rounded">
            EN
          </span>
        )}
      </div>

      {/* Right side: Publication date, Quick action buttons */}
      <div className="flex items-center gap-3 shrink-0">
        <span className="font-mono text-[10px] text-muted-foreground/70 shrink-0">
          {formatDate(article.pub_date)}
        </span>

        {/* Quick action buttons on hover / active */}
        <div className="flex items-center gap-1 opacity-0 group-hover:opacity-100 transition-opacity">
          {!isStarredMode && canToggleRead && (
            <button
              type="button"
              onClick={handleToggleRead}
              className={cn(
                "p-1 rounded hover:bg-background border border-border text-muted-foreground hover:text-foreground transition-colors",
                article.unread ? "hover:border-primary/50" : "",
              )}
              title={article.unread ? "Mark as read" : "Mark as unread"}
            >
              <Check className="w-3 h-3" />
            </button>
          )}

          <button
            type="button"
            onClick={handleToggleStar}
            className={cn(
              "p-1 rounded hover:bg-background border border-border text-muted-foreground hover:text-foreground transition-colors",
              isStarred
                ? "text-amber-600 dark:text-amber-400 border-amber-500/40 dark:border-amber-500/50 bg-amber-500/10 dark:bg-amber-500/15 hover:bg-amber-500/20 shadow-2xs"
                : "",
            )}
            title={isStarred ? "Remove from starred" : "Star article"}
          >
            <Star className={cn("w-3 h-3", isStarred ? "fill-amber-500 text-amber-500" : "text-muted-foreground")} />
          </button>
        </div>

        {/* Persistent Star if starred */}
        {isStarred && (
          <Star className="w-3 h-3 text-amber-500 fill-amber-500 shrink-0 group-hover:hidden" />
        )}
      </div>
    </div>
  );
}
