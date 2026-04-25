# Frontend

`frontend/` 是 LensChain 的 Web 前端，面向教师、学生、学校管理员和平台管理员提供统一界面。

它负责展示课程、实验、仿真、CTF、成绩和通知等业务内容，并通过后端 API 完成交互。

## 目录结构

```text
frontend/
├── src/app/         # 页面与路由
├── src/components/  # 页面组件和通用组件
├── src/hooks/       # 数据获取与状态逻辑
├── src/services/    # API 调用层
├── src/stores/      # 全局状态
├── src/types/       # 类型定义
└── README.md
```

## 本地启动

```bash
cd frontend
npm install
npm run dev
```

默认本地访问地址：

- `http://localhost:3000`

## 使用前建议

为了让页面打开后有真实内容，建议先完成：

1. 启动本地依赖
2. 初始化数据库和 demo 数据
3. 启动后端

这样登录、课程列表、实验入口和系统页面都会有可见内容。

## 常用命令

```bash
cd frontend
npm run dev
npm run build
npm run lint
```

## 适合谁阅读

- 想本地运行前端的人
- 前端开发者
- 负责联调和验收的人

如果你想了解整体项目，请先看根目录 [README.md](/abs/path/E:/code/LensChain/README.md)。
