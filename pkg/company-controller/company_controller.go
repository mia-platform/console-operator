/*
Copyright 2025.

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

package companycontroller

import (
	"context"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	corev1alpha1 "github.com/mia-platform/console-operator/api/core/v1alpha1"
)

// CompanyReconciler reconciles a Company object
type CompanyReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=core.mia-platform.eu,resources=companies,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core.mia-platform.eu,resources=companies/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=core.mia-platform.eu,resources=companies/finalizers,verbs=update

func (r *CompanyReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx, "company", req.NamespacedName)
	ctx = ctrl.LoggerInto(ctx, log)

	var company corev1alpha1.Company
	if err := r.Get(ctx, req.NamespacedName, &company); err != nil {
		log.Error(err, "unable to fetch Company")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	finalizerName := "console-controller"
	if company.ObjectMeta.DeletionTimestamp.IsZero() {
		if !containsString(company.GetFinalizers(), finalizerName) {
			controllerutil.AddFinalizer(&company, finalizerName)
			if err := r.Update(ctx, &company); err != nil {
				return ctrl.Result{}, err
			}
		}
	} else {
		// The object is being deleted
		if containsString(company.GetFinalizers(), finalizerName) {
			// if err := r.deleteExternalResources(&company); err != nil {
			//     return ctrl.Result{}, err
			// }

			// remove our finalizer from the list and update it.
			controllerutil.RemoveFinalizer(&company, finalizerName)
			if err := r.Update(ctx, &company); err != nil {
				return ctrl.Result{}, err
			}
		}
		// Stop reconciliation as the item is being deleted
		return ctrl.Result{}, nil
	}

	return ctrl.Result{}, nil
}

// A helper function to check if a string is present in a slice of strings.
func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

func (r *CompanyReconciler) updateCompanyStatus(ctx context.Context, company *corev1alpha1.Company, log *logr.Logger) error {
	log.Info("Updating Company status")
	return r.Status().Update(ctx, company)
}

// SetupWithManager sets up the controller with the Manager.
func (r *CompanyReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1alpha1.Company{}).
		Named("company").
		Complete(r)
}
