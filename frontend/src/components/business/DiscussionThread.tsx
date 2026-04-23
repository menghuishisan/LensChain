"use client";

// DiscussionThread.tsx
// 模块03讨论详情组件，安全展示 Markdown 文本、回复、点赞和置顶操作。

import { useState } from "react";

import { Button } from "@/components/ui/Button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/Card";
import { ErrorState } from "@/components/ui/ErrorState";
import { LoadingState } from "@/components/ui/LoadingState";
import { Textarea } from "@/components/ui/Textarea";
import { useToast } from "@/components/ui/Toast";
import { useDiscussion, useDiscussionMutations } from "@/hooks/useDiscussions";
import { safeMarkdownText } from "@/lib/content-safety";
import { formatDateTime } from "@/lib/format";

/**
 * DiscussionThread 组件属性。
 */
export interface DiscussionThreadProps {
  discussionID: string;
}

/**
 * DiscussionThread 讨论帖详情组件。
 */
export function DiscussionThread({ discussionID }: DiscussionThreadProps) {
  const query = useDiscussion(discussionID);
  const mutations = useDiscussionMutations(query.data?.course_id, discussionID);
  const { showToast } = useToast();
  const [reply, setReply] = useState("");

  if (query.isLoading) return <LoadingState />;
  if (query.isError) return <ErrorState description={query.error.message} />;
  if (!query.data) return null;

  return (
    <Card>
      <CardHeader>
        <CardTitle>{query.data.title}</CardTitle>
        <CardDescription>{query.data.author_name} · {formatDateTime(query.data.created_at)} · {query.data.like_count} 赞</CardDescription>
      </CardHeader>
      <CardContent className="space-y-5">
        <pre className="whitespace-pre-wrap rounded-xl bg-muted/60 p-4 text-sm">{safeMarkdownText(query.data.content)}</pre>
        <div className="flex gap-2">
          <Button variant="outline" onClick={() => (query.data.is_liked ? mutations.unlike.mutate() : mutations.like.mutate())}>{query.data.is_liked ? "取消点赞" : "点赞"}</Button>
          <Button variant="outline" onClick={() => mutations.pin.mutate(!query.data.is_pinned)}>{query.data.is_pinned ? "取消置顶" : "置顶"}</Button>
        </div>
        <div className="space-y-3">
          {query.data.replies.map((item) => (
            <div key={item.id} className="rounded-xl border border-border p-4">
              <p className="text-sm font-semibold">{item.author_name} · {formatDateTime(item.created_at)}</p>
              <pre className="mt-2 whitespace-pre-wrap text-sm">{safeMarkdownText(item.content)}</pre>
            </div>
          ))}
        </div>
        <Textarea placeholder="回复内容，支持Markdown纯文本" value={reply} onChange={(event) => setReply(event.target.value)} />
        <Button
          disabled={!reply.trim()}
          isLoading={mutations.reply.isPending}
          onClick={() => mutations.reply.mutate({ content: reply }, { onSuccess: () => { setReply(""); showToast({ title: "回复已发布", variant: "success" }); } })}
        >
          回复
        </Button>
      </CardContent>
    </Card>
  );
}
