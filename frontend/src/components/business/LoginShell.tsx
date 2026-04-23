"use client";

// LoginShell.tsx
// 登录页布局壳，按模块01文档展示品牌 Hero、手机号密码表单和 SSO 入口。

import { Blocks, FlaskConical, Network, ShieldCheck, Trophy } from "lucide-react";

import { LoginForm, SSOLoginButton } from "@/components/business/AuthPanels";
import { Card, CardContent } from "@/components/ui/Card";

const FEATURE_ITEMS = [
  { label: "多链教学覆盖", icon: Network },
  { label: "动态实验环境", icon: FlaskConical },
  { label: "CTF竞赛实战", icon: Trophy },
  { label: "统一权限审计", icon: ShieldCheck },
] as const;

/**
 * LoginShell 登录页布局壳组件。
 */
export function LoginShell() {
  return (
    <main className="grid min-h-screen overflow-hidden bg-slate-950 text-foreground lg:grid-cols-[1.08fr_0.92fr]">
      <section className="relative hidden min-h-screen overflow-hidden bg-[radial-gradient(circle_at_20%_20%,rgba(20,184,166,0.42),transparent_30rem),linear-gradient(135deg,#052e2b,#101827_58%,#2c1d12)] p-10 text-white lg:flex lg:flex-col lg:justify-between">
        <div className="absolute inset-0 opacity-30 [background-image:linear-gradient(rgba(255,255,255,.08)_1px,transparent_1px),linear-gradient(90deg,rgba(255,255,255,.08)_1px,transparent_1px)] [background-size:44px_44px]" />
        <div className="relative z-10 flex items-center gap-3">
          <div className="flex h-11 w-11 items-center justify-center rounded-2xl bg-white/12 ring-1 ring-white/20">
            <Blocks className="h-6 w-6" />
          </div>
          <div>
            <p className="font-display text-2xl font-semibold">链镜 LensChain</p>
            <p className="text-sm text-white/66">区块链教学与实践平台</p>
          </div>
        </div>

        <div className="relative z-10 max-w-2xl animate-fade-up">
          <div className="mb-8 inline-flex rounded-full border border-white/15 bg-white/10 px-4 py-2 text-sm text-white/76 backdrop-blur">
            教学、实验、竞赛三位一体
          </div>
          <h1 className="font-display text-6xl font-semibold leading-tight tracking-tight">
            把区块链课堂变成可观测、可演练、可评测的学习现场。
          </h1>
          <p className="mt-6 max-w-xl text-lg leading-8 text-white/72">
            统一承载课程教学、实验实践和攻防竞赛，多角色协同管理，多租户隔离运行。
          </p>
          <div className="mt-10 grid max-w-xl grid-cols-2 gap-3">
            {FEATURE_ITEMS.map((item) => {
              const Icon = item.icon;
              return (
                <div key={item.label} className="rounded-2xl border border-white/12 bg-white/10 p-4 backdrop-blur">
                  <Icon className="mb-3 h-5 w-5 text-teal-200" />
                  <p className="text-sm font-semibold text-white/88">{item.label}</p>
                </div>
              );
            })}
          </div>
        </div>

        <div className="relative z-10 text-sm text-white/55">© 2026 链镜平台</div>
      </section>

      <section className="relative flex min-h-screen items-center justify-center bg-[radial-gradient(circle_at_top_right,rgba(245,158,11,0.16),transparent_22rem),linear-gradient(180deg,hsl(var(--background)),hsl(42_68%_95%))] px-5 py-10">
        <Card className="w-full max-w-md animate-fade-up border-white/70 bg-white/86 shadow-[0_32px_100px_rgba(15,23,42,0.18)]">
          <CardContent className="p-7 sm:p-9">
            <div className="mb-8 text-center">
              <div className="mx-auto mb-4 flex h-14 w-14 items-center justify-center rounded-2xl bg-primary text-primary-foreground shadow-glow">
                <Blocks className="h-7 w-7" />
              </div>
              <h2 className="font-display text-3xl font-semibold">登录链镜</h2>
              <p className="mt-2 text-sm text-muted-foreground">手机号密码登录，或通过学校统一认证进入平台。</p>
            </div>

            <LoginForm />

            <div className="my-6 flex items-center gap-3 text-xs text-muted-foreground">
              <span className="h-px flex-1 bg-border" />
              或
              <span className="h-px flex-1 bg-border" />
            </div>

            <SSOLoginButton />
          </CardContent>
        </Card>
      </section>
    </main>
  );
}
