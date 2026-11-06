import { useState, useEffect, useRef } from "react";
import { Crosshair, Trash2, AlertCircle, CheckCircle2, Loader2 } from "lucide-react";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { useGroups } from "@/queries/groups";
import { useFeedLookup } from "@/queries/feeds";
import { useUpdateFocusFeed, useDeleteFocusFeed, usePreviewFocusFeed } from "@/queries/focus_feeds";
import { QUERY_OPERATOR_CHIPS } from "@/lib/focus-query";
import type { FocusFeed } from "@/lib/api";
import { toast } from "sonner";

interface EditFocusFeedDialogProps {
  focusFeed: FocusFeed | null;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onDeleted?: () => void;
}

export function EditFocusFeedDialog({
  focusFeed,
  open,
  onOpenChange,
  onDeleted,
}: EditFocusFeedDialogProps) {
  const { data: groups = [] } = useGroups();
  const { feeds } = useFeedLookup();
  const updateFocusFeed = useUpdateFocusFeed();
  const deleteFocusFeed = useDeleteFocusFeed();

  const [name, setName] = useState("");
  const [keywords, setKeywords] = useState("");
  const [debouncedKeywords, setDebouncedKeywords] = useState("");
  const [groupId, setGroupId] = useState<string>("");
  const [selectedFeedIds, setSelectedFeedIds] = useState<number[]>([]);
  const [isAllFeeds, setIsAllFeeds] = useState(true);

  const textareaRef = useRef<HTMLTextAreaElement>(null);

  useEffect(() => {
    if (focusFeed) {
      setName(focusFeed.name);
      setKeywords(focusFeed.keywords);
      setDebouncedKeywords(focusFeed.keywords);
      setGroupId(String(focusFeed.group_id));

      if (focusFeed.feed_ids && focusFeed.feed_ids.trim() !== "") {
        setIsAllFeeds(false);
        const ids = focusFeed.feed_ids
          .split(",")
          .map((id) => parseInt(id.trim(), 10))
          .filter((id) => !isNaN(id) && id > 0);
        setSelectedFeedIds(ids);
      } else {
        setIsAllFeeds(true);
        setSelectedFeedIds([]);
      }
    }
  }, [focusFeed]);

  // Debounce keywords input by 300ms for live preview
  useEffect(() => {
    const timer = setTimeout(() => {
      setDebouncedKeywords(keywords);
    }, 300);
    return () => clearTimeout(timer);
  }, [keywords]);

  const feedIDsStr = isAllFeeds ? "" : selectedFeedIds.join(",");
  const { data: previewData, isLoading: isPreviewLoading } = usePreviewFocusFeed(
    debouncedKeywords,
    feedIDsStr,
    open && Boolean(debouncedKeywords.trim()),
  );

  const insertOperator = (insertText: string) => {
    const textarea = textareaRef.current;
    if (!textarea) {
      setKeywords((prev) => (prev ? prev + insertText : insertText.trimStart()));
      return;
    }
    const start = textarea.selectionStart;
    const end = textarea.selectionEnd;
    const prev = keywords;
    const next = prev.substring(0, start) + insertText + prev.substring(end);
    setKeywords(next);
    setTimeout(() => {
      textarea.focus();
      const newPos = start + insertText.length;
      textarea.setSelectionRange(newPos, newPos);
    }, 0);
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!focusFeed) return;
    if (!name.trim() || !keywords.trim()) {
      toast.error("Please enter a name and at least one keyword");
      return;
    }

    if (previewData && !previewData.valid) {
      toast.error(`Invalid query syntax: ${previewData.error}`);
      return;
    }

    const targetGroupId = groupId
      ? parseInt(groupId, 10)
      : focusFeed.group_id;

    try {
      await updateFocusFeed.mutateAsync({
        id: focusFeed.id,
        data: {
          name: name.trim(),
          keywords: keywords.trim(),
          feed_ids: feedIDsStr,
          group_id: targetGroupId,
        },
      });
      toast.success(`Updated Focus Feed "${name}"`);
      onOpenChange(false);
    } catch (err: any) {
      toast.error(err.message || "Failed to update Focus Feed");
    }
  };

  const handleDelete = async () => {
    if (!focusFeed) return;
    if (!confirm(`Are you sure you want to delete Focus Feed "${focusFeed.name}"?`)) {
      return;
    }

    try {
      await deleteFocusFeed.mutateAsync(focusFeed.id);
      toast.success(`Deleted Focus Feed "${focusFeed.name}"`);
      onOpenChange(false);
      onDeleted?.();
    } catch (err: any) {
      toast.error(err.message || "Failed to delete Focus Feed");
    }
  };

  const toggleFeedSelection = (feedId: number) => {
    setSelectedFeedIds((prev) =>
      prev.includes(feedId)
        ? prev.filter((id) => id !== feedId)
        : [...prev, feedId],
    );
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[540px] font-mono">
        <DialogHeader>
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2">
              <Crosshair className="h-5 w-5 text-primary" />
              <DialogTitle className="text-base uppercase tracking-wider">
                Edit Focus Feed
              </DialogTitle>
            </div>
            <Button
              type="button"
              variant="destructive"
              size="sm"
              onClick={handleDelete}
              disabled={deleteFocusFeed.isPending}
              className="gap-1 text-xs"
            >
              <Trash2 className="h-3.5 w-3.5" />
              Delete
            </Button>
          </div>
          <DialogDescription className="text-xs text-muted-foreground">
            Modify keyword filters, boolean logic, or target feeds for this focus feed.
          </DialogDescription>
        </DialogHeader>

        <form onSubmit={handleSubmit} className="space-y-4 pt-2">
          {/* Name */}
          <div className="space-y-1.5">
            <label className="text-[11px] uppercase font-bold text-foreground">
              Focus Feed Name *
            </label>
            <Input
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="e.g. Semiconductor Intel"
              required
              className="text-xs font-mono"
            />
          </div>

          {/* Keywords & Boolean Logic */}
          <div className="space-y-1.5">
            <div className="flex items-center justify-between">
              <label className="text-[11px] uppercase font-bold text-foreground">
                Target Keywords & Logic *
              </label>
              {/* Quick Operators Toolbar */}
              <div className="flex items-center gap-1">
                <span className="text-[10px] text-muted-foreground mr-1">Insert:</span>
                {QUERY_OPERATOR_CHIPS.map((chip) => (
                  <button
                    key={chip.label}
                    type="button"
                    onClick={() => insertOperator(chip.insert)}
                    className="px-1.5 py-0.5 rounded text-[10px] font-mono font-medium bg-muted/60 hover:bg-primary/20 hover:text-primary border border-border/80 transition-colors cursor-pointer"
                  >
                    {chip.label}
                  </button>
                ))}
              </div>
            </div>

            <Textarea
              ref={textareaRef}
              value={keywords}
              onChange={(e) => setKeywords(e.target.value)}
              placeholder='e.g. TSMC AND (lithography OR "advanced packaging") NOT gaming'
              rows={3}
              required
              className="text-xs font-mono resize-y"
            />

            {/* Live Syntax & Count Feedback */}
            {keywords.trim() && (
              <div className="mt-1 text-xs">
                {isPreviewLoading ? (
                  <div className="flex items-center gap-1.5 text-[11px] text-muted-foreground py-1">
                    <Loader2 className="h-3 w-3 animate-spin" />
                    <span>Analyzing query & matching articles...</span>
                  </div>
                ) : previewData ? (
                  !previewData.valid ? (
                    <div className="flex items-start gap-1.5 text-[11px] text-destructive bg-destructive/10 border border-destructive/20 p-2 rounded">
                      <AlertCircle className="h-3.5 w-3.5 shrink-0 mt-0.5" />
                      <span>Syntax Error: {previewData.error}</span>
                    </div>
                  ) : (
                    <div className="flex flex-col gap-1.5 p-2 rounded bg-muted/40 border border-border text-[11px]">
                      <div className="flex items-center justify-between">
                        <div className="flex items-center gap-1.5">
                          <CheckCircle2
                            className={`h-3.5 w-3.5 ${
                              previewData.matched_count > 0 ? "text-emerald-500" : "text-amber-500"
                            }`}
                          />
                          <span className="font-semibold">
                            {previewData.matched_count} matching {previewData.matched_count === 1 ? "article" : "articles"}
                          </span>
                        </div>
                        <span className="text-[10px] text-muted-foreground">
                          {isAllFeeds ? "All feeds" : `${selectedFeedIds.length} feeds selected`}
                        </span>
                      </div>
                      {previewData.leaf_keywords && previewData.leaf_keywords.length > 0 && (
                        <div className="flex items-center gap-1 flex-wrap pt-0.5 border-t border-border/50">
                          <span className="text-[10px] text-muted-foreground">Matched terms:</span>
                          {previewData.leaf_keywords.slice(0, 6).map((kw) => (
                            <span
                              key={kw}
                              className="px-1.5 py-0.2 rounded text-[10px] bg-primary/10 text-primary border border-primary/20 font-mono"
                            >
                              {kw}
                            </span>
                          ))}
                          {previewData.leaf_keywords.length > 6 && (
                            <span className="text-[10px] text-muted-foreground">
                              +{previewData.leaf_keywords.length - 6} more
                            </span>
                          )}
                        </div>
                      )}
                    </div>
                  )
                ) : null}
              </div>
            )}

            <p className="text-[10px] text-muted-foreground">
              Supports: <code className="text-foreground">AND</code>, <code className="text-foreground">OR</code>, <code className="text-foreground">NOT</code>, parentheses <code className="text-foreground">()</code>, and <code className="text-foreground">"exact phrases"</code>. Spaces act as AND; commas act as OR.
            </p>
          </div>

          {/* Group */}
          <div className="space-y-1.5">
            <label className="text-[11px] uppercase font-bold text-foreground">
              Category Group
            </label>
            <Select value={groupId} onValueChange={(val) => setGroupId(val ?? "")}>
              <SelectTrigger className="text-xs font-mono">
                <SelectValue placeholder="Select group" />
              </SelectTrigger>
              <SelectContent className="font-mono text-xs">
                {groups.map((g) => (
                  <SelectItem key={g.id} value={String(g.id)}>
                    {g.name}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          {/* Feed Scope */}
          <div className="space-y-1.5 pt-1">
            <label className="text-[11px] uppercase font-bold text-foreground">
              Feed Sources Scope
            </label>
            <div className="flex items-center gap-3 text-xs mb-2">
              <label className="flex items-center gap-1.5 cursor-pointer">
                <input
                  type="radio"
                  name="editFeedScope"
                  checked={isAllFeeds}
                  onChange={() => setIsAllFeeds(true)}
                  className="accent-primary"
                />
                <span>All Feeds (Global)</span>
              </label>
              <label className="flex items-center gap-1.5 cursor-pointer">
                <input
                  type="radio"
                  name="editFeedScope"
                  checked={!isAllFeeds}
                  onChange={() => setIsAllFeeds(false)}
                  className="accent-primary"
                />
                <span>Specific Feeds ({selectedFeedIds.length} selected)</span>
              </label>
            </div>

            {!isAllFeeds && (
              <div className="max-h-36 overflow-y-auto border border-border p-2 rounded space-y-1 bg-muted/20">
                {feeds.map((feed) => (
                  <label
                    key={feed.id}
                    className="flex items-center gap-2 text-xs cursor-pointer hover:bg-muted/40 p-1 rounded"
                  >
                    <input
                      type="checkbox"
                      checked={selectedFeedIds.includes(feed.id)}
                      onChange={() => toggleFeedSelection(feed.id)}
                      className="accent-primary"
                    />
                    <span className="truncate">{feed.name}</span>
                  </label>
                ))}
              </div>
            )}
          </div>

          {/* Actions */}
          <div className="flex items-center justify-end gap-2 pt-3 border-t border-border">
            <Button
              type="button"
              variant="outline"
              size="sm"
              onClick={() => onOpenChange(false)}
              className="text-xs uppercase"
            >
              Cancel
            </Button>
            <Button
              type="submit"
              size="sm"
              disabled={updateFocusFeed.isPending || (previewData !== undefined && !previewData.valid)}
              className="text-xs uppercase font-bold"
            >
              {updateFocusFeed.isPending ? "Saving..." : "Save Changes"}
            </Button>
          </div>
        </form>
      </DialogContent>
    </Dialog>
  );
}
