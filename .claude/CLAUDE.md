# Ai-curton 项目指令

## 技术栈
- 后端：Go 1.22 + Gin + GORM + MySQL + Redis
- 前端：Vue 3 + TypeScript + Pinia + Tailwind CSS
- AI：多模型分发（智谱/通义/Kimi）+ DAG 工作流引擎

## 编程规范
开发新功能或修改代码时，请遵循 `.claude/skills/coding-standards/SKILL.md` 中的编程规范。

## 编译验证
后端改动后必须执行 `go1.22.12 build ./...` 验证编译通过（项目使用 Go 1.22.12）。
