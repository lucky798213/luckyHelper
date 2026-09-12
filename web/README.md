# luckyHelper 前端（Vite + Vue 3）

前后端分离的前端工程。开发期用 Vite dev server（:5173）+ 代理到后端 :8090；
生产期 `npm run build` 产出 `dist/`，由独立静态服务器 / nginx 托管。

## 开发

```bash
# 1) 先起后端（仓库根目录）
export JWT_SECRET=你的密钥          # ≥32 字节
go run cmd/server/main.go          # 监听 :8090

# 2) 起前端 dev server（本目录）
cd web
npm install
npm run dev                        # 打开 http://localhost:5173
```

`vite.config.js` 已把 `/api`、`/healthz`、`/readyz` 代理到 `http://localhost:8090`，
开发期同源、无 CORS。修改任意 `.vue/.js` 会热更新（HMR），不再需要浏览器硬刷新。

## 构建 & 部署（独立端口）

```bash
cd web
npm run build                      # 产出 web/dist/
```

用任意静态服务器托管 `dist/`（独立端口），并把 `/api` 反代到后端。示例见
`nginx.conf.example`。

两种接口寻址方式（二选一）：
- **nginx 反代（推荐）**：`VITE_API_BASE` 留空，前端请求同源 `/api`，nginx 转发到后端。
- **直连后端域名**：构建前在 `.env.production` 设 `VITE_API_BASE=https://api.你的域名`，
  并确保后端 CORS 放行该前端域名（后端默认 `*`，生产建议在 `middleware.DefaultCORSConfig` 收紧）。

## 目录结构

```
src/
  main.js              # 入口（createApp + Pinia）
  App.vue              # 布局
  api/client.js        # apiFetch（token 注入 / 401 / baseURL）
  stores/              # Pinia：auth / sessions / chat / docs / skills
  composables/useSSE.js# SSE 流解析
  utils/               # markdown 渲染 / HTML 转义
  components/          # AuthModal / SideBar / SkillHub / DocViewer /
                       # MessageList / MessageBubble / ThinkPanel / ChatInput
  assets/styles.css    # 全局样式：极简中性主题（灰白底 + 单一黑色强调）
```

## 界面约定

- **版式**：左侧 264px 侧栏（可折叠，状态存 `localStorage`）+ 右侧居中内容列
  （`--col: 780px`），消息与输入框共用同一列宽。
- **侧栏分组**：`个人文件` / `本地文档库` 默认折叠，只留 `近期对话`；顶部放大镜
  按钮按标题过滤对话。
- **输入框**：Codex 式圆角卡片，`知识库` 开关、`Skill` 入口、上传按钮、当前模型
  都收在底部一行；模型名读 `/api/status` 的 `llm_model`，`is_mock` 为真时圆点变琥珀色。
- **配色**：纯灰白，无渐变/发光；语义色仅保留中断提示（红）与状态圆点。
