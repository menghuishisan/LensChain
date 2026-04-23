"use client";

// NotificationPreferenceForm.tsx
// 模块07通知偏好组件，支持强制通知锁定和即时保存。

import { BellOff } from "lucide-react";
import { useState } from "react";

import { Button } from "@/components/ui/Button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/Card";
import { useNotificationPreferenceMutation, useNotificationPreferences } from "@/hooks/useNotificationTemplates";

/** NotificationPreferenceForm 通知偏好组件。 */
export function NotificationPreferenceForm() {
  const preferencesQuery = useNotificationPreferences();
  const mutation = useNotificationPreferenceMutation();
  const [draft, setDraft] = useState<Record<number, boolean>>({});

  const preferences = preferencesQuery.data?.preferences ?? [];

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <BellOff className="h-5 w-5 text-primary" />
          通知偏好设置
        </CardTitle>
      </CardHeader>
      <CardContent className="space-y-4">
        {preferences.map((item) => {
          const currentValue = draft[item.category] ?? item.is_enabled;
          return (
            <label key={item.category} className="flex items-center justify-between rounded-xl border border-border p-4">
              <div>
                <p className="font-semibold">{item.category_text}</p>
                <p className="mt-1 text-sm text-muted-foreground">{item.is_forced ? "强制接收，不可关闭" : "可按需开启或关闭"}</p>
              </div>
              <input type="checkbox" checked={currentValue} disabled={item.is_forced} onChange={(event) => setDraft((current) => ({ ...current, [item.category]: event.target.checked }))} />
            </label>
          );
        })}
        <Button
          onClick={() =>
            mutation.mutate(
              preferences
                .filter((item) => !item.is_forced)
                .map((item) => ({ category: item.category, is_enabled: draft[item.category] ?? item.is_enabled })),
            )
          }
          isLoading={mutation.isPending}
        >
          保存偏好
        </Button>
      </CardContent>
    </Card>
  );
}
