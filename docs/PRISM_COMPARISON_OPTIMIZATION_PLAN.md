# Veilium Browser 对照 Prism Browser 的优化开发方案

状态：**执行提案 / 默认自主推进 / 不替代当前阶段授权**  
更新日期：2026-08-19  
执行模式：**Autonomous by default，Owner Gate only when necessary**

对照基线：

- Prism Browser Community：`DFarm6/Prism-Browser-Community`，`main`；
- Veilium Browser：`knownothing20/veilium-browser`，PR #59，分支 `agent/handoff-m5-3`。

## 1. 文档目的

本文基于两个项目的实际代码重新对照，形成 Veilium 后续优化开发的统一方案，并明确开发过程中何时可以自主推进、何时必须由项目所有者决策。

本文回答四个问题：

1. Prism 已经实现、Veilium 当前真正缺少的核心能力是什么；
2. Veilium 哪些架构应保留，哪些代码必须补强；
3. 如何以最少返工形成真正的 Fingerprint Browser 闭环；
4. 如何让 Agent 连续开发到底，而不是每做一步就暂停询问。

本文不修改 `PRODUCT.md`、`ROADMAP.md`、`PHASE_05.md` 或 `STATUS.md` 的当前授权。当前任务仍然是完成 PR #59 的真实浏览器、中文界面和完整功能回归验证。Phase 5 正式关闭前，不得在当前 PR 中新增第二个 reviewed Provider 或新的 fingerprint capability 声明。

参考 Prism 时必须遵守 clean-room 原则：可以研究需求、数据流、测试矩阵、用户体验和公开接口，不直接复制应用代码，也不未经来源、许可证和安全审查就移植 Chromium patch。

---

## 2. 总体结论

两个项目的优势并不重叠：

### Prism 的优势

Prism 已经形成完整的指纹浏览器产品闭环：

- 定制 Fingerprint Chromium；
- Profile 到 Chromium 参数映射；
- Root Seed 与稳定 Surface Identity；
- Hardware Profile；
- Canvas、WebGL、Audio、DOMRect、Fonts、Speech、WebGPU 等内核能力；
- Proxy 位置与语言、时区、地理位置联动；
- Fingerprint Matrix 和多种运行审计；
- Cookie、Extension、Backup、Update、Automation、Scheduler 和 MCP 产品入口。

### Veilium 的优势

Veilium 已经形成更严格的浏览器运行平台：

- Provider Contract；
- exact archive、executable 和 package-tree 校验；
- OS Credential Vault，无 plaintext fallback；
- HTTP、HTTPS、SOCKS5 Proxy Bridge；
- Xray 和 sing-box 受控运行；
- Browser Supervisor、CDP readiness 和进程树清理；
- Identity Evidence、Network Evidence 和 Compatibility；
- Operation Journal、Locks、Cancellation、Snapshot、Restore、Trash、Rollback 和 Reconciliation；
- 更明确的 fail-closed、权限和数据边界。

### 正确方向

Veilium 不应该改成 Electron，也不应该复制 Prism 的整体项目。

正确目标是：

> 在 Veilium 现有 Provider、Evidence、Security、Proxy、Supervisor 和 Lifecycle 架构上，补齐真正可执行的 Fingerprint Chromium、Identity Template、Root Seed、Network Identity Advisor 和完整验证矩阵。

最终产品应当是：

```text
Prism 已验证的指纹内核产品能力
                  +
Veilium 更严格的 Provider / Evidence / Security / Lifecycle 架构
```

---

## 3. 源码级对照

| 维度 | Prism Browser Community | Veilium Browser | 优化判断 |
| --- | --- | --- | --- |
| 桌面架构 | Electron + React + TypeScript | Wails + Go + React + TypeScript | 保留 Veilium，不迁移 Electron |
| 浏览器内核 | Fingerprint Chromium 144，包含大量 Chromium/Blink patch | 一个 exact reviewed stock Chromium Snapshot 152 | Veilium 最大缺口是真正的 Fingerprint Kernel |
| Provider 模型 | 主要按 Kernel 版本和 fingerprint 标志判断 | Provider Contract v2，区分 reviewed/custom/legacy/disabled/invalid | 保留 Veilium 的契约模型 |
| 指纹配置执行 | 自定义 Chromium 参数与 patch 真正接收配置 | 配置、Capability 和 Args 框架已存在，但 stock Provider 不支持高级能力 | 新增独立 Fingerprint Provider，不能抬高 stock Provider 声明 |
| 身份模板 | Host-native、seeded GPU、固定硬件模板 | 用户仍可独立组合 CPU、GPU、Screen、Language、Timezone | 增加少量内置 Identity Template |
| Root Seed | Seed 参与 Canvas、WebGL、Audio、DOMRect、GPU bucket | 已有 Profile Seed，但缺少版本化整套派生规则 | Root Seed 应成为身份唯一入口 |
| 网络身份 | Proxy 结果可派生语言、时区和地理位置 | Proxy Diagnostics 和 Network Evidence 更严格，但没有 Advisor | 增加“观察—建议—用户确认” |
| Secret | Electron safeStorage 路径 | OS Keyring，无明文 fallback | 保留 Veilium 边界 |
| Evidence | 多个 audit 和 fingerprint matrix | Provider/Binary/Context/Network Evidence 更结构化 | 将矩阵思想接入现有 Evidence |
| Lifecycle | Profile 复制、备份、回收站、Workspace 迁移 | Journal、Locks、Snapshot、Restore、Rollback、Recovery | Veilium 已更强，不重写 |
| 产品完成度 | Cookie、Extension、Updater、Automation 等更完整 | PR #59 正在完成中文工作区、Portable Definition、Template、Batch、Storage | 先完成当前产品，再做 Kernel 主线 |

---

## 4. 当前代码中已经确认的关键问题

### 4.1 Reviewed Kernel Catalog 假设永远只有一个 Provider

当前 `internal/kernelrelease/catalog.go`：

- 写死 `ProviderID = official-chromium-snapshot-win64`；
- 校验 manifest 必须只有一个 release；
- 把 Windows amd64、下载来源和 archive layout 固定在单一实现中。

`internal/fingerprint/official.go` 同样要求 release 数量恰好为一个，否则 official Provider 进入 invalid。

这在 Phase 4 建立单一可信 Snapshot 时是正确的，但阻止未来并存：

```text
Official Stock Chromium
Veilium Fingerprint Chromium
Custom Chromium
Legacy Chromium
```

后续第一项基础重构必须是 Multi-Provider Reviewed Kernel Catalog，并保证现有 Provider ID、Profile、Kernel Record 和 Evidence 完全兼容。

### 4.2 FingerprintConfig 到 Chromium 的映射不完整

Veilium 已定义 `deviceMemoryGb`，Validator 和 Capability 也会检查，但 `internal/fingerprint/args.go` 没有生成对应 Kernel 参数。

Screen 也存在概念混用：当前 `screenWidth` 和 `screenHeight` 主要用于 `--window-size`，但窗口尺寸不等于网页读取到的：

- `screen.width`；
- `screen.height`；
- `screen.availWidth`；
- `screen.availHeight`；
- `devicePixelRatio`。

后续必须明确拆分：

- Window Plan：管理实际窗口；
- Screen Identity：控制网页可见屏幕信息；
- Device Memory；
- Hardware Concurrency；
- Language；
- Accept-Language；
- Timezone；
- GPU / WebGL Identity。

UI 有字段或普通 Chromium 接受参数，都不能被当作 capability 已验证。

### 4.3 身份配置仍然容易拼出矛盾设备

PR #59 已经把创建环境改造成中文分步流程，但用户仍可独立组合：

- Platform；
- Brand；
- Language；
- Timezone；
- Screen；
- CPU；
- Memory；
- GPU；
- WebRTC；
- Canvas、Audio、ClientRects 等模式。

当前 Validator 主要检查字段范围，不能判断整个组合是否像一台合理设备，例如：

- Windows 配 Apple GPU；
- CPU、内存、GPU 和屏幕档位严重冲突；
- 美国 Proxy 配中国语言和时区；
- GPU、WebGL、Canvas、Fonts 和 Audio 来自不同身份模型。

普通用户应该优先选择 Identity Template，高级用户才展开原始字段。

### 4.4 默认 Provider、Timezone 和展示逻辑存在漂移

当前前端默认值和显示逻辑仍有硬编码倾向：

- 新 Profile 默认可能进入 `custom-chromium`；
- 默认 Timezone 可能与真实本机或 Proxy 无关；
- Provider label 在前端按少数 ID 判断，official Provider 可能被错误显示为“本机 Chromium”；
- 前端 `profileHealth()` 只做少量表面判断，没有覆盖 Provider、Evidence 和 Lifecycle 全状态。

后续应改为：

1. Provider descriptor、名称、信任状态由后端返回；
2. 默认选择当前平台已安装、完整性通过、信任等级最高的 Kernel；
3. 没有 reviewed Kernel 时明确显示 custom/unverified；
4. 默认 Timezone 来自本机或明确用户选择，不固定某个国家；
5. 前端不自行拼装 Environment Health。

### 4.5 Proxy 很强，但还没有成为 Network Identity

Veilium 已经具备：

- Route classification；
- Proxy Bridge；
- Xray / sing-box；
- Exit IP；
- Latency；
- DNS route；
- WebRTC policy；
- Browser-side Network Evidence。

缺少的是把这些结果转化为有来源、有时效、可确认的身份建议：

```text
Proxy Observation
  ├─ IP / Country / Region / City
  ├─ Timezone candidates
  ├─ Locale / Accept-Language candidates
  ├─ Geolocation candidates
  ├─ Confidence / Conflict
  └─ ObservedAt / Expiry

            ↓

Identity Recommendation
            ↓

用户点击“应用建议”后才修改 Profile
```

禁止在 Proxy 变化后静默修改 Profile。

### 4.6 Evidence 基础很好，但覆盖面和时间维度不足

Veilium 已支持 Top-level、Iframe、Worker，并采集：

- User Agent；
- Platform；
- Languages；
- Timezone；
- Hardware Concurrency；
- Screen；
- Window；
- WebRTC；
- Canvas、WebGL、Audio、ClientRects Digest。

后续还需增加：

- Device Memory；
- Accept-Language；
- GPU Vendor / Renderer；
- WebGL Parameters；
- WebGPU Adapter；
- Fonts；
- Speech Voices；
- Restart Stability；
- Different-seed Separation；
- Evidence Freshness；
- Kernel Tamper Downgrade。

这些能力应升级现有 Evidence schema、Evaluation 和 Compatibility，不再建立第二套测试系统。

### 4.7 Environment Health 必须由后端统一计算

当前前端健康判断无法覆盖：

- Lifecycle；
- Kernel Integrity；
- Provider Trust；
- Capability Compatibility；
- Credential / Adapter Availability；
- Proxy Diagnostics；
- Identity Evidence；
- Network Evidence；
- Evidence Freshness；
- Recovery-required；
- Draft / Limited 状态。

后续应由后端输出版本化 `EnvironmentReadiness`，前端只负责展示。

---

## 5. 目标架构

```text
Browser Profile
      │
      ├── Identity Template
      │        └── Root Seed + Derivation Version
      │
      ├── Network Route
      │        ├── Proxy / Adapter / Credential Vault
      │        └── Network Identity Observation
      │
      └── Kernel Reference
                │
                ▼
        Provider Contract
                │
        Exact Kernel Registry
                │
        Launch Planner / Args
                │
        Browser Supervisor
                │
        Real Chromium Runtime
                │
        Identity Evidence + Network Evidence
                │
        Compatibility + Environment Readiness
                │
        Lifecycle / Snapshot / Restore / Portability
```

核心规则：

> UI 配置、Provider Contract、Kernel 文件、Launch Args 和真实网页结果全部一致，能力才可以标记为 verified。

---

## 6. 默认自主推进规则

### 6.1 总规则

除非触发本文定义的 Owner Gate 或强制升级条件，Agent 必须自主选择风险最低、改动最小、最符合现有架构和验收标准的实现继续推进，不得因普通技术选择暂停询问。

标准流程：

```text
读取 Source of Truth
        ↓
实现当前 Stage
        ↓
静态检查 / Unit Test / Integration Test
        ↓
真实运行验证
        ↓
失败分析与最小修复
        ↓
重新验证
        ↓
记录决策与限制
        ↓
自动进入下一 Stage
```

### 6.2 以下事项不得停下来询问 Owner

- 文件如何拆分；
- 函数、类型和内部参数命名；
- 局部架构重构；
- 测试失败后的修复方式；
- Chromium patch 小范围 rebase；
- 测试 fixture 和 mock 的组织；
- UI 组件布局和普通交互细节；
- 文档同步；
- 不改变外部合同的 schema 内部实现；
- 兼容性 bug；
- lint、format、typecheck、race、build 问题；
- 在既定安全边界内升级或锁定依赖；
- 将失败 capability 暂时标记为 unsupported/unverified；
- 选择更保守、更小范围的实现。

Agent 不得反复询问：

- “是否继续”；
- “要不要修改这个文件”；
- “A 和 B 选哪个”，如果 A/B 只是实现细节；
- “测试失败是否先修复”；
- “是否进入下一阶段”，如果当前退出条件已满足。

### 6.3 默认决策优先级

遇到技术分歧时按以下顺序自行选择：

1. Safety、Legality、Clean-room；
2. 不丢数据、不泄露 Secret；
3. 不弱化 Sandbox、Provider Trust、Evidence 或 fail-closed；
4. 保持现有 Profile、Kernel、Portable Definition 和 Lifecycle 兼容；
5. 最小可验证范围；
6. 可回滚、可维护；
7. 产品体验；
8. 实现便利。

### 6.4 Capability 隔离规则

单一 capability 失败不得自动阻塞整个计划。

例如 WebGPU 或 Fonts 未通过时：

```text
记录失败原因
→ 保持 unsupported/unverified
→ 不对外宣称
→ 继续完成 Screen / CPU / Canvas / Audio / WebGL 等已通过能力
```

只有失败破坏整个 Provider、安全或数据一致性时，才阻止发布。

### 6.5 非阻塞进度更新

Agent 可以在每个 Stage 结束时更新 `STATUS.md`、实施记录或 PR 描述，但这些更新是报告，不是请示。

更新应说明：

- 已完成；
- 验证结果；
- 已知限制；
- 自动选择的关键方案；
- 下一阶段。

---

## 7. Owner Gate 与强制升级条件

### 7.1 必须保留的 Owner Gate

#### Owner Gate 1 — 合并决定

Agent 可以把 PR #59 和后续实现做到 Ready for Review，但是否合并到 `main` 由 Owner 决定。

#### Owner Gate 2 — 重大产品方向变化

只有在实际验证证明既定目标不可行，并且替代方案会显著改变产品方向时才询问，例如：

- 计划使用较新 Chromium，但稳定实现只能停留在明显更旧的主版本；
- 必须删除某项既有核心能力才能实现 Fingerprint Provider；
- Windows-only V1 必须改成完全不同的平台策略；
- 必须引入远程服务、Telemetry 或 Cloud 才能继续。

如果只是 patch 需要调整、版本需要小幅更换或能力需要降级为 unsupported，Agent 自主处理。

#### Owner Gate 3 — License 与公开分发

以下事项必须由 Owner 决定：

- Veilium 项目许可证；
- 是否公开分发编译后的 Fingerprint Chromium；
- 商标和品牌使用；
- 二进制分发方式；
- 商业版与开源版边界；
- 需要法律判断的第三方 patch 或组件。

### 7.2 强制升级条件

遇到以下情况必须停止相关不可逆动作并请求 Owner 决策：

- 需要不可逆删除或破坏用户现有数据；
- 需要导出、明文保存或扩大 Secret 暴露；
- 需要禁用 Chromium Sandbox 或弱化进程隔离；
- 需要把 unsupported/unverified 能力伪装成 verified；
- 需要扩大到公网控制面、Remote API、Telemetry 或 Cloud Sync；
- 发现许可证、来源或再分发权存在实质不确定性；
- 方案用途转向绕过平台规则、欺诈、账号农场或未授权访问；
- 两个可行方案会形成明显不同的商业或产品定位。

### 7.3 不属于升级条件的失败

以下情况由 Agent 自主解决：

- 编译失败；
- Chromium API 变动；
- patch offset 失效；
- 某个测试不稳定；
- 第三方检测网站分数波动；
- 单项 capability 无法在当前版本验证；
- UI 需要重构；
- 测试依赖缺失；
- 运行时性能需要优化；
- 需要增加内部兼容层。

---

## 8. 分阶段开发方案

## Stage 0 — 完成 PR #59 和 Phase 5

优先级：P0  
执行模式：**完全自主，直到 Ready for Review**

必须完成：

- 真实 Chromium start/readiness/stop/process-tree cleanup；
- 1366×768 和 1920×1080 中文主要流程检查；
- Kernel、Credential、Proxy、Recovery、Portability、Template、Batch、Storage 和 Evidence smoke test；
- 应用重启后持久化验证；
- 证明管理界面语言不会修改 Profile 的 language、timezone、platform 或 fingerprint；
- 修复所有发现的问题；
- 同步 `STATUS.md` 和 PR 描述；
- 完成 Phase 5 closure 所需记录。

退出条件：

- PR #59 达到 Ready for Review；
- Owner Gate 1 决定是否合并；
- 合并前不得进入 Stage 1 产品代码。

## Stage 1 — Multi-Provider Reviewed Kernel Foundation

优先级：P1  
复杂度：M  
执行模式：**自主完成**

目标：移除单一 reviewed Provider 假设，但不改变任何现有用户行为。

主要改动：

- `internal/kernelrelease/catalog.go`
  - 支持多个 Provider 和多个 exact release；
  - 复合索引：Provider ID + revision + version + OS + arch；
  - 每种 Provider 使用明确的 source/layout validator；
  - 保留 strict decode、unknown field 和 trailing data 拒绝。
- `internal/kernelrelease/releases.json`
  - 升级 schema；
  - 保持原 Provider ID、digest 和 package identity；
  - 增加迁移和 downgrade 行为。
- `internal/fingerprint`
  - Provider definition 按 ID/revision 查询；
  - 不再假设 Release 总数为一；
  - stock Provider 高级 capability 继续为 unsupported。
- Kernel Installer、Registry、Portable Dependency Matching、Frontend 类型同步支持多 Provider。

验收：

- 旧 Kernel 和 Profile 无需用户迁移即可工作；
- 现有 exact package identity 和 Evidence 仍通过；
- 第二个测试 Provider fixture 不污染第一个；
- 不产生新的 verified fingerprint claim。

## Stage 2 — Veilium Fingerprint Chromium V1

优先级：P1  
复杂度：XL  
执行模式：**自主开发；只有重大方向变化才触发 Owner Gate 2**

目标：新增独立 Windows amd64 Fingerprint Provider。

V1 建议能力：

- Platform；
- Browser Brand；
- Language；
- Accept-Language；
- Timezone；
- Screen / available screen / device scale；
- Hardware Concurrency；
- Device Memory；
- Root-seeded Canvas；
- Root-seeded Audio；
- Root-seeded ClientRects / DOMRect；
- WebGL / GPU Identity；
- WebRTC proxy-only policy。

V1 暂缓：

- 完整 Fonts Catalog；
- Speech Voices；
- WebGPU Template；
- macOS reviewed Provider；
- 多个 Chromium 主版本并行维护。

实现要求：

- 使用 exact Chromium baseline，不解析 moving latest；
- 参数由版本化 Contract 定义；
- 补齐 Device Memory 和 Screen Identity；
- 每个 capability 都对应代码、Provider declaration、测试和 Evidence；
- archive、executable、package tree、provenance 全部校验；
- stock 与 fingerprint Provider 在产品中清晰区分；
- custom Kernel 永远不能手工升级为 reviewed。

基线自动选择规则：

1. 优先选择能够稳定构建、维护和验证的 exact Chromium 版本；
2. 不为了追最新版本牺牲 Sandbox、Evidence 或可维护性；
3. 如果候选版本差异只是普通技术取舍，Agent 自主选择；
4. 只有最终可行版本与既定产品定位发生重大偏差时触发 Owner Gate 2。

退出条件：

- 每一项公开 capability 都通过 exact binary 真实验证；
- 未通过项保持 partial/unverified/unsupported；
- Provider 可安装、启动、停止、校验、降级和恢复。

## Stage 3 — Identity Template 与 Root Seed

优先级：P2  
复杂度：M  
执行模式：**自主完成**

目标：让普通用户选择合理设备，而不是手工拼装矛盾字段。

第一版模板：

- Windows · 本机硬件；
- Windows · 标准办公设备；
- Windows · 主流桌面设备；
- Windows · 高性能桌面设备；
- Advanced · 自定义配置。

Root Seed 使用版本化派生：

```text
Root Seed
  ├─ Canvas sub-seed
  ├─ Audio sub-seed
  ├─ ClientRects sub-seed
  ├─ WebGL / GPU bucket
  ├─ Fonts policy seed（后续）
  └─ Speech / WebGPU seed（后续）
```

规则：

- 同 Profile + 同 derivation version 必须稳定；
- Clone 默认生成新 Root Seed；
- Preserve Identity 只能在明确高级流程中使用；
- Template 更新不能静默改变既有 Profile；
- Template 只生成配置，不赋予 Provider Trust 或 Evidence。

## Stage 4 — Network Identity Advisor

优先级：P2  
复杂度：M  
执行模式：**自主完成**

目标：把 Proxy 和 Network Evidence 转化为可审查的身份建议。

功能：

- 生成带时效的 Network Identity Observation；
- 给出 Timezone、Language、Accept-Language 和可选 Geolocation 建议；
- 显示来源、Confidence、Conflict 和更新时间；
- Profile 与网络观测冲突时提供明确提示；
- 用户点击“应用建议”后才修改 Profile；
- Proxy 出口变化使旧 Observation 失效；
- Advisor 不替代 Network Evidence。

明确不做：

- Proxy Pool；
- Proxy Rotation；
- 定时切换；
- 无人值守身份修改；
- Account Farming。

## Stage 5 — Evidence V2 与 Fingerprint Matrix

优先级：P1  
复杂度：L  
执行模式：**自主完成**

新增观测：

- Device Memory；
- Accept-Language；
- GPU Vendor / Renderer；
- WebGL Parameters；
- 可用时的 Fonts、Speech、WebGPU；
- Provider Capability Revision；
- Identity Derivation Version。

新增矩阵：

1. Context Consistency：Top-level / Iframe / Worker；
2. Restart Stability：同一 Profile 重启后身份稳定；
3. Seed Separation：不同 Seed 的目标 Surface 分离；
4. Template Coherence：CPU、Memory、GPU、Screen、Platform、Language、Timezone 不矛盾；
5. Network Coherence：Route、Exit IP、DNS、WebRTC 与 Profile Policy 一致；
6. Tamper Downgrade：修改受保护 Kernel 文件后立即降级；
7. Evidence Freshness：Kernel、Provider、Profile 或 Route 变化后旧 Evidence 失效。

第三方检测网站只作为补充，不替代可重复的本地 Evidence。

## Stage 6 — Environment Readiness 与 UI 修正

优先级：P2  
复杂度：M  
执行模式：**自主完成**

后端输出版本化 `EnvironmentReadiness`：

```text
ready
warning
blocked
recovery-required
unverified
```

组成项：

- Lifecycle；
- Kernel Integrity；
- Provider Trust；
- Capability Compatibility；
- Credential / Adapter Availability；
- Proxy Diagnostics；
- Identity Evidence；
- Network Evidence；
- Evidence Freshness。

产品修正：

- Provider label 不在前端硬编码；
- official Provider 不再显示成“本机 Chromium”；
- 默认 Provider 根据平台、安装、完整性和 Trust 选择；
- 默认 Timezone 不固定为 `America/Los_Angeles`；
- Profile Editor 以 Template 为主，高级字段渐进展开；
- 环境列表优先展示“可打开 / 需处理 / 已阻止”及原因；
- Evidence 技术细节保留在高级入口。

## Stage 7 — 核心闭环之后的产品补全

优先级：P3  
执行模式：**需要后续阶段授权，但开发过程仍默认自主**

候选功能：

- Cookie 查看、导入和导出；
- Extension 管理；
- 更简单的 Backup / Migration；
- Controlled Automation；
- Scheduler / MCP；
- Release Signing；
- Updater；
- SBOM；
- Reproducible Build。

不得为了追赶 Prism 的功能数量而提前进入这些工作。

---

## 9. 最小分支与 PR 策略

为减少流程停顿和分支泛滥：

1. Stage 0 继续使用当前 PR #59，不创建额外 handoff PR；
2. Phase 5 关闭后，Stage 1–6 默认使用一个专用长期 Draft PR；
3. 分支建议：`agent/fingerprint-platform-v1`；
4. 在同一 PR 内使用依赖有序、可独立审查的 atomic commits；
5. 不为每个小 capability 创建 Issue、Branch 或 PR；
6. 不创建 process-only、temporary、autofix 或 diagnostic workflow；
7. Agent 自主完成所有内部 Gate 后，将 PR 标记为 Ready for Review；
8. Owner 只在最终合并点进行决策。

如果单一 PR 已经无法可靠审查或回滚，Agent 可以在不改变产品方向的前提下拆成少量依赖 PR，但不得因此暂停等待普通实现确认；只需在状态文档中记录拆分原因。

---

## 10. 代码改动地图

| 模块 | 主要职责 |
| --- | --- |
| `internal/kernelrelease` | Multi-Provider exact release catalog、provenance、package identity |
| `internal/kernelinstaller` | Provider-aware install、verify、atomic activation、rollback |
| `internal/kernel` | Multi-Provider Kernel registry 和 in-use protection |
| `internal/fingerprint/catalog.go` | Capability Contract、Provider revision、支持边界 |
| `internal/fingerprint/args.go` | 完整、版本化的 Profile → Kernel Args 映射 |
| `internal/fingerprint/validator.go` | 字段验证、Template coherence、fail-closed |
| `internal/domain/types.go` | Root Seed、derivation version、template reference |
| `internal/launch/planner.go` | Provider-aware launch plan，不把描述性字段当作已应用能力 |
| `internal/evidence` | Evidence V2、Restart、Seed、Context Matrix |
| `internal/networkevidence` | Route-bound browser Network Evidence |
| `internal/proxydiagnostics` | Proxy Observation 和 Advisor 输入 |
| `internal/desktop` | EnvironmentReadiness、统一后端产品状态 |
| `frontend/src/lib/model.ts` | 删除过度简化的前端健康推断 |
| `frontend/src/components/ProfileEditor.tsx` | Template-first 和 Network Recommendation |
| `frontend/src/components/ProfileTable.tsx` | Provider、Readiness 和 Blocker 展示 |
| `frontend/src/components/OfficialKernelCard.tsx` | 多 reviewed Provider 安装和限制说明 |
| `docs` | Provider、Kernel、Identity、Evidence、Security、Migration、Decision Log |

---

## 11. Provider 能力目标矩阵

| 能力 | Official Stock Chromium | Veilium Fingerprint V1 | Custom Chromium |
| --- | --- | --- | --- |
| Managed Launch | reviewed | reviewed | custom/unverified |
| Exact Package Integrity | reviewed | reviewed | 本地完整性记录 |
| Platform Override | unsupported | Evidence 通过后 verified | unsupported/unverified |
| Brand Override | unsupported | Evidence 通过后 verified | unsupported/unverified |
| Language / Accept-Language | 记录实际行为 | Evidence 通过后 verified | unverified |
| Timezone | unsupported | Evidence 通过后 verified | unsupported/unverified |
| Screen Identity | unsupported | Evidence 通过后 verified | unsupported/unverified |
| Hardware Concurrency | unsupported | Evidence 通过后 verified | unsupported/unverified |
| Device Memory | unsupported | Evidence 通过后 verified | unsupported/unverified |
| Canvas / Audio / ClientRects Seed | unsupported | Evidence 通过后 verified | unsupported/unverified |
| WebGL / GPU Identity | unsupported | Evidence 通过后 verified | unsupported/unverified |
| WebRTC Proxy-only | 不作高级声明 | Network Evidence 后 verified/partial | unverified |
| Fonts / Speech / WebGPU | unsupported | V2 或验证后再声明 | unsupported/unverified |

状态来自 Provider Contract 和 Evidence，不来自 UI 开关。

---

## 12. 完整验证矩阵

### 静态与合同

- Manifest strict decode；
- Unknown field 和 trailing data；
- Provider/revision/version/platform/arch 唯一性；
- Archive、Executable 和 Package Tree Digest；
- Capability schema 和 Args 生成；
- Unsupported fail-closed；
- 旧 Profile、Portable Definition、Kernel Record 兼容。

### 真实浏览器

- Exact binary start/readiness/stop/process-tree cleanup；
- UA / Client Hints / Platform；
- Language / Accept-Language / Timezone；
- Screen / Window / DPR；
- CPU / Device Memory；
- Canvas / Audio / ClientRects / WebGL；
- Top-level / Iframe / Worker；
- Same-seed restart stability；
- Different-seed separation；
- Unsupported 配置阻止保存或启动。

### 网络

- Direct baseline；
- HTTP / HTTPS / SOCKS5；
- OS Vault Credential Bridge；
- Xray / sing-box 受支持子集；
- Exit IP / DNS Route / WebRTC；
- Profile Identity Conflict；
- Proxy Exit 变化；
- 旧 Network Evidence 失效。

### 失败与安全

- Archive / Package Tamper；
- Missing Dependency；
- Disk Full / Permission / Rename / Flush；
- Cancellation / Shutdown / Interruption；
- CDP Readiness Failure；
- Process Leak；
- Secret 不进入 Log、Bootstrap、Report、Args 和 Export；
- 不使用 `--no-sandbox` 通过测试；
- Rollback 后原健康 Kernel 和 Profile 可恢复。

### 产品

- 1366×768；
- 1920×1080；
- 首次安装；
- 首次创建；
- 打开、关闭、诊断和修复；
- Template 和 Advanced Settings；
- Provider 限制和 Evidence 失败可理解；
- 管理界面语言不改变 Profile Identity。

---

## 13. 数据迁移与兼容

- 保留现有 Provider ID；
- Stock Provider 不因 Fingerprint Provider 出现而自动升级；
- 新 Profile 字段必须 optional 或有明确 migration；
- 旧 Profile 没有 template reference 时按 `legacy-custom` 解释；
- Template 更新不改变既有 Profile；
- Kernel 或 Provider revision 变化使旧 Evidence stale；
- Portable Definition 不携带本地 Kernel ID、路径、Secret、Browser Data、Trust 或 Evidence；
- Preserve Identity 仍创建新的本地 Profile ID 和目录；
- Downgrade 不得删除未知字段或乐观重写数据；
- 所有持久化改变都必须有 rollback 和 recovery 分析。

---

## 14. 安全、隐私与许可证

必须保留：

- Secret 只进入 OS Credential Vault；
- 无 plaintext/base64 fallback；
- Profile、Log、Args、Bootstrap、Evidence 和 Export 不包含代理密码；
- 本地控制面只绑定 loopback 并认证；
- Browser、Proxy Bridge、Xray、sing-box 由 Supervisor 管理；
- Kernel 和 Adapter 通过 exact package identity 管理；
- 不静默下载、更新或替换敏感 Runtime。

内核工作必须记录：

- Upstream Repository、Tag、Commit、License；
- 每个 Patch 的来源、目的、SHA-256、应用顺序和独立修改；
- Chromium 和第三方 Notices；
- Build Args、Toolchain 和 Artifact Manifest；
- 二进制再分发权；
- Clean-room Design Notes。

公开发布前必须触发 Owner Gate 3，明确 Veilium License、二进制分发、品牌和商业边界。

---

## 15. 明确不做

- 不承诺绕过风控、CAPTCHA 或平台规则；
- 不承诺账号存活率或检测规避率；
- 不做每次启动随机身份；
- 不做 Account Farming、Proxy Rotation 或批量无人值守启动；
- 不在 V1 同时覆盖 Windows、Linux、macOS 和多个架构；
- 不把第三方检测站评分当作唯一验收；
- 不把 Custom Chromium 自动提升为 reviewed；
- 不弱化 Sandbox、Secret、Evidence、Rollback；
- 不直接复制 Prism 应用代码或未经审查的 Patch；
- 不在 Phase 5 未关闭时开始新 Provider 产品代码。

---

## 16. 优先级与依赖

| 优先级 | 工作 | 复杂度 | 前置依赖 | Owner 是否需要中途决策 |
| --- | --- | --- | --- | --- |
| P0 | 完成 PR #59 和 Phase 5 Closure | M | 无 | 只在最终合并 |
| P1 | Multi-Provider Reviewed Catalog | M | Phase 5 Closure | 否 |
| P1 | Fingerprint Chromium V1 | XL | Multi-Provider | 仅重大方向变化 |
| P1 | Evidence V2 与 Restart/Seed Matrix | L | Fingerprint V1 可运行 | 否 |
| P2 | Identity Template 与 Root Seed | M | V1 Contract | 否 |
| P2 | Network Identity Advisor | M | Proxy Observation | 否 |
| P2 | Environment Readiness 与 UI | M | Evidence/Network 状态稳定 | 否 |
| P3 | Cookie / Extension | L | 核心闭环完成 | 后续阶段授权 |
| P3 | Automation / MCP / Release | XL | 后续阶段 | 产品与许可证 Gate |

---

## 17. 自动内部 Gate

这些 Gate 由 Agent 根据测试自动判断，不需要 Owner 每次批准。

### Internal Gate A — Phase 5 Baseline

PR #59 的真实浏览器和产品回归全部完成。

### Internal Gate B — Provider Container

Multi-Provider 上线，现有 Stock Provider 行为和 Evidence 完全不变。

### Internal Gate C — Fingerprint V1

每项对外 capability 在 exact binary、多个 Context 和 Restart 中通过；失败项被隔离。

### Internal Gate D — Product Usability

Template、Network Advisor、Readiness 和错误说明足以让普通用户正确创建环境。

### Internal Gate E — Release Candidate

Build、Package、Runtime、Security、Migration、Evidence 和 Manual Smoke Test 完成。

通过一个 Internal Gate 后，Agent 自动进入下一 Stage，并在文档中记录，不等待 Owner 回复。

---

## 18. 推荐的连续执行路径

```text
当前 PR #59 收尾
        ↓
Ready for Review
        ↓
Owner 合并决定
        ↓
Phase 5 Closure
        ↓
一个长期 Draft PR：fingerprint-platform-v1
        ↓
Multi-Provider Foundation
        ↓
Fingerprint Chromium V1
        ↓
Identity Template + Root Seed
        ↓
Network Identity Advisor
        ↓
Evidence V2 + Matrix
        ↓
Environment Readiness + UI
        ↓
完整构建、真实浏览器、安全和产品验证
        ↓
Ready for Review
        ↓
Owner 合并 / License / 发布决定
```

开发中不应出现：

```text
完成一个文件 → 问是否继续
完成一个模块 → 问是否进入下一步
测试失败 → 问是否修复
有两个内部实现方案 → 要求 Owner 选技术细节
```

只有触发本文 Owner Gate 或强制升级条件，才暂停相关不可逆动作。

---

## 19. 当前唯一下一步

当前只执行 Stage 0：自主完成 PR #59 的真实运行与手工验证，直到 Ready for Review。

Phase 5 正式关闭后，第一项代码任务是：

> 将 `internal/kernelrelease` 和 `internal/fingerprint` 从单一 reviewed Provider 重构为 Multi-Provider 容器，同时保持现有 Stock Chromium、Profile、Evidence、Portable Definition 和 Lifecycle 行为完全不变。

完成该基础后，Agent 可以按照本文 Stage 2–6 连续开发，无需在普通技术节点等待 Owner 决策。