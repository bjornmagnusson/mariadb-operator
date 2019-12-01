/*

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/go-logr/logr"
	"golang.org/x/crypto/ssh/terminal"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes/scheme"
	corev1client "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/remotecommand"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	mariadbv1beta1 "github.com/bjornmagnusson/mariadb-operator/api/v1beta1"
)

// DatabaseServerReconciler reconciles a DatabaseServer object
type DatabaseServerReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=mariadb.bjornmagnusson.com,resources=databaseservers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=mariadb.bjornmagnusson.com,resources=databaseservers/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=*,resources=pods,verbs=get;list
// +kubebuilder:rbac:groups=*,resources=pods/exec,verbs=create

func (r *DatabaseServerReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("databaseserver", req.NamespacedName)

	log.Info("Reconsile started")

	var instance mariadbv1beta1.DatabaseServer
	if err := r.Get(ctx, req.NamespacedName, &instance); err != nil {
		if errors.IsNotFound(err) {
			// Object not found, return.  Created objects are automatically garbage collected.
			// For additional cleanup logic use finalizers.
			log.Error(err, "unable to find DatabaseServer")
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	spec := instance.Spec
	var replicas int32
	replicas = 1

	labels := map[string]string{
		"app.kubernetes.io/name":       instance.Name,
		"app.kubernetes.io/managed-by": "mariadb-operator",
	}

	log.Info("-- DEPLOYMENT START --")

	deploy := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      spec.Name,
			Namespace: "default",
			Labels:    labels,
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Replicas: &replicas,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: v1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  spec.Name,
							Image: spec.Image,
							Ports: []corev1.ContainerPort{
								{
									Name:          "db",
									ContainerPort: 3306,
								},
							},
							Env: []corev1.EnvVar{
								{
									Name:  "MYSQL_ROOT_PASSWORD",
									Value: spec.Password,
								},
							},
							ReadinessProbe: &corev1.Probe{
								Handler: v1.Handler{
									Exec: &v1.ExecAction{
										Command: []string{
											"mysql",
											"-uroot",
											"-p" + spec.Password,
											"-estatus",
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	found := &appsv1.Deployment{}
	var isNew bool
	err := r.Get(ctx, types.NamespacedName{Name: deploy.Name, Namespace: deploy.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		log.Info("Creating Deployment", "namespace", deploy.Namespace, "name", deploy.Name)
		err = r.Create(ctx, deploy)
		isNew = true
		if err != nil {
			return reconcile.Result{}, err
		}
	} else if err != nil {
		return reconcile.Result{}, err
	}

	if !reflect.DeepEqual(deploy.Spec, found.Spec) && !isNew {
		log.Info("Deleting Deployment (update)", "namespace", deploy.Namespace, "name", deploy.Name)
		_ = r.Delete(ctx, found)
		log.Info("Creating Deployment (update)", "namespace", deploy.Namespace, "name", deploy.Name)
		_ = r.Create(ctx, deploy)
	}

	log.Info("-- DEPLOYMENT END --")

	log.Info("-- SERVICE START --")

	var svcPort int32
	if spec.Port > 0 {
		svcPort = spec.Port
	} else {
		svcPort = 3306
	}

	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      spec.Name + "-service",
			Namespace: "default",
			Labels:    labels,
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Name: "db",
					Port: svcPort,
					TargetPort: intstr.IntOrString{
						IntVal: 3306,
					},
				},
			},
			Selector: labels,
			Type:     "LoadBalancer",
		},
	}

	foundSvc := &corev1.Service{}
	errSvc := r.Get(ctx, types.NamespacedName{Name: service.Name, Namespace: service.Namespace}, foundSvc)
	if errSvc != nil && errors.IsNotFound(errSvc) {
		log.Info("Creating Service", "namespace", service.Namespace, "name", service.Name)
		err = r.Create(ctx, service)
	} else if err != nil {
		return reconcile.Result{}, err
	}

	log.Info("-- SERVICE END --")

	log.Info("-- DEPLOYMENT STATUS START --")
	log.Info("Waiting 10s to check status")
	time.Sleep(10 * time.Second)
	deploymentIsRunning := false
	for !deploymentIsRunning {
		foundStatus := &appsv1.Deployment{}
		err := r.Get(ctx, types.NamespacedName{Name: deploy.Name, Namespace: deploy.Namespace}, foundStatus)
		if err != nil {
			log.Error(err, "Failed to lookup deployment status")
		} else if foundStatus.Status.UnavailableReplicas == 0 {
			log.Info("Deployment is ready")
			deploymentIsRunning = true
		} else {
			log.Info("MariaDB is not running, retrying in 10s",
				"replicas", foundStatus.Status.Replicas,
				"ready-replicas", foundStatus.Status.ReadyReplicas,
				"unavailable-replicas", foundStatus.Status.UnavailableReplicas)
			time.Sleep(10 * time.Second)
		}
	}
	log.Info("-- DEPLOYMENT STATUS END --")

	log.Info("-- DATABASE START --")
	var childPods corev1.PodList
	if err := r.List(ctx, &childPods, client.InNamespace(req.Namespace), client.MatchingLabels{
		"app.kubernetes.io/managed-by": "mariadb-operator",
		"app.kubernetes.io/name":       req.Name}); err != nil {
		log.Error(err, "unable to list child ReplicaSet")
		return ctrl.Result{}, err
	}

	var dbPod corev1.Pod
	for i, pod := range childPods.Items {
		log.Info(strconv.Itoa(i+1) + ") " + pod.Name)
		if pod.Status.Phase == corev1.PodRunning {
			dbPod = pod
		}
	}

	log.Info("MariaDB: Will connect to " + dbPod.Name)
	baseExecOptions := corev1.PodExecOptions{
		Stdin:  true,
		Stdout: true,
		Stderr: true,
		TTY:    true,
	}

	statusExecOptions := baseExecOptions.DeepCopy()
	statusExecOptions.Command = []string{
		"mysql",
		"-uroot",
		"-p" + spec.Password,
		"-estatus",
	}
	executeCommand(dbPod, *statusExecOptions, log)

	showDatabaseExecOptions := baseExecOptions.DeepCopy()
	showDatabaseExecOptions.Command = []string{
		"mysql",
		"-uroot",
		"-p" + spec.Password,
		"-e",
		"SHOW DATABASES",
	}
	executeCommand(dbPod, *showDatabaseExecOptions, log)

	for i, db := range spec.Databases {
		log.Info(strconv.Itoa(i+1) + ") " + db.Name)
		createDatabaseExecOptions := baseExecOptions.DeepCopy()
		createDatabaseExecOptions.Command = []string{
			"mysql",
			"-uroot",
			"-p" + spec.Password,
			"-e",
			"CREATE DATABASE IF NOT EXISTS " + db.Name,
		}
		executeCommand(dbPod, *createDatabaseExecOptions, log)
	}
	executeCommand(dbPod, *showDatabaseExecOptions, log)
	log.Info("-- DATABASE END --")

	log.Info("Reconsile finished")

	return reconcile.Result{}, nil
}

var (
	jobOwnerKey = ".metadata.controller"
	apiGVStr    = appsv1.SchemeGroupVersion.Version
)

func executeCommand(pod corev1.Pod, execOptions corev1.PodExecOptions, log logr.Logger) {
	log.Info("Executing " + strings.Join(execOptions.Command, " "))

	// Instantiate loader for kubeconfig file.
	kubeconfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		clientcmd.NewDefaultClientConfigLoadingRules(),
		&clientcmd.ConfigOverrides{},
	)

	// Get a rest.Config from the kubeconfig file.  This will be passed into all
	// the client objects we create.
	restconfig, err := kubeconfig.ClientConfig()
	if err != nil {
		panic(err)
	}

	// Create a Kubernetes core/v1 client.
	coreclient, err := corev1client.NewForConfig(restconfig)
	if err != nil {
		panic(err)
	}
	req := coreclient.RESTClient().
		Post().
		Namespace(pod.Namespace).
		Resource("pods").
		Name(pod.Name).
		SubResource("exec").
		VersionedParams(&execOptions, scheme.ParameterCodec)

	log.Info("Command: " + req.URL().String())

	exec, err := remotecommand.NewSPDYExecutor(restconfig, "POST", req.URL())
	if err != nil {
		panic(err)
	}

	// Put the terminal into raw mode to prevent it echoing characters twice.
	oldState, err := terminal.MakeRaw(0)
	if err != nil {
		panic(err)
	}
	defer terminal.Restore(0, oldState)

	// Connect this process' std{in,out,err} to the remote shell process.
	err = exec.Stream(remotecommand.StreamOptions{
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
		Tty:    true,
	})
	if err != nil {
		panic(err)
	}

	fmt.Println()
}

func (r *DatabaseServerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := mgr.GetFieldIndexer().IndexField(&appsv1.ReplicaSet{}, jobOwnerKey, func(rawObj runtime.Object) []string {
		deployment := rawObj.(*appsv1.ReplicaSet)
		owner := metav1.GetControllerOf(deployment)
		if owner == nil {
			return nil
		}
		// ...make sure it's a CronJob...
		if owner.APIVersion != apiGVStr || owner.Kind != "Deployment" {
			return nil
		}

		// ...and if so, return it
		return []string{owner.Name}
	}); err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&mariadbv1beta1.DatabaseServer{}).
		Owns(&appsv1.ReplicaSet{}).
		Complete(r)
}
