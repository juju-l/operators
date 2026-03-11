# Hlm CRD 与示例：保留未知字段

我直接给你 **CRD YAML + 示例 HLM CR YAML + 已经打开 ** **`preserveUnknownFields: true`**，你复制就能用。

---

# 1. hlm-crd.yaml（CRD 定义）

```YAML

apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: hlms.example.com
spec:
  group: example.com
  names:
    kind: Hlm
    listKind: HlmList
    plural: hlms
    singular: hlm
    shortNames:
      - hlm
  scope: Namespaced
  versions:
    - name: v1beta1
      served: true
      storage: true
      schema:
        openAPIV3Schema:
          type: object
          # 允许未知字段，你在 CR 里随便加字段都不会被校验拒绝
          x-kubernetes-preserve-unknown-fields: true
          properties:
            spec:
              type: object
              # 这里你后面可以慢慢加字段，现在先全开
              x-kubernetes-preserve-unknown-fields: true
            status:
              type: object
              x-kubernetes-preserve-unknown-fields: true
      subresources:
        # 启用 status 子资源（你的 controller 更新 status 必须要）
        status: {}
```

作用：

- 你在 **HlmSpec 里随便写字段**，都不会被 K8s 校验拒绝

- 启用了 `/status` 子资源，你之前的 `UpdateStatus` 才能正常工作

---

# 2. example-hlm.yaml（示例 HLM CR）

```YAML

apiVersion: example.com/v1beta1
kind: Hlm
metadata:
  name: my-hlm
  namespace: default
spec:
  # 这里你可以随便写任何字段，CRD 已经允许未知字段
  replicas: 3
  image: "nginx:alpine"
  resources:
    cpu: "100m"
    memory: "128Mi"
  anyOtherField: "可以随便加"
  a: 1
  b:
    c: test
status:
  ready: true
  phase: Running
```

你现在 **完全不用先定义 Spec 结构**，直接在 CR 里写任何字段都能存进去。

---

# 3. 你在 Go 里怎么读取这些任意字段？

因为你开了 `preserveUnknownFields`，建议把 `HlmSpec` 改成：

## hlm/types.go

```Go

type HlmSpec struct {
    // 用 map 接收所有字段，包括你没定义的
    Raw map[string]interface{} `json:"-"`
}
```

或者更简单、**最适合你现在这种场景**：

```Go

import "k8s.io/apimachinery/pkg/runtime"

type HlmSpec struct {
    // 直接接收所有 JSON
    runtime.RawExtension `json:",inline"`
}
```

这样：

- CR 里的所有 spec 字段都会进到 `Spec.Raw` 里

- 你后面再慢慢解析，**不需要提前定义结构体**

---

# 你现在只需要做 3 步：

1. `kubectl apply -f hlm-crd.yaml`

2. `kubectl apply -f example-hlm.yaml`

3. 运行你刚才的 controller，它就能：

    - 监听 Hlm

    - 读取 Spec

    - 更新 Status

    - 不会报任何校验错误

需要我再把 **HlmSpec 改成 runtime.RawExtension 的完整 types.go** 直接给你吗？
> （注：文档部分内容可能由 AI 生成）