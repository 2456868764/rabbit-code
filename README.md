# rabbit-code

Go 实现 Claude Code 能力全集（规划见 `docs/GOLANG_CLAUDE_CODE_FULL_IMPLEMENTATION_PLAN.md`）。**模块根目录即本仓库 `rabbit-code/`，不再使用单独的 `codego/` 目录。**

## Phase 0

```bash
cd rabbit-code
go test ./... -count=1
make build    # 输出 bin/rabbit-code
./bin/rabbit-code version
```

- **Lint**：需安装 [golangci-lint](https://golangci-lint.run/)，然后 `make lint`。
- **CI**：父仓库（`bot/`）使用 `.github/workflows/rabbit-code-ci.yml`；若本目录单独作为 git 根，请按该文件顶部说明复制/调整 workflow。
- **验收**：`docs/phases/PHASE00_SPEC_AND_ACCEPTANCE.md`、`PHASE00_E2E_ACCEPTANCE.md`。

## 二进制名

- 命令行工具：**`rabbit-code`**（`cmd/rabbit-code`）。
