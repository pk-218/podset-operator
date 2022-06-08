/*
Copyright 2022.

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
	"reflect"

	// don't forget to add the particular version of the API in the import path
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	appv1alpha1 "github.com/pk-218/pod-set/api/v1alpha1"
)

// PodSetReconciler reconciles a PodSet object
type PodSetReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=app.github.com,resources=podsets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=app.github.com,resources=podsets/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=app.github.com,resources=podsets/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the PodSet object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.2/pkg/reconcile
func (r *PodSetReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	// fetch the PodSet instance
	instance := &appv1alpha1.PodSet{}
	// context.TODO() is passed when we might have to cancel some long running task mid-way
	err := r.Client.Get(context.TODO(), req.NamespacedName, instance)
	if err != nil {
		return ctrl.Result{}, err
	}

	// now, from the instance, we need to get the list of pods
	podList := &corev1.PodList{}

	// the pods are to be selected according to the labels
	labelSelectorSet := map[string]string{
		"app":     instance.Name,
		"version": "v0.1",
	}
	labelSelector := labels.SelectorFromSet(labelSelectorSet)

	// obtain the list options
	listOptions := &client.ListOptions{LabelSelector: labelSelector, Namespace: instance.Namespace}

	// obtain the podList from the client after giving it the listOptions
	if err = r.Client.List(context.TODO(), podList, listOptions); err != nil {
		return ctrl.Result{}, err
	}

	// from the podList, identify the available pods according to their phase i.e., PodRunning or PodPending
	var availablePods []corev1.Pod
	for _, pod := range podList.Items {
		if pod.ObjectMeta.DeletionTimestamp != nil {
			continue
		}
		if pod.Status.Phase == corev1.PodRunning || pod.Status.Phase == corev1.PodPending {
			availablePods = append(availablePods, pod)
		}
	}

	currentAvailablePods := int32(len(availablePods))
	availablePodNames := []string{}

	// get the names of the available pods
	for _, pod := range availablePods {
		availablePodNames = append(availablePodNames, pod.ObjectMeta.Name)
	}

	// after we know which and how many pods are available, the controller can act upon the current state by updating the status
	status := appv1alpha1.PodSetStatus{
		PodNames: availablePodNames,
	}

	// check if desired status is equal to the current state
	if !reflect.DeepEqual(instance.Status, status) {
		instance.Status = status
		err = r.Client.Status().Update(context.TODO(), instance)
		if err != nil {
			log.Log.Error(err, "Failed to update status of PodSet")
			return ctrl.Result{}, err
		}
	}

	// if there are more pods than number of specified replicas --> scale down
	if currentAvailablePods > instance.Spec.Replicas {
		log.Log.Info("Scaling down PodSet", "currently available", currentAvailablePods, "required", instance.Spec.Replicas)
		difference := currentAvailablePods - instance.Spec.Replicas
		podsToBeDestroyed := availablePods[:difference]
		for _, soonToBeDestroyedPod := range podsToBeDestroyed {
			err = r.Client.Delete(context.TODO(), &soonToBeDestroyedPod)
			log.Log.Error(err, "Failed to delete Pod from PodSet", soonToBeDestroyedPod.Name)
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	// if there are less pods than number of specified replicas --> scale up
	if currentAvailablePods < instance.Spec.Replicas {
		log.Log.Info("Scaling up PodSet", "currently available", currentAvailablePods, "required", instance.Spec.Replicas)
		pod := newPodForPodSetCustomResource(instance)

		// set PodSet instance as the owner and controller
		if err = controllerutil.SetControllerReference(instance, pod, r.Scheme); err != nil {
			return ctrl.Result{}, err
		}
		err = r.Client.Create(context.TODO(), pod)
		if err != nil {
			log.Log.Error(err, "Failed to create a new Pod for the PodSet custom resource")
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}
	return reconcile.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *PodSetReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&appv1alpha1.PodSet{}).
		Complete(r)
}

// new Pod created for scaling up the PodSet CR
func newPodForPodSetCustomResource(cr *appv1alpha1.PodSet) *corev1.Pod {
	labelsForNewPod := map[string]string{
		"app":     cr.Name,
		"version": "v0.1",
	}

	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: cr.Name + "-pod-",
			Namespace:    cr.Namespace,
			Labels:       labelsForNewPod,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:    "busybox",
					Image:   "busybox",
					Command: []string{"sleep", "3600"},
				},
			},
		},
	}
}
