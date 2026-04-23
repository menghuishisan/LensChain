"use client";

// useDiscussions.ts
// 模块03讨论、公告、评价 hook，统一处理互动 mutation 后的缓存刷新。

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import {
  createAnnouncement,
  createDiscussion,
  createDiscussionReply,
  createEvaluation,
  deleteAnnouncement,
  deleteDiscussion,
  getDiscussion,
  likeDiscussion,
  listAnnouncements,
  listDiscussions,
  listEvaluations,
  pinAnnouncement,
  pinDiscussion,
  unlikeDiscussion,
  updateAnnouncement,
  updateEvaluation,
} from "@/services/courseDiscussion";
import type { ID, QueryParams } from "@/types/api";

/**
 * useAnnouncements 查询课程公告。
 */
export function useAnnouncements(courseID: ID, params: QueryParams) {
  return useQuery({ queryKey: ["course", "announcements", courseID, params], queryFn: () => listAnnouncements(courseID, params), enabled: courseID.length > 0 });
}

/**
 * useAnnouncementMutations 公告创建、编辑、置顶、删除 mutation。
 */
export function useAnnouncementMutations(courseID: ID) {
  const queryClient = useQueryClient();
  const refresh = () => void queryClient.invalidateQueries({ queryKey: ["course", "announcements", courseID] });
  return {
    create: useMutation({ mutationFn: (payload: Parameters<typeof createAnnouncement>[1]) => createAnnouncement(courseID, payload), onSuccess: refresh }),
    update: useMutation({ mutationFn: ({ id, payload }: { id: ID; payload: Parameters<typeof updateAnnouncement>[1] }) => updateAnnouncement(id, payload), onSuccess: refresh }),
    pin: useMutation({ mutationFn: ({ id, isPinned }: { id: ID; isPinned: boolean }) => pinAnnouncement(id, isPinned), onSuccess: refresh }),
    remove: useMutation({ mutationFn: deleteAnnouncement, onSuccess: refresh }),
  };
}

/**
 * useDiscussions 查询课程讨论列表。
 */
export function useDiscussions(courseID: ID, params: QueryParams) {
  return useQuery({ queryKey: ["course", "discussions", courseID, params], queryFn: () => listDiscussions(courseID, params), enabled: courseID.length > 0 });
}

/**
 * useDiscussion 查询帖子详情。
 */
export function useDiscussion(discussionID: ID) {
  return useQuery({ queryKey: ["course", "discussion", discussionID], queryFn: () => getDiscussion(discussionID), enabled: discussionID.length > 0 });
}

/**
 * useDiscussionMutations 讨论发帖、回复、点赞、置顶、删除 mutation。
 */
export function useDiscussionMutations(courseID?: ID, discussionID?: ID) {
  const queryClient = useQueryClient();
  const refresh = () => {
    if (courseID) {
      void queryClient.invalidateQueries({ queryKey: ["course", "discussions", courseID] });
    }
    if (discussionID) {
      void queryClient.invalidateQueries({ queryKey: ["course", "discussion", discussionID] });
    }
  };

  return {
    create: useMutation({ mutationFn: (payload: Parameters<typeof createDiscussion>[1]) => createDiscussion(courseID ?? "", payload), onSuccess: refresh }),
    reply: useMutation({ mutationFn: (payload: Parameters<typeof createDiscussionReply>[1]) => createDiscussionReply(discussionID ?? "", payload), onSuccess: refresh }),
    like: useMutation({ mutationFn: () => likeDiscussion(discussionID ?? ""), onSuccess: refresh }),
    unlike: useMutation({ mutationFn: () => unlikeDiscussion(discussionID ?? ""), onSuccess: refresh }),
    pin: useMutation({ mutationFn: (isPinned: boolean) => pinDiscussion(discussionID ?? "", isPinned), onSuccess: refresh }),
    remove: useMutation({ mutationFn: () => deleteDiscussion(discussionID ?? ""), onSuccess: refresh }),
  };
}

/**
 * useEvaluations 查询课程评价。
 */
export function useEvaluations(courseID: ID, params: QueryParams) {
  return useQuery({ queryKey: ["course", "evaluations", courseID, params], queryFn: () => listEvaluations(courseID, params), enabled: courseID.length > 0 });
}

/**
 * useEvaluationMutations 评价提交与修改 mutation。
 */
export function useEvaluationMutations(courseID: ID) {
  const queryClient = useQueryClient();
  return {
    create: useMutation({
      mutationFn: (payload: Parameters<typeof createEvaluation>[1]) => createEvaluation(courseID, payload),
      onSuccess: () => void queryClient.invalidateQueries({ queryKey: ["course", "evaluations", courseID] }),
    }),
    update: useMutation({
      mutationFn: ({ id, payload }: { id: ID; payload: Parameters<typeof updateEvaluation>[1] }) => updateEvaluation(id, payload),
      onSuccess: () => void queryClient.invalidateQueries({ queryKey: ["course", "evaluations", courseID] }),
    }),
  };
}
