"use client";

// CourseInteractionPanels.tsx
// 课程互动组件：讨论区、公告、评价。

import Link from "next/link";
import { useState } from "react";

import { Button } from "@/components/ui/Button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/Card";
import { ConfirmDialog } from "@/components/ui/ConfirmDialog";
import { Input } from "@/components/ui/Input";
import { Pagination } from "@/components/ui/Pagination";
import { Textarea } from "@/components/ui/Textarea";
import { useAnnouncementMutations, useAnnouncements, useDiscussionMutations, useDiscussions, useEvaluations, useEvaluationMutations } from "@/hooks/useDiscussions";
import { safeMarkdownText } from "@/lib/content-safety";
import type { ID } from "@/types/api";

/**
 * DiscussionListPanel 课程讨论区列表组件。
 */
export function DiscussionListPanel({ courseID }: { courseID: ID }) {
  const [page, setPage] = useState(1);
  const query = useDiscussions(courseID, { page, page_size: 20 });
  const mutations = useDiscussionMutations(courseID);
  const [title, setTitle] = useState("");
  const [content, setContent] = useState("");
  const pinnedDiscussions = (query.data?.list ?? []).filter((item) => item.is_pinned);
  const normalDiscussions = (query.data?.list ?? []).filter((item) => !item.is_pinned);
  return (
    <Card>
      <CardHeader>
        <CardTitle>课程讨论区</CardTitle>
      </CardHeader>
      <CardContent className="space-y-4">
        <div className="grid gap-2">
          <Input placeholder="标题" value={title} onChange={(event) => setTitle(event.target.value)} />
          <Textarea placeholder="内容，Markdown纯文本安全渲染" value={content} onChange={(event) => setContent(event.target.value)} />
          <Button
            disabled={!title || !content}
            onClick={() => mutations.create.mutate({ title, content }, { onSuccess: () => { setTitle(""); setContent(""); } })}
          >
            发新帖
          </Button>
        </div>
        {pinnedDiscussions.length > 0 ? (
          <div className="space-y-3">
            <p className="text-sm font-semibold text-muted-foreground">置顶帖子</p>
            {pinnedDiscussions.map((item) => (
              <div key={item.id} className="rounded-xl border border-border bg-muted/40 p-4">
                <Link className="font-semibold hover:text-primary" href={`/discussions/${item.id}`}>置顶 · {item.title}</Link>
                <p className="mt-1 text-sm text-muted-foreground">回复{item.reply_count} · 赞{item.like_count}</p>
              </div>
            ))}
          </div>
        ) : null}
        {normalDiscussions.map((item) => (
          <div key={item.id} className="rounded-xl border border-border p-4">
            <Link className="font-semibold hover:text-primary" href={`/discussions/${item.id}`}>{item.title}</Link>
            <p className="mt-1 text-sm text-muted-foreground">回复{item.reply_count} · 赞{item.like_count}</p>
          </div>
        ))}
        {query.data?.pagination ? (
          <Pagination page={query.data.pagination.page} totalPages={query.data.pagination.total_pages} total={query.data.pagination.total} onPageChange={setPage} />
        ) : null}
      </CardContent>
    </Card>
  );
}

/**
 * AnnouncementPanel 课程公告组件。
 */
export function AnnouncementPanel({ courseID, role }: { courseID: ID; role: "teacher" | "student" }) {
  const [page, setPage] = useState(1);
  const query = useAnnouncements(courseID, { page, page_size: 20 });
  const mutations = useAnnouncementMutations(courseID);
  const [title, setTitle] = useState("");
  const [content, setContent] = useState("");
  const isTeacher = role === "teacher";
  const pinnedAnnouncements = (query.data?.list ?? []).filter((item) => item.is_pinned);
  const normalAnnouncements = (query.data?.list ?? []).filter((item) => !item.is_pinned);
  return (
    <Card>
      <CardHeader>
        <CardTitle>课程公告</CardTitle>
      </CardHeader>
      <CardContent className="space-y-4">
        {isTeacher ? (
          <div className="grid gap-2">
            <Input placeholder="公告标题" value={title} onChange={(event) => setTitle(event.target.value)} />
            <Textarea placeholder="公告内容" value={content} onChange={(event) => setContent(event.target.value)} />
            <Button
              disabled={!title || !content}
              onClick={() => mutations.create.mutate({ title, content }, { onSuccess: () => { setTitle(""); setContent(""); } })}
            >
              发布公告
            </Button>
          </div>
        ) : null}
        {pinnedAnnouncements.length > 0 ? (
          <div className="space-y-3">
            <p className="text-sm font-semibold text-muted-foreground">置顶公告</p>
            {pinnedAnnouncements.map((item) => (
              <div key={item.id} className="rounded-xl border border-border bg-muted/40 p-4">
                <div className="flex justify-between gap-3">
                  <p className="font-semibold">置顶 · {item.title}</p>
                  {isTeacher ? (
                    <div className="flex gap-2">
                      <Button size="sm" variant="outline" onClick={() => mutations.pin.mutate({ id: item.id, isPinned: !item.is_pinned })}>取消置顶</Button>
                      <ConfirmDialog
                        title="删除公告"
                        description="删除后该公告将无法恢复，确定继续吗？"
                        confirmText="删除"
                        onConfirm={() => mutations.remove.mutate(item.id)}
                        trigger={<Button size="sm" variant="destructive">删除</Button>}
                      />
                    </div>
                  ) : null}
                </div>
                <pre className="mt-2 whitespace-pre-wrap text-sm">{safeMarkdownText(item.content)}</pre>
              </div>
            ))}
          </div>
        ) : null}
        {normalAnnouncements.map((item) => (
          <div key={item.id} className="rounded-xl border border-border p-4">
            <div className="flex justify-between gap-3">
              <p className="font-semibold">{item.title}</p>
              {isTeacher ? (
                <div className="flex gap-2">
                  <Button size="sm" variant="outline" onClick={() => mutations.pin.mutate({ id: item.id, isPinned: !item.is_pinned })}>置顶</Button>
                  <ConfirmDialog
                    title="删除公告"
                    description="删除后该公告将无法恢复，确定继续吗？"
                    confirmText="删除"
                    onConfirm={() => mutations.remove.mutate(item.id)}
                    trigger={<Button size="sm" variant="destructive">删除</Button>}
                  />
                </div>
              ) : null}
            </div>
            <pre className="mt-2 whitespace-pre-wrap text-sm">{safeMarkdownText(item.content)}</pre>
          </div>
        ))}
        {query.data?.pagination ? (
          <Pagination page={query.data.pagination.page} totalPages={query.data.pagination.total_pages} total={query.data.pagination.total} onPageChange={setPage} />
        ) : null}
      </CardContent>
    </Card>
  );
}

/**
 * EvaluationPanel 课程评价组件。
 */
export function EvaluationPanel({ courseID, role }: { courseID: ID; role: "teacher" | "student" }) {
  const isTeacher = role === "teacher";
  const query = useEvaluations(isTeacher ? courseID : "", { page: 1, page_size: 20 });
  const mutations = useEvaluationMutations(courseID);
  const [rating, setRating] = useState(5);
  const [comment, setComment] = useState("");
  return (
    <Card>
      <CardHeader>
        <CardTitle>课程评价</CardTitle>
        <CardDescription>{isTeacher ? "查看学生对本课程的评价与评分统计。" : "课程结束后可提交评价。"}</CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        {!isTeacher ? (
          <div className="grid gap-2">
            <Input type="number" min={1} max={5} value={rating} onChange={(event) => setRating(Number(event.target.value))} />
            <Textarea value={comment} onChange={(event) => setComment(event.target.value)} />
            <Button onClick={() => mutations.create.mutate({ rating, comment })}>提交评价</Button>
          </div>
        ) : null}
        {isTeacher ? (
          <>
            <div className="rounded-xl bg-muted/60 p-4">
              平均评分：{query.data?.summary.avg_rating ?? 0} · 共{query.data?.summary.total_count ?? 0}条
            </div>
            {(query.data?.items ?? []).map((item) => (
              <div key={item.id} className="rounded-xl border border-border p-4">
                {item.student_name} · {item.rating}星
                <p className="mt-2 text-sm">{item.comment}</p>
              </div>
            ))}
          </>
        ) : null}
      </CardContent>
    </Card>
  );
}
