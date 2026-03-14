# 原生 K8s Operator 设计（group: example.com, kind: Tst, version: v1beta1）

该设计以“完全原生、无框架、兼容 Kubernetes 1.35”为原则，整合此前评估与优化建议，目标产出：一份可直接落地的 Operator 设计文档（Namespaced CRD、leader election、workqueue、status Patch、变更检测、可选分布式锁、RBAC 与部署模板）。

---

## 一、概要
- API: example.com/v1beta1
- Kind: Tst (Namespaced)
- 运行模式：Deployment 多副本 + LeaseLock leader election（coordination.k8s.io/v1）
- 并发控制：client-go workqueue（RateLimitingInterface）+ 同一 key 串行处理；按需 per-resource Lease 做跨实例互斥
- Status 更新：PATCH 到 status 子资源（types.MergePatchType），遇 409 做 GET+重建 Patch+指数退避重试
- 变更检测：优先用 metadata.generation，再用 spec 哈希（SHA256）与 status.lastHandledSpecHash 进行精细判定

---

## 二、总体架构
- 多副本部署（Deployment）；所有实例参与 leader election，只有 leader 启动 reconcile worker 并执行写操作
- SharedInformer / dynamicinformer 监听 Tst（以及下游相关资源），事件入队至 workqueue
- Worker 池（N workers）从 queue 取 key；对同一 key 在任意时刻只允许一个 goroutine 处理
- Reconcile 流程中：获取最新对象 -> 变更判定 -> 下游资源操作（SSA 或 Patch）-> Status Patch（MergePatch）-> 完成
- 可选：对需要跨资源原子性操作，使用命名的 Lease 作为分布式锁

---

## 三、CRD（示例，Namespaced，启用 status 子资源）
```yaml
apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: tsts.example.com
spec:
  group: example.com
  versions:
    - name: v1beta1
      served: true
      storage: true
      schema:
        openAPIV3Schema:
          type: object
          properties:
            spec:
              type: object
              # ... 定义业务 spec 字段 ...
            status:
              type: object
              properties:
                observedGeneration:
                  type: integer
                conditions:
                  type: array
                  items:
                    type: object
                lastHandledSpecHash:
                  type: string
      subresources:
        status: {}
  scope: Namespaced
  names:
    plural: tsts
    singular: tst
    kind: Tst
```

说明：在 `status` 中明确包含 `observedGeneration`、`conditions`、`lastHandledSpecHash`（用于变更检测），并启用 status 子资源。

---

## 四、客户端选择与 informer
- 若不手写 Go 类型，使用 `dynamic.Interface` + `dynamicinformer` + `unstructured.Unstructured`。
- 若选择手写类型（维护 Go struct），可使用 typed client + generated informer（但本设计目标是“无自动生成工具”，推荐 dynamic）。
- Informer 仅在事件阶段做快速过滤（使用 generation 和关键 spec 字段比较），并将 namespace/name 入队。

---

## 五、变更检测策略（高效避免冗余 reconcile）
1. Informer Update 事件阶段：比较 old.metadata.generation 与 new.metadata.generation；若相等且只有 status 变动，可跳过入队。
2. 在 Reconcile 开始（防御式）：GET API Server 上的最新对象（避免缓存 stale），计算 `specHash = sha256(spec)`（规范化 JSON 后哈希）。
3. 判定条件（需要执行下游变更时）：
   - object.metadata.generation != status.observedGeneration
   - 或 specHash != status.lastHandledSpecHash
4. 若上述均为 false，则跳过对下游资源的变更逻辑，但可选择同步/更新 conditions/status（如子资源状态需要同步）。

---

## 六、Reconcile 主流程（伪代码）

1. key := queue.Get()
2. ns,name := split(key)
3. obj := dynamicClient.Resource(gvr).Namespace(ns).Get(name)
4. if NotFound:
     - 如果 finalizer 存在，执行 cleanup 并移除 finalizer
     - queue.Forget(key); return
5. if obj.DeletionTimestamp != nil:
     - 执行删除清理逻辑（确保幂等），移除 finalizer
6. // 变更检测
   specHash := HashSpec(obj.spec)
   if obj.metadata.generation == obj.status.observedGeneration && specHash == obj.status.lastHandledSpecHash:
     - 可能仅 status 需更新或无需处理 -> 更新 status.conditions（若需要）并 queue.Forget(key); return
7. // 执行业务变更（下游资源）
   - 使用 Server-Side Apply（建议）或 Patch 更新下游资源；若用 SSA，统一 fieldManager 名称
8. // 更新 status（仅 status 子字段）
   patch := {"status": {"observedGeneration": obj.metadata.generation, "lastHandledSpecHash": specHash, "conditions": [...]}}
   try PATCH status (types.MergePatchType)
   on 409: do GET latest -> rebuild minimal patch -> retry with backoff (limit retries)
9. 完成：queue.Forget(key)

错误处理：临时错误 queue.AddRateLimited(key)

---

## 七、Status Patch 细节
- 使用 JSON Merge Patch（types.MergePatchType），只包含要修改的 status 字段，避免全量覆盖。
- PATCH 到 subresource `status`：
  dynamicClient.Resource(gvr).Namespace(ns).Patch(ctx, name, types.MergePatchType, patchBytes, metav1.PatchOptions{}, "status")
- 冲突处理：当返回 409（Conflict）时，重新 GET 最新对象、重建最小 patch、指数退避重试（例如 5 次上限），记录警报并退出时把 key 放回队列或标记 Condition 为失败。

Patch 示例：
```json
{ "status": { "observedGeneration": 7, "lastHandledSpecHash": "<sha256>", "conditions": [{"type":"Ready","status":"True","lastTransitionTime":"..."}] } }
```

---

## 八、并发控制与分布式锁
- 本地：workqueue 保证同一 key 在任意时刻只被一个 worker 处理；若仅 leader 执行 reconcile，则跨 Pod 也保证串行。
- 进程内互斥：通常不需要额外 sync.Map，但可作为防护层用于保护共享内存结构。
- 跨实例互斥（可选）：对需跨资源或跨多个对象的原子操作，使用 per-resource Lease（coordination.k8s.io/v1）作为分布式锁，命名规则如 `lease-<ns>-<name>-<op>`。
- 注意：频繁创建/删除 Lease 会增加 apiserver 负荷，建议按需重用并短期 renew。

---

## 九、Leader Election（推荐参数示例）
- 使用 client-go leaderelection + resourcelock.LeaseLock
- 建议参数：
  - leaseDuration: 15s
  - renewDeadline: 10s
  - retryPeriod: 2s
- 实现方式：只有在 leader 的回调中启动 reconcile worker 池；失去 leadership 时优雅停止 worker（等待当前 reconcile 完成或中止并 requeue）。

---

## 十、Server-Side Apply（SSA）与 Patch 的使用准则
- SSA 推荐用于管理 Kubernetes 下游普通资源（Deployment/Service/ConfigMap），并且为 operator 指定唯一 fieldManager 名称（例如 "tst-operator"）。
- Status 更新使用 Patch（subresource "status"），不要用 SSA 改写 status。
- 避免 SSA 与手工 Patch 在同一字段上混用，必要时在设计中划分 field ownership（文档化）。

---

## 十一、RBAC 最小权限（示例）
- 对 Tst CRD: get,list,watch,create,update,patch,delete
- status: update,patch
- leases: get,list,watch,create,update,patch
- 子资源（deployment/pods/configmaps...）: get,list,watch,create,update,patch,delete
```
# Role 示例（高层）
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
# ... namespace scope ...
```

---

## 十二、Deployment 模板（要点）
- replicas: >=2
- 启动参数示例：
  - --leader-election=true
  - --leader-election-namespace=<namespace>
  - --leader-election-id=tst-operator-lease
  - --workers=4
- probes: liveness/readiness
- metrics endpoint 启用用于 Prometheus

---

## 十三、可观测性与日志
- 指标：reconcile_duration_seconds, reconcile_errors_total, queue_depth, requeue_counts
- 关键日志字段：key, resourceVersion, observedGeneration, specHash, workerID, leader=true/false
- 报警：频繁 Conflict / 频繁 leader 切换 / 反复重试

---

## 十四、测试与验证计划
1. 单元：fake dynamic client 模拟 GET/PATCH/Conflict 场景
2. 集成（KinD）：部署 CRD + operator，测试：create/update/delete/failover/leader switch
3. E2E：并发修改、跨资源原子需求验证、per-resource Lease 场景测试
4. Chaos：Leader 宕机，验证 failover 与重新调度

---

## 十五、实施步骤（迭代优先级）
1. 编写 CRD YAML（Namespaced，启用 status，并加入 observedGeneration 与 lastHandledSpecHash）
2. 实现 dynamic client + dynamicinformer + workqueue，基础事件流
3. 实现 reconcile skeleton（GET 最新、变更检测、下游 SSA/patch、status patch + 409 重试）
4. 加入 leader election（只在 leader 启动并运行 workers）
5. 添加 metrics、health probes、RBAC、Deployment manifest
6. 编写单元与集成测试（KinD）
7. 如需跨实例强互斥，再引入 per-resource Lease

---

## 十六、注意事项与风险提醒（简要）
- 确保 client-go 版本与集群版本兼容（针对 k8s 1.35 使用对应 client-go/apimachinery 版本）
- Patch 时仅修改需要的 status 字段，避免丢失其它 controller 写入的 status
- SSA 与 Patch 的混用需谨慎，明确 field ownership
- Lease 参数不当会导致频繁切换或不可用，要基于集群特性调优

---

文档已导出至本地：C:\Users\Administrator\Downloads\example.com-Tst-operator-design.md

如需我把 CRD YAML、Reconcile 伪代码或 Deployment/RBAC manifest 单独拆出为文件，也可以一并导出。
