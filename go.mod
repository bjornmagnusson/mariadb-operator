module github.com/bjornmagnusson/mariadb-operator

go 1.13

require (
	github.com/go-logr/logr v0.2.0
	github.com/onsi/ginkgo v1.11.0
	github.com/onsi/gomega v1.7.0
	golang.org/x/crypto v0.0.0-20200622213623-75b288015ac9
	k8s.io/api v0.20.0-alpha.2
	k8s.io/apimachinery v0.20.0-alpha.2
	k8s.io/client-go v0.20.0-alpha.2
	k8s.io/klog v1.0.0 // indirect
	sigs.k8s.io/controller-runtime v0.4.0
)
