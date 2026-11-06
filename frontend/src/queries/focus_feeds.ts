import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { focusFeedAPI, type CreateFocusFeedRequest, type UpdateFocusFeedRequest, type FocusFeed } from "@/lib/api";
import { queryKeys } from "./keys";

export function useFocusFeeds() {
  return useQuery({
    queryKey: queryKeys.focusFeeds.list(),
    queryFn: async () => {
      const response = await focusFeedAPI.list();
      return response.data ?? [];
    },
    refetchInterval: 60_000,
    refetchOnWindowFocus: true,
    staleTime: 0,
  });
}

export function useFocusFeed(id: number | null) {
  const queryClient = useQueryClient();
  return useQuery({
    queryKey: queryKeys.focusFeeds.detail(id ?? 0),
    queryFn: async () => {
      if (!id || id <= 0) return null;
      const cachedList = queryClient.getQueryData<FocusFeed[]>(queryKeys.focusFeeds.list());
      const cached = cachedList?.find((ff) => ff.id === id);
      if (cached) {
        return cached;
      }
      const response = await focusFeedAPI.get(id);
      return response.data ?? null;
    },
    initialData: () => {
      if (!id || id <= 0) return undefined;
      const cachedList = queryClient.getQueryData<FocusFeed[]>(queryKeys.focusFeeds.list());
      return cachedList?.find((ff) => ff.id === id);
    },
    enabled: Boolean(id && id > 0),
  });
}

export function useCreateFocusFeed() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: CreateFocusFeedRequest) => focusFeedAPI.create(data),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.focusFeeds.all });
    },
  });
}

export function useUpdateFocusFeed() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ id, data }: { id: number; data: UpdateFocusFeedRequest }) =>
      focusFeedAPI.update(id, data),
    onSuccess: (_, variables) => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.focusFeeds.all });
      void queryClient.invalidateQueries({
        queryKey: queryKeys.focusFeeds.detail(variables.id),
      });
      void queryClient.invalidateQueries({ queryKey: queryKeys.items.all });
    },
  });
}

export function useDeleteFocusFeed() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (id: number) => focusFeedAPI.delete(id),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.focusFeeds.all });
      void queryClient.invalidateQueries({ queryKey: queryKeys.items.all });
    },
  });
}

export function usePreviewFocusFeed(keywords: string, feedIds?: string, enabled = true) {
  return useQuery({
    queryKey: ["focusFeeds", "preview", keywords, feedIds],
    queryFn: async () => {
      const response = await focusFeedAPI.preview({ keywords: keywords.trim(), feed_ids: feedIds });
      return response.data;
    },
    enabled: enabled && keywords.trim().length > 0,
    staleTime: 5_000,
  });
}
