"use client";

// DiscussionThread.tsx
// 模块03讨论详情组件，安全展示 Markdown 文本、回复、点赞和置顶操作。

import React from "react";
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
    <div className="space-y-5">
      <div className="rounded-3xl border border-border/70 bg-[linear-gradient(135deg,hsl(182_34%_14%),hsl(24_46%_28%))] p-6 text-primary-foreground">
        <p className="text-sm text-primary-foreground/75">帖子详情</p>
        <h1 className="mt-2 font-display text-3xl font-semibold">{query.data.title}</h1>
        <p className="mt-3 text-sm text-primary-foreground/80">
          {query.data.is_pinned ? "置顶帖" : "普通帖子"} · {query.data.author_name} · {formatDateTime(query.data.created_at)}
        </p>
        <p className="mt-2 text-sm text-primary-foreground/80">回复 {query.data.reply_count} · 点赞 {query.data.like_count}</p>
      </div>

      <Card>
      <CardHeader>
        <CardTitle>正文</CardTitle>
        <CardDescription>支持点赞、置顶和继续回复讨论。</CardDescription>
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
              {item.reply_to_name ? <p className="mt-1 text-xs text-muted-foreground">@{item.reply_to_name}</p> : null}
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
          发布回复
        </Button>
      </CardContent>
      </Card>
    </div>
  );
}
