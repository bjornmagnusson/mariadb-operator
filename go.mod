module github.com/bjornmagnusson/mariadb-operator

go 1.13

require (
	github.com/evanphx/json-patch v4.5.0+incompatible
	github.com/go-logr/logr v0.1.0
	github.com/onsi/ginkgo v1.10.1
	github.com/onsi/gomega v1.7.0
	golang.org/x/crypto v0.0.0-20190820162420-60c769a6c586
	k8s.io/api v0.0.0-20191121015604-11707872ac1c
	k8s.io/apimachinery v0.0.0-20191121015412-41065c7a8c2a
	k8s.io/cli-runtime v0.0.0-20191121021703-be566597aa73
	k8s.io/client-go v0.0.0-20191121015835-571c0ef67034
	sigs.k8s.io/controller-runtime v0.4.0
)
