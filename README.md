# Hlm Operator（最小可运行示例）

## 简介

这是一个使用原生 client-go + Helm v4 SDK（v4.1.1）实现的轻量级 Operator 骨架，用于通过自定义资源（CRD: `Hlm`）管理 Helm chart 的安装、升级与卸载。项目不依赖 controller-runtime / operator-sdk，便于理解原生实现细节。

## 设计目标

- 使用原生 Kubernetes 客户端（dynamic client + informer）监听 CR 并进行 reconcile。 
- 使用 Helm v4 SDK 执行 template、install、upgrade、uninstall、rollback 操作。
- 将渲染结果外部化保存到 ConfigMap（保留最近 7 次），在 CR 的 status 中记录最近一次的 ConfigMap 名称。 
- 使用 finalizer 确保在 CR 删除前完成 Helm 卸载。 
- Status 更新采用 server-side apply，带指数退避重试（可通过环境变量调整）。

## 组件文件说明

- `types.go`：CR Spec/Status 定义
- `api.go`：将 CR 类型注册到 scheme（Group: `tpl.vipex.cc`, Version: `v1alpha1`）
- `controllers.go`：原生 controller 主循环与 reconcile 逻辑（包含 finalizer、status patch、ConfigMap 存储、rollback）
- `helms.go`：封装 Helm SDK 的 install/upgrade/uninstall/template/rollback
- `main.go`：operator 入口（创建 helm client 与 controller）
- `customResourceDefinitions.yaml`：CRD 定义
- `tpl_vipex_cc-0.1.0.tgz`：示例 chart 包（需有效）
- `sample-hlm.yaml`：示例 Hlm CR

## 先决条件

- Go >= 1.20
- 能访问 Kubernetes 集群（在集群内运行或通过 KUBECONFIG）
- `tpl_vipex_cc-0.1.0.tgz` 为合法 Helm chart 或指定为可访问仓库的 chart 名称

## 快速部署（本地 or 集群）

1. 拉取依赖并编译

    ```bash
    go mod tidy
    go build -v ./...
    ```

2. 在集群中注册 CRD

    ```bash
    kubectl apply -f customResourceDefinitions.yaml
    ```

3. 部署 RBAC 与 Operator（示例清单已集成在下文）

完整可直接 apply 的 RBAC 与 Deployment 示例

以下 YAML 可直接保存为 `hlm-operator-deploy.yaml` 并 apply。请根据需要调整 `namespace` 与 `image` 字段。

```yaml
# Namespace（可选）
apiVersion: v1
kind: Namespace
metadata:
  name: default
---
# ServiceAccount
apiVersion: v1
kind: ServiceAccount
metadata:
  name: hlm-operator-sa
  namespace: default
---
# Role: 最小命名空间权限（如果需要跨命名空间或集群级资源，请使用 ClusterRole）
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: hlm-operator-role
  namespace: default
rules:
  - apiGroups: ["tpl.vipex.cc"]
    resources: ["hlms", "hlms/status"]
    verbs: ["get","list","watch","update","patch"]
  - apiGroups: [""]
    resources: ["configmaps", "pods", "events"]
    verbs: ["get","list","watch","create","update","patch","delete"]
---
# RoleBinding
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: hlm-operator-rb
  namespace: default
subjects:
  - kind: ServiceAccount
    name: hlm-operator-sa
    namespace: default
roleRef:
  kind: Role
  name: hlm-operator-role
  apiGroup: rbac.authorization.k8s.io
---
# Deployment
apiVersion: apps/v1
kind: Deployment
metadata:
  name: hlm-operator
  namespace: default
spec:
  replicas: 1
  selector:
    matchLabels:
      app: hlm-operator
  template:
    metadata:
      labels:
        app: hlm-operator
    spec:
      serviceAccountName: hlm-operator-sa
      containers:
        - name: hlm-operator
          image: your-registry/hlm-operator:latest # <--- 替换为你的镜像
          imagePullPolicy: IfNotPresent
          env:
            - name: NAMESPACE
              value: "default"
            - name: PATCH_MAX_ATTEMPTS
              value: "5"
            - name: PATCH_BASE_DELAY_MS
              value: "200"
          args: ["-namespace", "default"]
          resources:
            requests:
              cpu: 100m
              memory: 128Mi
            limits:
              cpu: 500m
              memory: 512Mi
```

> 如果需要 operator 拥有集群级权限（例如 chart 会创建 ClusterRole），请改用 ClusterRole 与 ClusterRoleBinding，并扩大 rules 中对应的 apiGroups/resources。

## 运行 Operator（本地）

- 使用 KUBECONFIG（例如你的 ~/.kube/config）：

    ```bash
    export KUBECONFIG=$HOME/.kube/config
    go run ./
    ```

或构建镜像并部署为 Deployment（按上面清单）。

## Status 字段说明（所有可能值）

Hlm.Status 字段（重要）说明：

- LastReconcile (Time)
  - 最后一次 operator 对该 CR 执行 reconcile 的时间。

- LastAction (string)
  - 最近执行的动作，可能值：
    - "Template"：对 chart 进行了渲染（helm template）
    - "InstallOrUpgrade"：执行了 install 或 upgrade
    - "Uninstall"：执行了 uninstall
    - "Rollback"：在升级失败后执行了回滚

- LastActionStatus (string)
  - 最近动作的结果，可能值："Succeeded" 或 "Failed"

- LastActionMessage (string)
  - 对应动作的简短说明或错误消息（便于排查）

- LastRevision (int)
  - Helm release 的最后已知 revision（如果可用）

- RenderedManifestConfigMap (string)
  - 存储最近一次渲染结果的 ConfigMap 名称（格式：<cr-name>-rendered-<timestamp>）。operator 会为每次渲染创建一个时间戳命名的 ConfigMap，并保留最近 7 次记录。

## 状态场景示例

- 安装成功：
  - LastAction="InstallOrUpgrade"，LastActionStatus="Succeeded"，LastRevision=1
  - RenderedManifestConfigMap 指向最新渲染的 ConfigMap

- 渲染失败：
  - LastAction="Template"，LastActionStatus="Failed"，LastActionMessage 包含错误信息

- 升级失败并回滚成功：
  - 先写入 InstallOrUpgrade Failed，然后尝试 Rollback，最后写入 LastAction="Rollback" LastActionStatus="Succeeded"

- 删除（卸载）成功：
  - LastAction="Uninstall" LastActionStatus="Succeeded"，并在删除完成后移除 finalizer

## 测试用例（操作步骤与预期）

准备：已 apply CRD、部署 operator 或在本地运行 operator。

用例 1 — 基本安装

1. 应用示例 CR：
   kubectl apply -f sample-hlm.yaml
2. 检查渲染结果的 ConfigMap：
   kubectl get configmap -n default -l hlm-rendered-of=app07-hlm
   预期：存在名为 `app07-hlm-rendered-<ts>` 的 ConfigMap，并包含键 `manifest`。
3. 检查 CR status：
   kubectl get hlm app07-hlm -n default -o yaml
   预期：status.LastAction=InstallOrUpgrade，LastActionStatus=Succeeded，LastRevision >= 1，RenderedManifestConfigMap 指向上一步的 ConfigMap 名称。

用例 2 — 卸载（删除 CR）

1. 删除 CR：
   kubectl delete hlm app07-hlm -n default
2. 观察 CR 的删除流程（finalizer）：
   kubectl get hlm app07-hlm -n default -o yaml --show-managed-fields
   预期：在 finalizer 被清除之前，operator 会调用 Helm uninstall；卸载成功后 finalizer 被移除，资源最终消失。

用例 3 — 渲染历史保留

1. 多次修改 spec（例如修改 spec.values.image.tag）并重新 apply，观察 ConfigMap 数量：
   kubectl get configmap -n default -l hlm-rendered-of=app07-hlm
2. 预期：operator 为每次 render 创建新的 ConfigMap，但只保留最近 7 个，较早的会被自动删除。

用例 4 — 模拟升级失败并回滚（手动触发）

说明：要模拟升级失败，你可以通过使 chart 中的某个资源模板产生不合法的内容或在 chart 中设置一个会导致 API 服务器拒绝的字段。步骤示例：

1. 在 chart （或指定的 values）中引入一个非法字段，或把 image 指向一个不存在的自定义资源（导致 Admission/Validations 失败）；
2. 更新 CR 的 spec.values 并 apply，使 operator 执行升级；
3. 观察 CR status：
   - 预期：先出现 InstallOrUpgrade Failed，并随后出现 Rollback 的记录（Succeeded 或 Failed，根据回滚结果）。

## 注意与调优

- RenderedManifestConfigMap 可能会包含敏感信息（如 secret 的明文）；在生产中务必谨慎处理，建议对敏感信息做脱敏或不要把 secret 值渲染并保存。 
- status 更新采用 server-side apply，可能会与其他系统的 fieldManager 冲突，系统设置了指数退避重试并可通过环境变量调整：
  - PATCH_MAX_ATTEMPTS（默认 5）
  - PATCH_BASE_DELAY_MS（默认 200 毫秒）

## 清理

- 删除 operator（Deployment + Role/RoleBinding + ServiceAccount）
- 删除 CRD：
  kubectl delete -f customResourceDefinitions.yaml

## 后续改进建议

- 将部分复杂的 Spec 子结构建模为更严格的 Go struct 并在 CRD 中生成完整 openAPIV3 schema 校验；
- 将 RenderedManifest 存储移到外部存储（例如 S3）并在 status 中只保留引用，以避免 ConfigMap 体积限制与敏感信息问题；
- 提供更完善的重试/告警与 metrics（Prometheus）支持；
- 增加 e2e 测试脚本（使用 kind / kindctl）以自动化验证 operator 行为。

## 联系方式

如需继续扩展（生成 RBAC 与 Deployment 清单、改进 CRD schema 或实现 e2e 测试脚本），回复你想要的项，我会继续实现。

