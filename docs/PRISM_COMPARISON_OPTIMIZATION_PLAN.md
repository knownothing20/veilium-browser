# Veilium Browser 对照 Prism Browser 的优化开发方案

状态：**提案 / 参考文档，不构成当前阶段的实现授权**  
创建日期：2026-08-19  
对照基线：

- Prism Browser Community：`main`，提交 `004758a17fe3a08ad33c28f6ece8537967e239d0`；
- Veilium Browser：PR #59，分支 `agent/handoff-m5-3`，分析时提交 `8d16ddc6decd2b072cc876c3d1365ad54f9500bf`。

## 1. 文档目的与边界

本文基于两个项目的实际代码重新对照，回答三个问题：

1. Prism 已经实现、Veilium 当前真正缺少的核心能力是什么；
2. Veilium 已有的哪些架构应保留，哪些地方需要补强或修正；
3. Phase 5 完成后，如何以尽量少的返工把 Veilium 增强成可验证、可维护的指纹浏览器产品。

本文不修改 `PRODUCT.md`、`ROADMAP.md`、`PHASE_05.md` 或 `STATUS.md` 的现有授权。当前唯一任务仍是完成 PR #59 的真实浏览器、中文界面和完整功能回归验证。Phase 5 明确禁止在当前 PR 中新增第二个 reviewed Provider 或新的指纹能力声明，因此本文只能作为后续阶段的规划输入，不能被解释为可以立即开发新内核。

参考 Prism 时遵守 clean-room 原则：可以研究功能边界、数据流、测试方法和用户体验，不直接复制其应用代码或未经审查地移植内核补丁。任何第三方代码、补丁或上游组件都必须单独审查许可证、来源、修改记录和分发义务。

## 2. 核心结论

两个项目的优势并不重叠：

- **Prism 的优势**是已经形成完整的指纹浏览器闭环：定制 Chromium、指纹启动参数、硬件模板、根种子、代理位置联动、真实指纹矩阵，以及 Cookie、扩展、备份、更新和自动化等产品功能。
- **Veilium 的优势**是更严格的运行平台：Provider Contract、完整包校验、OS Credential Vault、代理桥与 Xray/sing-box、Browser Supervisor、真实 Evidence、Compatibility、Operation Journal、Snapshot/Restore 和可恢复 Lifecycle。

Veilium 不应该改成 Electron，也不应该照搬 Prism 的整体代码。正确目标是：

> 在 Veilium 现有 Provider、Evidence、Security、Proxy、Supervisor 和 Lifecycle 架构上，补齐真正可执行的 Fingerprint Chromium、身份模板、网络身份建议和更完整的指纹验证矩阵。

最终产品应同时具备：

```text
Prism 已验证的指纹内核产品能力
                  +
Veilium 更严格的 Provider / Evidence / Security / Lifecycle 架构
```

## 3. 两个项目的源码级对照

| 维度 | Prism Browser Community | Veilium Browser | 结论 |
| --- | --- | --- | --- |
| 桌面架构 | Electron + React + TypeScript | Wails + Go + React + TypeScript | 保留 Veilium，不迁移 Electron |
| 浏览器内核 | Fingerprint Chromium 144，包含大量 Chromium/Blink patch | 一个 exact reviewed stock Chromium Snapshot 152 | Veilium 最大缺口是“真正的指纹内核” |
| Provider 模型 | 主要按内核版本和 `fingerprintKernel` 判断 | Provider Contract v2，区分 reviewed/custom/legacy/disabled/invalid | Veilium 的契约模型更适合长期维护 |
| 指纹配置执行 | 自定义 Chromium 参数与 patch 实际接收配置 | 配置、Capability 和启动参数框架已存在，但 stock Provider 不支持高级能力 | 需要新增独立 Fingerprint Provider，不能提升 stock Provider 的声明 |
| 身份模板 | 有 host-native、seeded GPU、固定硬件模板 | 主要让用户手工组合 CPU、屏幕、GPU、语言和时区 | Veilium 应增加成套、可验证的 Identity Template |
| 根种子 | Seed 参与 Canvas、WebGL、Audio、DOMRect、GPU bucket 等 | 已有 Profile seed，但尚未形成版本化的整套派生规则 | 应把 Root Seed 变成稳定身份的唯一入口 |
| 网络身份 | 代理检测结果可派生语言、时区和地理位置 | 代理诊断、Network Evidence 更强，但没有身份建议层 | 增加“观察—建议—用户确认”，不允许静默修改 |
| 密钥保护 | Electron `safeStorage`，不可用时存在 base64 fallback | OS keyring，无明文 fallback | 保留 Veilium 的安全边界 |
| 运行验证 | 多个 audit 工具和 fingerprint matrix | Provider/Binary/Context/Network Evidence 与 Compatibility 更结构化 | 把 Prism 的矩阵思想接到 Veilium Evidence 上 |
| Lifecycle | Profile 备份、复制、回收站、工作区迁移 | Journal、锁、取消、Snapshot、Restore、Trash、Rollback、Reconciliation | Veilium 已明显更强，不重写 |
| 产品完整度 | Cookie、扩展、更新、Automation、Scheduler、MCP 已有产品入口 | PR #59 正在完成中文环境管理、可移植定义和批量管理 | 先完成 PR #59，再补核心内核，不继续横向堆功能 |

## 4. 重新分析后发现的关键代码问题

### 4.1 Reviewed Kernel Catalog 被写成“永远只有一个 Provider”

当前 `internal/kernelrelease/catalog.go` 存在两个硬约束：

- `ProviderID` 被写死为 `official-chromium-snapshot-win64`；
- manifest 校验要求 `len(manifest.Releases) == 1`。

`internal/fingerprint/official.go` 同样要求 release 数量必须恰好为一个，否则整个 official Provider 会进入 invalid 状态。

这套实现对 Phase 4 的单一 reviewed Snapshot 是正确且安全的，但会阻止未来同时存在：

```text
Official Stock Chromium
Veilium Fingerprint Chromium
Custom Chromium
Legacy Chromium
```

后续第一项代码工作应是把“单一官方 release”重构为“按 Provider ID、revision、version、platform、arch 索引的 reviewed release catalog”，同时保持现有 Provider ID 和数据完全兼容。此步骤只改变容器能力，不增加任何新指纹声明。

### 4.2 FingerprintConfig 到 Chromium 的映射不完整

Veilium 已经定义 `deviceMemoryGb`，Validator 和 Capability 也会检查它，但 `internal/fingerprint/args.go` 只验证 capability，没有生成对应 Chromium 参数。这意味着即使未来 Provider 宣称支持，配置也不会真正传给浏览器。

屏幕也存在类似问题。当前 `screenWidth` 和 `screenHeight` 主要参与 `--window-size`；窗口大小不等于网页读取到的 `screen.width`、`screen.height` 和 `availWidth/availHeight`。未来必须新增独立的 Screen Capability 和内核参数，不能用窗口设置冒充屏幕指纹。

建议把能力拆清楚：

- `window-size`：管理浏览器窗口；
- `screen-identity`：控制网页可见的屏幕信息；
- `device-memory`：控制 `navigator.deviceMemory`；
- `hardware-concurrency`：控制 `navigator.hardwareConcurrency`；
- `language` 与 `accept-language`：分别验证，不只设置 `--lang`；
- `timezone`：必须由 exact Provider 和真实 Evidence 证明。

### 4.3 身份配置仍是“手工拼字段”，缺少一致性模板

PR #59 的 `ProfileEditor.tsx` 已经把创建环境改造成五段式中文流程，这是明显进步；但身份部分仍允许用户独立组合 Platform、Brand、Language、Timezone、Screen、CPU、WebRTC 和 GPU。

当前 Validator 主要验证字段是否合法，却无法判断组合是否像一台真实、稳定的设备。例如：

- Windows 平台配 Apple GPU；
- 低端设备配置异常高的 CPU、内存和分辨率；
- 代理位于美国，但语言和时区仍为中国；
- GPU、WebGL、WebGPU、Canvas、字体和 Audio 来自不同身份模型。

Prism 的 `hardware-profiles.ts` 已经证明“模板优先、手工高级设置为辅”更适合普通用户。Veilium 应独立设计自己的内置 Identity Template Catalog，而不是让用户自由拼出矛盾身份。

### 4.4 前端默认策略和 Provider 展示存在漂移

当前 `frontend/src/lib/model.ts` 默认选择 `custom-chromium`、Chromium 148、`America/Los_Angeles`。这会让第一次创建环境的用户默认进入一个无 reviewed 指纹声明的 Provider，并得到与真实网络无关的固定时区。

`ProfileTable.tsx` 的 `kernelProviderLabel()` 只识别 `patched-chromium` 和 `custom-chromium`，其他 Provider 都显示为“本机 Chromium”。因此现有 `official-chromium-snapshot-win64` 可能被错误展示。

优化方向：

1. 默认选择当前平台上已安装、已验证且信任级别最高的 Kernel；
2. 没有 reviewed Kernel 时明确显示“自定义 / 未验证”，不要伪装成推荐状态；
3. Provider 名称和信任状态由后端 descriptor 提供，不在前端硬编码 ID；
4. 默认时区来自本机或用户明确选择，不硬编码美国时区；
5. Profile 创建页先选择“设备身份模板”和“网络方式”，高级字段再渐进展开。

### 4.5 Proxy 很强，但还没有形成 Network Identity

`internal/proxydiagnostics` 已经能够验证 Route、Bridge、Connectivity、Exit IP、Latency，并说明 DNS 和 WebRTC 检测边界；`internal/networkevidence` 还包含真实浏览器侧的受控网络验证。这些能力比 Prism 的普通 Proxy Tester 更严格。

缺口是代理结果没有转化为可审查的身份建议。建议增加独立的 `NetworkIdentityObservation`：

```text
代理出口观测
  ├─ IP / Country / Region / City
  ├─ Timezone candidates
  ├─ Locale / Accept-Language candidates
  ├─ Geo confidence / conflict
  └─ ObservedAt / Source / Expiry

          ↓

Identity Recommendation
          ↓
用户点击“应用建议”后才修改 Profile
```

禁止在代理变化后自动修改 Profile。代理观测失效、冲突或来源不足时只提示，不阻止用户查看现有配置；是否阻止启动由明确的 Profile Policy 决定。

### 4.6 Evidence 基础很好，但覆盖面还不够

Veilium Evidence 已经支持 Top-level、Iframe、Worker 三个上下文，并采集 UA、Platform、Languages、Timezone、CPU、Screen、Window、WebRTC 和四种 Surface Digest。这是值得保留的核心优势。

当前不足：

- Surface 仅允许 `canvas`、`webgl`、`audio`、`clientRects`；
- 没有 `deviceMemory`；
- 没有 GPU vendor/renderer、WebGPU adapter；
- 没有 Fonts 和 Speech voices；
- 没有 Profile 关闭后再次启动的 Restart Stability；
- 没有同模板不同 seed 的 Separation Matrix；
- 没有把 Profile 配置、Provider revision、Kernel package identity、Route identity 和 Evidence freshness 汇总成一个用户可理解的 Readiness 结果。

这些不应新建一套测试系统，而应升级现有 Evidence schema、Evaluation 和 Compatibility。

### 4.7 前端 `profileHealth()` 过于简单

目前前端健康判断主要检查名称、可执行文件、WebRTC 与代理组合、代理 URL 中是否含凭据。它没有覆盖：

- Kernel 完整性；
- Provider trust；
- Capability 是否支持当前配置；
- Evidence 是否通过、过期或与当前配置失配；
- Network Evidence；
- Lifecycle lock / recovery-required；
- Credential 或 Adapter 是否缺失；
- Profile 是否仍为 draft。

后续应由后端返回版本化的 `EnvironmentReadiness`，前端只负责展示，不在多个组件里重复推断。

## 5. 目标架构

```text
Browser Profile
      │
      ├── Identity Template ── Root Seed ── Versioned derivation
      │
      ├── Network Route ── Proxy / Adapter / Credential Vault
      │                         │
      │                         └── Network Identity Observation
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

这条链的关键规则是：

> UI 配置、启动参数、Kernel 文件和真实网页结果必须同时一致，才能把能力标记为 verified。

## 6. 实施原则

1. **保留 Go/Wails。** 不因 Prism 使用 Electron 而更换技术栈。
2. **Stock 与 Fingerprint Provider 并存。** 普通 Chromium 保留为稳定、可信的基线，不偷偷升级为指纹 Provider。
3. **Exact pin，不解析 moving latest。** Provider release 必须锁定版本、来源、archive、executable、完整 package tree 和 capability revision。
4. **Evidence before claim。** 没有 real-browser test 的能力只能是 unsupported 或 unverified。
5. **Root Seed 稳定。** 同一 Profile 重启不变化；不同 Profile 默认分离；Clone 默认生成新 seed。
6. **模板优先。** 普通用户选择一套合理设备，高级用户才展开字段。
7. **网络只提供建议。** 任何语言、时区、地理位置修改必须由用户确认。
8. **最小 V1。** 第一版不追求一次覆盖 Fonts、Speech、WebGPU 和所有平台。
9. **向后兼容。** 不静默修改现有 Provider、Profile、Portable Definition 或 Evidence。
10. **Clean-room。** 记录参考来源、行为需求、独立设计和验证结果。

## 7. 分阶段优化开发方案

### Stage 0 — 完成并关闭当前 Phase 5

优先级：P0  
目标：先把已经开发的产品变成可信基线，避免在未验证的大 PR 上继续叠加内核工作。

必须完成：

- PR #59 的真实 Chromium start/readiness/stop/process-tree cleanup；
- 1366×768 和 1920×1080 中文主要流程检查；
- Kernel、Credential、Proxy、Recovery、Portability、Template、Batch、Storage 和 Evidence smoke test；
- 重启后数据持久化；
- 证明管理界面语言不会修改 Profile 的 language、timezone、platform 或 fingerprint；
- 修复发现的问题并同步 `STATUS.md`；
- 完成 Phase 5 的正式关闭流程。

退出条件：PR #59 合并并且 Phase 5 closure 通过。在此之前不得实现下面的 Stage 1–7。

### Stage 1 — Multi-Provider Reviewed Kernel Foundation

优先级：P1  
复杂度：M  
目标：消除单一 Provider 假设，但不改变现有用户行为。

主要改动：

- `internal/kernelrelease/catalog.go`
  - 移除全局唯一 `ProviderID`；
  - 支持多个 Provider 和多个 exact release；
  - 使用复合键：Provider ID + revision + version + OS + arch；
  - 每个 Provider 可以定义自己的来源和 archive layout validator。
- `internal/kernelrelease/releases.json`
  - schema 升级并提供严格迁移；
  - 现有 `official-chromium-snapshot-win64` 保持原 ID 和原 digest。
- `internal/fingerprint/official.go` / `catalog.go`
  - Provider definition 按 release/provider 查询，不再要求 release 总数为一；
  - stock Provider 的高级 fingerprint capability 继续为 unsupported。
- Kernel installer、registry、portable dependency matching 和 frontend pin 类型同步支持多 Provider。

验收：

- 旧 Kernel 和 Profile 无需迁移即可工作；
- 现有 exact package identity 和 Evidence 仍通过；
- 添加第二个测试 Provider fixture 不会污染第一个 Provider；
- 不产生任何新的 verified fingerprint claim。

### Stage 2 — Veilium Fingerprint Chromium V1

优先级：P1  
复杂度：XL  
目标：新增独立的 Windows amd64 Fingerprint Provider，形成真正的指纹浏览器执行层。

建议 V1 能力：

- Platform / browser brand；
- Language / Accept-Language；
- Timezone；
- Screen / available screen / device scale；
- Hardware concurrency；
- Device memory；
- Root-seeded Canvas；
- Root-seeded Audio；
- Root-seeded ClientRects / DOMRect；
- WebGL/GPU identity；
- WebRTC proxy-only policy。

V1 暂缓：

- 完整字体目录模拟；
- Speech voice catalog；
- WebGPU template identity；
- macOS reviewed Provider；
- 多个 Chromium 主版本并行维护。

实现要求：

- 先选择一个 exact Chromium 基线，不跟随最新版本自动升级；
- 内核参数由版本化 Contract 定义，不复用含义模糊的自由字符串；
- 补齐 `device-memory` 和真正的 `screen-identity` 参数映射；
- 每个 capability 对应代码、Provider declaration、测试和 Evidence；
- 构建产物执行 archive、executable、package tree 和 provenance 校验；
- stock Chromium 和 Fingerprint Chromium 在 UI 中清晰区分；
- 不允许用户把 custom Kernel 手工标记成 reviewed。

退出条件：每个对外宣称的 capability 都在 exact package 上通过真实浏览器验证，否则保持 partial/unverified/unsupported。

### Stage 3 — Identity Template 与 Root Seed

优先级：P2  
复杂度：M  
目标：让普通用户选择合理设备，而不是手工拼装矛盾字段。

第一版模板控制在少量：

- Windows · 本机硬件；
- Windows · 标准办公设备；
- Windows · 主流桌面设备；
- Windows · 高性能桌面设备；
- Advanced · 自定义配置。

Root Seed 使用版本化派生规则：

```text
Root Seed
  ├─ canvas sub-seed
  ├─ audio sub-seed
  ├─ client-rects sub-seed
  ├─ webgl/gpu bucket
  ├─ font policy seed（后续）
  └─ speech/webgpu seed（后续）
```

规则：

- 同一个 Profile 和同一个 derivation version 始终得到相同身份；
- Clone 默认生成新 Root Seed；
- “保留身份”只在高级、明确确认的迁移流程中出现；
- 模板升级不能静默改变既有 Profile；
- 模板只生成配置，不赋予 Provider trust 或 Evidence。

建议新增版本化内置 catalog，而不是先做复杂的在线模板市场。

### Stage 4 — Proxy 到 Network Identity Advisor

优先级：P2  
复杂度：M  
目标：把 Veilium 已有的网络验证转化为身份一致性建议。

功能：

- 在 Proxy Diagnostics 后生成有时效的网络位置 Observation；
- 给出 timezone、language、Accept-Language 和可选 geolocation 建议；
- 显示来源、置信度、冲突和更新时间；
- 当前 Profile 与网络观测冲突时显示可理解的警告；
- 用户点击“应用建议”后才更新 Profile；
- 代理出口变化使旧 Observation 失效，但不自动改写 Profile；
- Network Evidence 仍负责真实浏览器外部网络行为，Advisor 不替代 Evidence。

不做：代理池、轮换、自动选择线路、定时切换或无人值守身份修改。

### Stage 5 — Evidence V2 与 Fingerprint Matrix

优先级：P1  
复杂度：L  
目标：用 Veilium 现有 Evidence 系统证明新内核稳定、一致且可区分。

新增观测：

- `deviceMemory`；
- Accept-Language；
- GPU vendor / renderer；
- WebGL parameters；
- 可用时的 Fonts、Speech、WebGPU；
- Provider capability revision 和 identity derivation version。

新增矩阵：

1. **Context Consistency**：Top-level / Iframe / Worker；
2. **Restart Stability**：同一 Profile 关闭并重新启动，身份不变；
3. **Seed Separation**：不同 seed 的目标 surface 应分离；
4. **Template Coherence**：CPU、Memory、GPU、Screen、Platform、Language、Timezone 不矛盾；
5. **Network Coherence**：Route、Exit IP、DNS、WebRTC 与 Profile Policy 一致；
6. **Tamper Downgrade**：修改 Kernel package 任意受保护文件后立即失去 reviewed/compatible 状态；
7. **Evidence Freshness**：Kernel、Provider、Profile fingerprint 或 Route 变化后旧 Evidence 不再适用。

第三方检测站只能作为补充观测，不能代替项目自己的可重复测试和 exact binary Evidence。

### Stage 6 — Environment Readiness 与产品修正

优先级：P2  
复杂度：M  
目标：把复杂技术状态汇总成普通用户能理解的环境状态。

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
- Kernel integrity；
- Provider trust；
- Capability compatibility；
- Credential / Adapter availability；
- Proxy Diagnostics；
- Identity Evidence；
- Network Evidence；
- Evidence freshness。

同时修复：

- Provider label 不再前端硬编码；
- `official-chromium-snapshot-win64` 不再显示成“本机 Chromium”；
- 默认 Provider 根据当前平台、安装状态、完整性和信任等级选择；
- 默认 timezone 不再固定为 `America/Los_Angeles`；
- Profile Editor 默认显示设备模板，原始字段移入高级设置；
- 环境列表优先展示“可打开 / 需处理 / 已阻止”的原因，Evidence 细节继续渐进展开。

### Stage 7 — 核心闭环完成后的产品补全

优先级：P3  
目标：在 Fingerprint Kernel 和 Evidence 闭环稳定后，再决定是否补齐 Prism 已有的产品功能。

候选内容：

- Cookie 查看、导入和导出；
- Extension 管理；
- 更简单的 Profile backup / migration 入口；
- Controlled Automation；
- Scheduler / MCP；
- Release signing、Updater、SBOM 和可复现构建。

这些功能必须遵守现有 Vault、Lifecycle、Authorization、Audit 和 Recovery 边界。不要为了追赶 Prism 的功能数量而提前实现。

## 8. 建议的代码改动地图

| 模块 | 主要职责 |
| --- | --- |
| `internal/kernelrelease` | 多 Provider exact release catalog、来源和 package identity |
| `internal/kernelinstaller` | 按 Provider 安装、验证、原子激活和回滚 |
| `internal/kernel` | 多 Provider Kernel registry 和 in-use protection |
| `internal/fingerprint/catalog.go` | Capability Contract、Provider revision 和支持边界 |
| `internal/fingerprint/args.go` | 完整、版本化的配置到 Kernel 参数映射 |
| `internal/fingerprint/validator.go` | 单字段验证、模板一致性和 fail-closed 规则 |
| `internal/domain/types.go` | Root Seed、derivation version、template reference 等稳定合同 |
| `internal/launch/planner.go` | Provider-aware launch plan，不把描述性字段当作已应用能力 |
| `internal/evidence` | Evidence V2、重启稳定、surface/context 矩阵 |
| `internal/networkevidence` | Route 绑定的真实网络 Evidence 和 Compatibility |
| `internal/proxydiagnostics` | 代理观测、位置元数据和 Advisor 输入 |
| `frontend/src/lib/model.ts` | 删除过度简化的前端健康推断，消费后端 Readiness |
| `frontend/src/components/ProfileEditor.tsx` | Template-first 创建流程和网络身份建议 |
| `frontend/src/components/ProfileTable.tsx` | Provider 正确展示、Readiness 和阻止原因 |
| `frontend/src/components/OfficialKernelCard.tsx` | 多 reviewed Provider 安装与限制说明 |
| `docs` | Provider、Kernel、Identity、Evidence、Migration、Security 和 Compatibility 文档 |

## 9. Provider 能力目标矩阵

| 能力 | Official Stock Chromium | Veilium Fingerprint V1 | Custom Chromium |
| --- | --- | --- | --- |
| Managed launch | reviewed | reviewed | custom/unverified |
| Exact package integrity | reviewed | reviewed | 仅本地校验 |
| Platform override | unsupported | 真实验证后 verified | unsupported/unverified |
| Brand override | unsupported | 真实验证后 verified | unsupported/unverified |
| Language / Accept-Language | 仅记录实际行为 | 真实验证后 verified | unverified |
| Timezone override | unsupported | 真实验证后 verified | unsupported/unverified |
| Screen identity | unsupported | 真实验证后 verified | unsupported/unverified |
| Hardware concurrency | unsupported | 真实验证后 verified | unsupported/unverified |
| Device memory | unsupported | 真实验证后 verified | unsupported/unverified |
| Canvas / Audio / ClientRects seed | unsupported | 真实验证后 verified | unsupported/unverified |
| WebGL / GPU identity | unsupported | 真实验证后 verified | unsupported/unverified |
| WebRTC proxy-only | 不作高级指纹声明 | Network Evidence 通过后 verified/partial | unverified |
| Fonts / Speech / WebGPU | unsupported | V2 或验证完成后再声明 | unsupported/unverified |

状态必须来自 Provider Contract 和 Evidence，不能由 UI 是否显示开关决定。

## 10. 完整验证矩阵

每一个 reviewed Fingerprint Provider 至少完成：

### 静态和合同验证

- manifest strict decoding、unknown field 和 trailing data；
- Provider/revision/version/platform/arch 唯一性；
- archive、executable 和 package tree digest；
- capability schema、参数生成和 unsupported fail-closed；
- 旧 Profile、Portable Definition 和 Kernel Record 的兼容读取。

### 真实浏览器验证

- exact binary 启动、readiness、关闭和进程树清理；
- UA / Client Hints / Platform / Language / Accept-Language / Timezone；
- Screen / Window / DPR；
- CPU / Device Memory；
- Canvas / Audio / ClientRects / WebGL；
- Top-level / Iframe / Worker 一致；
- 同 seed 重启稳定；
- 不同 seed 按预期分离；
- Provider 不支持的配置必须阻止保存或启动。

### 网络验证

- Direct baseline；
- HTTP / HTTPS / SOCKS5；
- OS Vault credential bridge；
- Xray / sing-box 受支持子集；
- Exit IP、DNS route、WebRTC；
- Network Observation 与 Profile identity 冲突；
- 代理出口变化和旧 Evidence 失效。

### 失败与安全验证

- archive 或 package tamper；
- 缺失依赖；
- 磁盘满、权限、rename、flush、取消和中断；
- 启动失败、CDP readiness 失败、进程残留；
- 日志、Bootstrap、报告和导出中没有 secret；
- 不使用 `--no-sandbox` 作为通过测试的方法；
- rollback 后原健康 Kernel 和 Profile 仍可恢复。

### 产品验证

- 1366×768 和 1920×1080；
- 第一次安装、第一次创建、打开、关闭和修复环境；
- 普通模板和高级配置；
- Provider 限制和 Evidence 失败可理解；
- 管理界面语言不会改变浏览器身份。

## 11. 数据迁移与兼容策略

- 保留现有 Provider ID，不静默重命名；
- 现有 stock Provider 永远不会因新 Fingerprint Provider 出现而自动升级；
- 新 Profile 字段必须 optional 或具有明确 schema migration；
- 旧 Profile 没有 template reference 时按 `legacy-custom` 解释；
- Template 更新不改变既有 Profile；
- Kernel 或 Provider revision 变化会使旧 Evidence stale，而不是自动继承；
- Portable Definition 不携带本地 Kernel ID、路径、secret、browser data、Provider trust 或 Evidence；
- Preserve Identity 模式仍创建新的本地 Profile ID 和目录；
- 降级到旧版本时不能乐观删除未知字段或重写数据。

## 12. 安全、隐私和许可证

必须保留的 Veilium 优势：

- secret 只进入 OS Credential Vault；
- 不提供 plaintext/base64 fallback；
- Profile、日志、启动参数、Bootstrap、Evidence 和导出不包含代理密码；
- 本地控制面只绑定 loopback，并执行认证和权限边界；
- Browser、Proxy Bridge、Xray 和 sing-box 由 Supervisor 持有和清理；
- Kernel 和 Adapter 通过 exact package identity 管理；
- 不静默下载、升级或替换敏感运行时。

后续内核工作必须额外记录：

- 上游仓库、tag、commit 和 license；
- 每个 patch 的来源、目的、SHA-256、适用顺序和独立修改；
- Chromium 和第三方 notices；
- 构建参数、工具链和产物清单；
- 是否允许二进制再分发；
- clean-room 设计说明。

Veilium 仓库目前没有明确的 GitHub license metadata。准备公开发布或接受外部贡献前，项目所有者必须作出许可证决定并加入相应文件，不能只在 README 中写“open-source”。

## 13. 明确不做

- 不承诺绕过风控、CAPTCHA 或平台规则；
- 不承诺账号存活率或检测规避率；
- 不做每次启动随机身份；
- 不做账号农场、代理轮换或批量无人值守启动；
- 不在第一版同时支持 Windows、Linux、macOS 和多个架构；
- 不把第三方检测站评分当作唯一验收；
- 不把 custom Chromium 自动提升为 reviewed；
- 不因追求功能数量而弱化 sandbox、secret、Evidence 或 rollback；
- 不直接复制 Prism 应用代码或未经许可证、来源和安全审查的补丁。

## 14. 优先级与依赖

| 优先级 | 工作 | 复杂度 | 前置依赖 |
| --- | --- | --- | --- |
| P0 | 完成 PR #59 和 Phase 5 closure | M | 无 |
| P1 | Multi-Provider reviewed catalog | M | Phase 5 closure |
| P1 | Fingerprint Chromium V1 | XL | Multi-Provider foundation |
| P1 | Evidence V2 与 restart/seed matrix | L | Fingerprint V1 可运行 |
| P2 | Identity Template 与 Root Seed | M | V1 capability contract |
| P2 | Network Identity Advisor | M | Proxy Observation 结构 |
| P2 | Environment Readiness 与 UI 修正 | M | Evidence/Network 状态稳定 |
| P3 | Cookie / Extension 产品面 | L | 核心指纹闭环完成 |
| P3 | Automation / MCP / Release hardening | XL | 后续阶段单独授权 |

## 15. 决策门

### Gate A — Phase 5 基线可信

PR #59 已合并，真实浏览器和中文产品完整回归通过。

### Gate B — Provider 容器可扩展

多 Provider catalog 上线，但现有 stock Provider 行为和 Evidence 完全不变。

### Gate C — Fingerprint V1 可声明

每一项 capability 都在 exact binary、多个上下文和重启测试中通过；失败项不对外宣称。

### Gate D — 普通用户可正确使用

模板、网络建议、Readiness 和错误说明让用户无需理解底层参数即可创建一致环境。

### Gate E — 可公开发布

许可证、构建来源、notices、签名、更新、SBOM、安全审查和分发边界全部明确。

## 16. 推荐的下一步

当前只执行 Stage 0：完成 PR #59 的真实运行和手工验证，不在该 PR 中加入 Fingerprint Kernel 或新 Provider 代码。

Phase 5 正式关闭后，后续单一开发路径的第一个任务应是：

> **把 `internal/kernelrelease` 和 `internal/fingerprint` 从单一 reviewed Provider 改造成多 Provider 容器，同时保持所有现有行为不变。**

只有这个基础任务通过审查和回归后，才开始 Veilium Fingerprint Chromium V1。这样能避免把高风险内核工作、产品 UI、Lifecycle 和数据迁移混在同一个大改动中。