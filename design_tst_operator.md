# TST Operator 设计方案（无框架、无代码生成）

本文档给出在 Kubernetes v1.35 上，用原生 Go（不使用 controller-runtime / codegen / operator-sdk 等框架或自动生成工具）实现一个对 CRD group: `example.com`, version: `v1beta1`, kind: `Tst`（命名为 `tst`，CRD scope: Namespaced）的 Operator 的最优设计方案与实现要点。

目标要求摘要：
- 支持多副本运行（Deployment n>1）
- 拥有锁与队列机制，防止并发对同一资源的冲突
- 使用最新 API（兼容 Kubernetes v1.35、客户端库与 CRD API）并避免已废弃的接口
- CRD Scope: Namespace
- 更新 Status 使用 Patch（而非完整 Update）
- 在轮询/处理过程中需能检测资源是否已变更并合理重试/放弃
- CRD 为自定义（用户已提供）

设计概要
-----------
- 基础库：使用 `k8s.io/client-go`（对应 k8s 1.35 的 client-go 版本），`k8s.io/apimachinery`，以及 `k8s.io/client-go/dynamic` 和 `dynamicinformer`。不使用 controller-runtime。
- 事件驱动 + 工作队列：使用 informer（dynamic informer）监听自定义资源的 add/update/delete 事件，统一入列到 `workqueue.RateLimitingInterface`。由固定数量 worker goroutine 消费队列。
- 并发控制：在单个副本内使用本地 per-key mutex（例如 sync.Mutex 存于 sync.Map）保证同一资源不会被同一进程的多个 worker 并发处理；跨副本使用 Kubernetes 的 Lease（coordination.k8s.io/v1 Lease）作为分布式锁来保证多个副本不会同时对同一资源执行对立操作。
- Leader election（可选）：对需要全局单实例执行的任务（例如定时全局清理、迁移），可以使用 client-go 的 leader election 支持（基于 Lease）。但常规的资源级并发控制依赖 per-resource Lease 而非单一 leader，从而允许多副本并行处理不同资源。

组件与关键流程
----------------
1. 客户端与 informer
   - 使用 dynamic.Interface 处理 CRD（无需 codegen）。
   - 通过 dynamicinformer.NewFilteredDynamicInformer 创建 namespace-scoped informer（或对所有命名空间根据部署需求）监听 `example.com/v1beta1` 的资源。
   - 事件处理函数（Add/Update/Delete）统一构建 key：`<namespace>/<name>` 并入队。

2. 工作队列
   - 使用 `workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "tst-controller")`。
   - 启动 N 个 worker goroutine（N 可配置，通常与副本数/CPU 相关）。
   - 每个 worker 从队列取 key，执行处理函数 `processItem(key)`，失败则 rate-limit requeue。

3. 本地 & 分布式锁设计
   - 本地锁：实现一个 sync.Map<string, *sync.Mutex>，在处理 key 前先 Lock 本地 mutex，处理完 Unlock，防止同一 pod 内的并发处理冲突。
   - 分布式锁（跨 pod）：为每个资源采用一个 Lease 对象，命名规则例如 `tst-lock-<namespace>-<name>`，使用 `coordination.k8s.io/v1` 的 Lease 资源：
     - 处理前尝试 Acquire：创建或 Patch Lease，将 holderIdentity 设为本 pod（环境变量 POD_NAME）并设置短 TTL（例如 15s），周期性 renew。
     - 获取失败（已被其它 pod 占用且未过期）则返回并把 key 以延迟（Jitter）重新入队。
     - 当本次处理完成或遇到不可恢复错误时，删除或释放 Lease（Patch 清空 holder 或直接删除 Lease）。
   - 使用 Lease 的同时注意使用 ResourceVersion 做乐观并发控制，确保对 Lease 的 Patch/Update 是安全的（进行 atomic Patch）。

4. 处理流程细节（processItem）
   - 从 informer 的缓存读取当前资源（unstructured）及其 metadata.generation / resourceVersion。
   - 本地 Lock + 尝试 Acquire 分布式 Lease。
   - 再次 GET 资源以确保未在获取 Lease 到处理之间发生变更（比较 resourceVersion/generation）。如果变更则 Release Lease + requeue (立即或延迟), 以确保处理的是最新状态。
   - 执行具体业务逻辑（可分步、并在每步前后检测 resourceVersion 是否变化以决定是否继续）。
   - Status 更新：使用 Patch（建议 types.MergePatchType 或 JSONPatch 视需求）仅修改 `status` 子资源。示例（伪 JSON）:
     - MergePatch: {"status":{"conditions":[{"type":"Ready","status":"True","lastTransitionTime":"..."}]}}
     - 使用 dynamic client 或 REST client：Resource(namespace).Patch(ctx, name, types.MergePatchType, patchBytes, metav1.PatchOptions{}, "status")
   - 在提交 Status patch 之前，再 GET 一次资源并比较 resourceVersion，若已改变则不要覆盖（放弃本次状态更新并 requeue）。
   - Release 本地 lock 与 Lease。

5. 检测资源变更策略
   - 优先使用 metadata.generation 判断 spec 是否变更（generation 变化表明 spec 更新）。
   - 对所有变更（status 或非结构化字段），使用 metadata.resourceVersion 做最终一致性比较。处理过程中任何关键步骤前后都应 re-get 并对比 resourceVersion；若检测到变更，可选择：
     - 立即放弃当前处理并将 key 重新入列以让最新事件驱动处理；或
     - 合并/补充处理（视业务场景）。

6. Status Patch 细节
   - 使用 Patch 而不是 Update：Patch 更安全且能保证只修改 status 子资源；避免覆盖其它控制器/用户在 status 上的并发更改。
   - 使用子资源 `status`：在 Patch 调用中传入 subresource "status"。
   - 在 Patch 前后都 re-get 以避免覆盖已更新的 status（并用 resourceVersion 检测）。

7. RBAC 需求（最小权限）
   - 对自定义资源： get/list/watch/patch/status
   - 对 Lease 资源： get/create/patch/update/delete
   - 对 events/ConfigMaps（如需要）：create/update
   - 必要时权限细化到命名空间范围。

实现注意事项与最佳实践
----------------------
- client-go 版本应与集群版本对齐（k8s v1.35 -> client-go v0.35.x）。
- 动态客户端处理 unstructured，需要对字段做严格校验（避免 runtime panic）。
- 启动参数：worker 数量、lease ttl、lease-renew-interval、queue 重试策略应可配置。
- 处理超时保护：每次处理应有上下文超时，超时则释放 Lease 并 requeue。
- 指标与事件：导出 Prometheus 指标（队列长度、处理时长、失败率、lease 获取失败次数等）；为重要变化记录 Kubernetes Event。
- 测试：编写单元测试（对业务逻辑），集成测试使用 kind/minikube，模拟多副本并发场景验证 Lease 行为。
- 日志：结构化日志（建议使用 klog/v2 或 zap），包含 key、pod、lease 状态、resourceVersion。

CRD 示例要点（namespace scope）
- apiVersion: apiextensions.k8s.io/v1
- scope: Namespaced
- spec.versions: - name: v1beta1, served: true, storage: true（或根据迁移需要配置）

部署建议
---------
- 使用 Deployment，replicas >= 2（测试多副本）。
- 在 Pod 中注入环境变量 POD_NAME（downward API）与 POD_NAMESPACE 以用于 Lease holderIdentity 与日志。
- 给 Operator 的 ServiceAccount 绑定必要的 RBAC。

总结
-----
该方案利用 client-go 的 informer + dynamic client + workqueue 实现事件驱动控制器，结合本地 mutex 与基于 Kubernetes Lease 的分布式锁，既能在多副本场景下并行处理不同资源，又能保证对同一资源的互斥处理。Status 更新使用 Patch 到 status 子资源，且在关键步骤通过 resourceVersion / generation 做变更检测以避免覆盖或处理过时数据。整体方案避免使用任何自动生成工具或 controller 框架，兼容 k8s v1.35 的最新 API。

如需我将此设计细化为接口与关键伪代码（不使用 codegen），或生成 CRD/YAML、RBAC、Deployment 清单，请告知，我可以继续输出实现级别的设计或代码样例。
