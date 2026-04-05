# Phase 6 单测与 E2E 验收清单

与 `PHASE06_SPEC_AND_ACCEPTANCE.md` 对应。

---

## 0. 迭代前说明（[PHASE_ITERATION_RULES.md](./PHASE_ITERATION_RULES.md)）

- **SPEC §0、§4、§6** 须在首次大规模代码合入前就绪（**2026-04-01** 已填基线）。
- **工具语义（全量 / 非 headless 子集）**：与 **`PHASE06_SPEC_AND_ACCEPTANCE.md`** 文首 **「工具实现原则：全量上游对齐」** 一致；单测与后续 E2E fixture 须覆盖与 **`src/tools/*`** 对齐的主要分支，**不得以 headless 为由删减**上游 **`call`** 已有路径（呈现层 UI 除外）。
- **Phase 6 门禁**：**`make test-phase6`**（模块根 **Makefile**），等价于对 **`internal/tools/...`** 执行 **`go test -race -count=1`**；与 **`PHASE06_SPEC_AND_ACCEPTANCE.md` §3** 一致。

---

## 1. 单测

```bash
make test-phase6
```

（等价：`go test ./internal/tools/... -race -count=1`。）

| 范围 | 覆盖内容 |
|------|----------|
| 每工具子包 | **Run** 成功、权限拒绝、非法 JSON 输入（**AC6-1**） |
| **`internal/tools/registry`** | **`RegisterMCP` / `UnregisterMCP` / `ByName`（含 alias）/ `RunTool`**；实现 **`query.ToolRunner`**（见 **`registry_test.go`**）；含 **`Read`** 内置路由测 |
| **`internal/tools/filereadtool`** | **`FileRead.Run`**：文本成功读、**`.ipynb`**、**`.png`**、**`MapReadResultForMessagesAPI`**（**`text`/`notebook`/`file_unchanged`/pdf/parts/image**）；坏 JSON/空路径、**`.exe`** 拒绝、**`/dev/zero`**（非 Windows）、**offset** 大于行数（空内容短回包） |
| **`internal/tools/greptool`** | **`Grep.Run`**：严格 JSON、缺 **`pattern`** / 非法 **`output_mode`**；**`MapGrepToolResultForMessagesAPI`**（content/count/files、分页）；无 **`rg`** 时相关用例 **Skip** |

---

## 2. E2E

- [ ] fixture 仓库：read → 断言文件内容。
- [ ] fixture：write/edit 后磁盘 hash。
- [ ] fixture：bash（受权限 allow）执行 echo 类安全命令。
- [ ] 其余工具按 SPEC **§2** / 附录扩展勾选（与 PARITY 同步）。

---

## 3. 引用

- SPEC：`PHASE06_SPEC_AND_ACCEPTANCE.md`
- 主计划：`../GOLANG_CLAUDE_CODE_FULL_IMPLEMENTATION_PLAN.md` §6.6
