# Claude 开发备忘

- 本项目使用 **docker compose** 部署
- 编译命令：`./start.sh start --build`

## 代码编辑规则

- **只改需要改的行**，不要为了对齐而修改其他行
- 冒号后面不对齐不会有任何问题，语言不关心格式对齐
- 前端的所有显示 label, wording 这些，都要做多语言支持，多语言在 `web/src/i18n/translations.ts` 中，一定不能直接在 tsx 文件中写 raw string
- Never run ./start.sh, 用户会自己在 terminal 运行这个命令