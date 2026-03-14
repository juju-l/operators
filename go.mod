module github.com/juju-l/operators/tst-operator

go 1.20

require (
	k8s.io/api v0.35.0
	k8s.io/apimachinery v0.35.0
	k8s.io/client-go v0.35.0
)

replace k8s.io/api => k8s.io/api v0.35.0
replace k8s.io/apimachinery => k8s.io/apimachinery v0.35.0
replace k8s.io/client-go => k8s.io/client-go v0.35.0
