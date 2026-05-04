"use client";

// ProfilePanel.tsx
// 模块01个人中心组件：只展示和编辑个人基础资料；学习概览明确由模块06提供。

import { Pencil, UserRound } from "lucide-react";
import Link from "next/link";
import { useEffect, useState } from "react";

import { Button, buttonClassName } from "@/components/ui/Button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/Card";
import { EmptyState } from "@/components/ui/EmptyState";
import { ErrorState } from "@/components/ui/ErrorState";
import { FormField } from "@/components/ui/FormField";
import { Input } from "@/components/ui/Input";
import { LoadingState } from "@/components/ui/LoadingState";
import { useToast } from "@/components/ui/Toast";
import { useProfile, useUpdateProfileMutation } from "@/hooks/useProfile";
import { validateOptionalEmail } from "@/lib/auth-validation";
import { LearningOverviewPanel } from "@/components/business/LearningOverviewPanel";

/**
 * ProfilePanel 个人中心组件。
 */
export function ProfilePanel() {
  const { data, isLoading, isError, error } = useProfile();
  const mutation = useUpdateProfileMutation();
  const { showToast } = useToast();
  const [nickname, setNickname] = useState("");
  const [avatarUrl, setAvatarUrl] = useState("");
  const [email, setEmail] = useState("");
  const [isEditing, setIsEditing] = useState(false);

  useEffect(() => {
    if (data === undefined) {
      return;
    }

    setNickname(data.nickname ?? "");
    setAvatarUrl(data.avatar_url ?? "");
    setEmail(data.email ?? "");
  }, [data]);

  if (isLoading) {
    return <LoadingState />;
  }

  if (isError) {
    return <ErrorState description={error.message} />;
  }

  if (data === undefined) {
    return <EmptyState title="暂无个人资料" description="请重新登录后查看个人中心。" />;
  }

  const isEmailValid = validateOptionalEmail(email);
  const showLearningOverview = data.roles.includes("student");

  return (
    <div className="space-y-6">
      <section className="rounded-[2rem] bg-[radial-gradient(circle_at_top_left,hsl(var(--primary)/0.18),transparent_28rem),linear-gradient(135deg,hsl(var(--card)),hsl(40_76%_95%))] p-6 shadow-panel">
        <div className="flex flex-col gap-5 md:flex-row md:items-center">
          <div className="flex h-24 w-24 items-center justify-center rounded-3xl bg-primary/12 text-primary">
            {data.avatar_url ? (
              // eslint-disable-next-line @next/next/no-img-element
              <img src={data.avatar_url} alt={data.name} className="h-full w-full rounded-3xl object-cover" />
            ) : (
              <UserRound className="h-12 w-12" />
            )}
          </div>
          <div className="min-w-0 flex-1">
            <h2 className="font-display text-4xl font-semibold">{data.name}</h2>
            <p className="mt-2 text-sm text-muted-foreground">
              {[data.school_name, data.college, data.major].filter(Boolean).join(" · ") || "学校与院系信息待完善"}
            </p>
            <p className="mt-1 text-sm text-muted-foreground">学号/工号：{data.student_no ?? "未填写"}</p>
          </div>
          <Link className={buttonClassName({ variant: "outline" })} href="/profile/password">
            修改密码
          </Link>
        </div>
      </section>

      <div className={showLearningOverview ? "grid gap-6 xl:grid-cols-[1.2fr_0.8fr]" : "grid gap-6"}>
        <Card>
          <CardHeader className="flex-row items-center justify-between space-y-0">
            <div>
              <CardTitle>基本信息</CardTitle>
              <CardDescription>昵称、头像、邮箱可编辑；学籍信息由管理员维护。</CardDescription>
            </div>
            <Button type="button" variant="outline" onClick={() => setIsEditing((current) => !current)}>
              <Pencil className="h-4 w-4" />
              {isEditing ? "取消编辑" : "编辑"}
            </Button>
          </CardHeader>
          <CardContent>
            <form
              className="grid gap-4"
              onSubmit={(event) => {
                event.preventDefault();
                if (!isEmailValid) {
                  return;
                }

                mutation.mutate(
                  { nickname, avatar_url: avatarUrl, email },
                  {
                    onSuccess: () => {
                      setIsEditing(false);
                      showToast({ title: "个人信息已更新", variant: "success" });
                    },
                    onError: (mutationError) => showToast({ title: "更新失败", description: mutationError.message, variant: "destructive" }),
                  },
                );
              }}
            >
              <FormField label="昵称" id="nickname">
                <Input id="nickname" value={nickname} onChange={(event) => setNickname(event.target.value)} disabled={!isEditing} />
              </FormField>
              <FormField label="头像链接" id="avatar-url" description="当前可填写头像图片链接。">
                <Input id="avatar-url" value={avatarUrl} onChange={(event) => setAvatarUrl(event.target.value)} disabled={!isEditing} />
              </FormField>
              <FormField label="邮箱" id="email" error={isEmailValid ? undefined : "邮箱格式不正确"}>
                <Input id="email" value={email} onChange={(event) => setEmail(event.target.value)} disabled={!isEditing} hasError={!isEmailValid} />
              </FormField>
              <ReadonlyGrid
                items={[
                  ["手机号", data.phone],
                  ["学院", data.college ?? "不可修改"],
                  ["专业", data.major ?? "不可修改"],
                  ["班级", data.class_name ?? "不可修改"],
                  ["学业层次", data.education_level_text ?? "不可修改"],
                ]}
              />
              {isEditing ? (
                <Button type="submit" isLoading={mutation.isPending} disabled={!isEmailValid}>
                  保存修改
                </Button>
              ) : null}
            </form>
          </CardContent>
        </Card>

        {showLearningOverview ? (
          <Card>
            <CardHeader>
              <CardTitle>学习概览</CardTitle>
              <CardDescription>这里会汇总你的课程、实验、竞赛和学习时长情况。</CardDescription>
            </CardHeader>
            <CardContent>
              <LearningOverviewPanel />
            </CardContent>
          </Card>
        ) : null}
      </div>
    </div>
  );
}

function ReadonlyGrid({ items }: { items: [string, string][] }) {
  return (
    <div className="grid gap-3 rounded-xl bg-muted/60 p-4 sm:grid-cols-2">
      {items.map(([label, value]) => (
        <div key={label}>
          <p className="text-xs text-muted-foreground">{label}</p>
          <p className="mt-1 text-sm font-semibold text-foreground">{value}</p>
        </div>
      ))}
    </div>
  );
}
