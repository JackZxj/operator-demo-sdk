/*
Copyright 2021.

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
	"time"

	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	demov1alpha1 "operator-demo-sdk/api/v1alpha1"
)

// SampleReconciler reconciles a Sample object
type SampleReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=demo.test.io,resources=samples,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=demo.test.io,resources=samples/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=demo.test.io,resources=samples/finalizers,verbs=update

//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Sample object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.8.3/pkg/reconcile
func (r *SampleReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("sample", req.NamespacedName)

	// ??????????????? sample ??????
	sample := &demov1alpha1.Sample{}
	if err := r.Get(ctx, req.NamespacedName, sample); err != nil {
		if errors.IsNotFound(err) {
			// ??????????????? sample ??????
			log.Info("Sample resource not found. Ignoring since object must be deleted")
			// ?????? sample ?????????????????????????????????????????? sample
			return ctrl.Result{}, nil
		}
		// ?????? Sample ????????????
		log.Error(err, "Failed to get Sample")
		return ctrl.Result{}, err
	}
	// ???????????? Sample ??????????????? sample ??????
	nsn := req.NamespacedName
	if sample.Spec.DeployName != "" {
		nsn.Name = sample.Spec.DeployName
	}
	deploy := &appsv1.Deployment{}
	if err := r.Get(ctx, nsn, deploy); err != nil {
		if errors.IsNotFound(err) {
			// ??????????????? deployment ??????
			log.Info("Deployment resource not found. Create it.")
			// ?????? Deployment ??????
			deploy = newDeployment(nsn.Namespace, nsn.Name, sample.Spec.Replicas)
			// ????????????
			ctrl.SetControllerReference(sample, deploy, r.Scheme)
			// ?????? Deployment
			if err := r.Create(ctx, deploy); err != nil {
				log.Error(err, "can not create the target deployemnt:", "namespace", nsn.Namespace, "name", nsn.Name)
				return ctrl.Result{}, err
			}
		}
		// ?????? deployment ????????????
		log.Error(err, "Failed to get Deployment")
		return ctrl.Result{}, err
	}
	// ?????? deployment ????????? crd ??????
	if !metav1.IsControlledBy(deploy, sample) {
		log.Error(fmt.Errorf("deployment is not controlled by Sample controller"), "controller", "namespace:", req.Namespace, "name", req.Name)
		return ctrl.Result{}, nil
	}
	// ????????? CRD ??????????????????
	if *deploy.Spec.Replicas != *sample.Spec.Replicas {
		deploy.Spec.Replicas = sample.Spec.Replicas
		if err := r.Update(ctx, deploy); err != nil {
			log.Error(err, "can not update deployemnt", "namespace:", nsn.Namespace, "name", nsn.Name)
			return ctrl.Result{}, nil
		}
	}

	// ??????????????? CRD ??? status
	if sample.Status.AvailableReplicas != deploy.Status.AvailableReplicas {
		sample.Status.AvailableReplicas = deploy.Status.AvailableReplicas
		if err := r.Status().Update(ctx, sample); err != nil {
			log.Error(err, "Failed to update NatsCo status")
			return ctrl.Result{}, err
		} else {
			log.Info("sample", sample.Name, "Status has updated")
		}
	}

	// ?????????????????? 5 ?????????????????????
	return ctrl.Result{RequeueAfter: time.Second * 5}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *SampleReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&demov1alpha1.Sample{}).
		Owns(&appsv1.Deployment{}).
		Complete(r)
}

func newDeployment(namespace, name string, replicas *int32) *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"sample": name},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"sample": name},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{Name: "nginx", Image: "nginx:alpine"}},
				},
			},
		},
	}
}
