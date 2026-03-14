# 深度对比评估：两份设计文档

对比对象：
- 文档 A：`example.com-Tst-operator-design.md`（称为 A）
- 文档 B：`K8s 1.35 原生 Operator 方案设计：example.com-Tst-v1beta1.md`（称为 B）

下面按关键维度做精简、聚焦的深度分析与评估，并给出最终建议与可落地的下一步。

---

## 一、核心一致点（两者均满足的要点）
- 使用原生 client-go（兼容 k8s 1.35）为基础，不使用 controller-runtime/Operator SDK/codegen。 
- 采用 Informer + workqueue （RateLimitingInterface）事件驱动架构。
- 建议使用 dynamic client / unstructured（或手写类型作为可选），并强调对 resourceVersion/generation 的检查。
- Status 必须用 Patch（subresource=status，MergePatch 或 JSONPatch），避免全量 Update，遇 409 做重试。
- 推荐使用 Server-Side Apply（SSA）管理下游资源，但 Status 更新仍用 Patch。
- 建议记录 specHash（sha256）与 observedGeneration 用于变更检测与幂等。
- 建议导出指标、提供 RBAC 最小权限、做单元/集成/Chaos 测试。

## 二、关键差异（对设计与运行模式影响最大）

1) 多副本并发模型：
- A：推荐“多副本并行处理不同资源”的模型。采用 local mutex + per-resource Lease（coordination.k8s.io/v1）作为跨 Pod 的分布式锁，允许多个 Pod 并发处理不同对象。Leader election 为“可选”，主要用于全局任务。优点：高并发，低延迟。缺点：实现复杂、Lease 创建/删除会带来 apiserver 负载与竞争处理复杂性。
- B：主张“Leader-only Reconcile”（必须 leader election），只有 leader 实际执行 reconcile，其他副本处于待命。优点：实现简单、几乎无分布式锁复杂性，冲突面小。缺点：牺牲并发能力（leader 成为瓶颈），在大量对象或高吞吐场景下性能受限。

2) 锁的粒度与策略：
- A：推荐对象级 Lease（per-resource），以及本地 sync.Map mutex 做双层保证。
- B：以 leader election 为主，辅以进程内锁（sync.Map）作为防护，但不主张 per-resource Lease（除特殊原子操作）。

3) 运行复杂度与 APIServer 负载：
- A：更复杂，更多 APIServer write（Lease 操作、短期 renew），需要在规模/频率上谨慎调优。
- B：更简单，较少额外 API 请求，易于运维。

## 三、评估：何时选 A（per-resource Lease，多活） vs 何时选 B（leader-only）

选择 A（多活）当：
- 需要高并发处理大量 Tst 资源（例如上千对象，频繁更新）。
- 低延迟响应是关键（不能集中到单 leader）。
- 团队可以承担实现复杂性与更严格的测试。

选择 B（leader-only）当：
- 资源规模中小或变更频率低，单实例能满足吞吐。
- 希望尽量降低实现与运维复杂度、减少对 apiserver 的额外负载。
- 优先稳定性与可维护性而非极限性能。

## 四、折衷与推荐（针对你的原始需求）
你要求：多副本支持 + 锁与队列机制防并发。推荐采用“混合方案（默认 A 的多活模型，但带可切换到 B 的 leader-only 模式）”：
- 默认允许多副本并行工作，使用 workqueue + 本地锁；跨实例使用 per-resource Lease 作为互斥（按需开启）。
- 提供启动参数开关：--multi-active=true/false。若 false，则启用 leader election 并仅 leader 执行 reconcile（B 模式）。
- 初始实现优先以 B 模式快速、安全落地；在验证稳定后再实现 A 模式并发优化（渐进式交付）。

理由：降低首版风险（leader-only 易实现、易测试），并留出后续扩展路径以满足高并发需求。

## 五、实现要点与注意事项（技术细节提醒）
- client-go 版本必须与集群 1.35 对齐（v0.35.x）。
- per-resource Lease 命名与回收策略：建议统一前缀（如 `tst-lock-<ns>-<name>`），重用 Lease（不要频繁创建/删除），ttl 15s，renew 每 5-8s。对高频短任务，可考虑将锁状态作为 CRD annotation/field（降低 apiserver 写入）但需权衡一致性。
- Patch status：始终 GET 最新对象，构建最小 MergePatch，仅 patch status 子资源。冲突 409 时重建 patch 并做指数退避（最多 5 次）。
- 测试场景：必须覆盖 leader failover、Lease 抢占、并发更新冲突、高频 create/update/delete。
- 指标：统计 Lease 获取失败率、lease-churn、queue 延迟、重试次数。

## 六、下一步建议（可执行的迭代计划）
1. 确认首版运行模式：建议先实现 leader-only（B），快速上线验证业务逻辑与 status patch 流程。
2. 在稳定基础上迭代实现多活（A）：实现 per-resource Lease、参数化开关、压力测试并调优 Lease 策略。 
3. 编写集成测试（KinD）：覆盖 leader 切换、并发修改与冲突回退。 
4. 最终交付：文档化 field ownership、RBAC、部署与调优建议。

---

我已将本比较评估导出为文件：
`c:\Users\Administrator\Desktop\operators - 副本 - 副本 - 副本\tst_operator_deep_comparison.md`

是否现在需要我：
- 基于选定方案生成 Reconcile 伪代码与主要模块清单？
- 或直接生成首版 leader-only 的完整骨架代码与 manifest？

简短回复要做的下一步即可。
