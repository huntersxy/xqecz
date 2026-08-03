---
name: codebase-lean
description: 降低大仓库的 token 消耗并安全瘦身代码（保持功能不变）。Use when the user wants to slim down / 轻量化 a codebase, cut AI token or context overhead, audit code size, remove dead code or duplication, or make Codex work more token-efficiently in a large repo.
---

# Codebase Lean — 省 token 的代码瘦身

目标：在不改变任何外部行为的前提下，减少代码体积与每次任务的 token 消耗。

## 省 token 工作模式（每次遵守）

1. 先定位后读取：用 `rg` / `rg --files` / `Select-String` 定位符号，只读相关行区间；禁止整文件输出大文件（>200 行先定位再读）。
2. 最小上下文：先读 AGENTS.md 与契约文件，再只读改动涉及的代码；不要通读整个模块目录。
3. 忽略噪音：node_modules、dist、build、lock 文件、生成代码（proto/gen、*.pb.go）、.git、日志。
4. 精简输出：只给改动的 diff 或关键结论，不重复已确认信息；commentary 每次 ≤3 句。
5. 能跑脚本就不贴代码：统计/检索用脚本和命令行完成。

## 瘦身流程（功能不变）

1. 审计：从仓库根目录运行 `python <skill目录>/scripts/repo_stats.py`，锁定最大手写文件。
2. 按安全顺序削减：
   - 死代码：未使用的导出、分支、注释掉的代码块、无用依赖（以 typecheck / go vet / 无未用变量为准）。
   - 重复：同一逻辑出现 ≥2–3 次才提取共用函数/组件；一次性重复不值得抽象。
   - 冗余：合并重复 DTO/类型定义、删除多余包装层、简化嵌套表达式。
   - 大文件拆分：仅当提升可导航性时才拆，不为拆而拆。
3. 红线（禁止）：
   - 改对外契约：API 路径与字段、gRPC proto、DB schema、env 语义、前端唯一接口契约。
   - 删行为：软删除、降级分支、错误处理、缓存/锁语义。
   - 无关的大范围重构或重排格式。
4. 验证（必做）：逐包 typecheck / 单测 / 构建；在 xqecz 仓库的验证命令见 references/xqecz.md。
5. 提交：小步提交；汇报 `git diff --stat` 与净删减行数。

## 参考

- 仅当工作目录是 `D:\xqecz\xqecz`（或其子目录）时，加载 references/xqecz.md 获取契约速查与验证命令。
