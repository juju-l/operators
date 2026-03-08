module github.com/juju-l/helms

go 1.26

require (
	helm.sh/helm/v4 v4.1.1
	k8s.io/api v0.30.3
	k8s.io/apimachinery v0.30.3
	k8s.io/client-go v0.30.3
	sigs.k8s.io/controller-runtime v0.18.0
)

require (
	// 自动依赖，由 go mod tidy 生成
)