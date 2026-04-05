# Codego 分阶段规格与验收文档

本目录存放 **Claude Code 全功能 Go 实现**（见 [../GOLANG_CLAUDE_CODE_FULL_IMPLEMENTATION_PLAN.md](../GOLANG_CLAUDE_CODE_FULL_IMPLEMENTATION_PLAN.md)）各 Phase 的 **实现前强制** 文档。

**Phase 0** 已在仓库 **`rabbit-code/`**（与 `go.mod` 同级）落地：见 `PHASE00_SPEC_AND_ACCEPTANCE.md` §6；命令 **`rabbit-code`**，**不使用** `codego/` 目录。环境变量前缀 **`RABBIT_CODE_*`**。

## 命名规则

| 类型 | 格式 | 说明 |
|------|------|------|
| 功能规格与验收 | `PHASEXX_SPEC_AND_ACCEPTANCE.md` | `XX` = `00`…`12`，与主计划 Phase 0–12 对应。 |
| 单测与 E2E | `PHASEXX_E2E_ACCEPTANCE.md` | 每个 Phase 一份。 |
| 交互 UI 验收 | `PHASEXX_UI_ACCEPTANCE.md` | 仅当该 Phase 含 Bubble Tea/终端向导等交互时编写；见主计划 §2.5.3。 |

## 格式参考（ttrabbit）

- SPEC：`ttrabbit/docs/PHASE8_SPEC_AND_ACCEPTANCE.md`
- E2E：`ttrabbit/docs/PHASE8_E2E_ACCEPTANCE.md`
- UI：`ttrabbit/docs/PHASE8_CONTROL_UI_ACCEPTANCE.md`

## 流程

1. **编写/评审** 本 Phase 的 `PHASEXX_SPEC_AND_ACCEPTANCE.md`（**§2** 功能清单 + **§3** 验收；**§4** `src/` 路径对照；**§6** 迭代记录；**§0** 迭代前核对声明见 [`PHASE_ITERATION_RULES.md`](./PHASE_ITERATION_RULES.md)）。  
2. **编写** `PHASEXX_E2E_ACCEPTANCE.md`（`go test` 范围 + E2E 勾选步骤）。  
3. 若含 UI：**编写** `PHASEXX_UI_ACCEPTANCE.md`（逐步操作与预期）。  
4. **§0 三项门槛满足后**再开始本 Phase 代码合入。

**强制迭代规则**（迭代前门槛 + 迭代中执行计划与逐项 commit）：**[`PHASE_ITERATION_RULES.md`](./PHASE_ITERATION_RULES.md)**（纳入 git，见 `.gitignore` 例外）。**各 Phase 代码与交付的迭代记录写在对应 `PHASEXX_SPEC_AND_ACCEPTANCE.md` §6**；`PHASE_ITERATION_RULES.md` 文末修订表 **只记录规则文稿自身** 的修改。**Phase 5** SPEC（含 §6）与 E2E §0 见同目录对应文件（`PHASE05_SPEC_AND_ACCEPTANCE.md` 已设 git 例外以便跟踪 §6）。

## 本目录文件一览

| Phase | SPEC | E2E | UI |
|-------|------|-----|-----|
| 0 | `PHASE00_SPEC_AND_ACCEPTANCE.md` | `PHASE00_E2E_ACCEPTANCE.md` | — |
| 1 | `PHASE01_SPEC_AND_ACCEPTANCE.md` | `PHASE01_E2E_ACCEPTANCE.md` | `PHASE01_UI_ACCEPTANCE.md` |
| 2 | `PHASE02_SPEC_AND_ACCEPTANCE.md` | `PHASE02_E2E_ACCEPTANCE.md` | `PHASE02_UI_ACCEPTANCE.md` |
| 3 | `PHASE03_SPEC_AND_ACCEPTANCE.md` | `PHASE03_E2E_ACCEPTANCE.md` | — |
| 4 | `PHASE04_SPEC_AND_ACCEPTANCE.md` | `PHASE04_E2E_ACCEPTANCE.md` | — |
| 5 | `PHASE05_SPEC_AND_ACCEPTANCE.md` | `PHASE05_E2E_ACCEPTANCE.md` | — |
| 6 | `PHASE06_SPEC_AND_ACCEPTANCE.md`（§0/§4/§6 迭代前基线 **2026-04-01**） | `PHASE06_E2E_ACCEPTANCE.md` | `make test-phase6` |
| 7 | `PHASE07_SPEC_AND_ACCEPTANCE.md` | `PHASE07_E2E_ACCEPTANCE.md` | —（权限/MCP 弹层验收见 Phase 9 UI） |
| 8 | `PHASE08_SPEC_AND_ACCEPTANCE.md` | `PHASE08_E2E_ACCEPTANCE.md` | — |
| 9 | `PHASE09_SPEC_AND_ACCEPTANCE.md` | `PHASE09_E2E_ACCEPTANCE.md` | `PHASE09_UI_ACCEPTANCE.md` |
| 10 | `PHASE10_SPEC_AND_ACCEPTANCE.md` | `PHASE10_E2E_ACCEPTANCE.md` | `PHASE10_UI_ACCEPTANCE.md` |
| 11 | `PHASE11_SPEC_AND_ACCEPTANCE.md` | `PHASE11_E2E_ACCEPTANCE.md` | —（插件若带 TUI 则另补 `PHASE11_UI_ACCEPTANCE.md`） |
| 12 | `PHASE12_SPEC_AND_ACCEPTANCE.md` | `PHASE12_E2E_ACCEPTANCE.md` | `PHASE12_UI_ACCEPTANCE.md` |

## 索引表

见主计划 [§9.1 阶段文档索引](../GOLANG_CLAUDE_CODE_FULL_IMPLEMENTATION_PLAN.md#91-阶段文档索引rabbit-codedocsphases)。

## 还原树 `feature('…')` 与 Phase 的对应

- **全量映射表**：[../SOURCE_FEATURE_FLAGS.md](../SOURCE_FEATURE_FLAGS.md)（§2 按标志，§3 按 Phase）。
- **主计划**：§2.6；每个 **`PHASEXX_SPEC_AND_ACCEPTANCE.md`** 在 **§2 功能清单**（`P#.F.*` 等行）与 **§3 验收标准**（`AC#-F*` 等）中列出本 Phase 标志子集，与上表一致。
