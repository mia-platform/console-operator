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
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	corev1alpha1 "github.com/mia-platform/console-operator/api/core/v1alpha1"
	consoleclient "github.com/mia-platform/console-operator/pkg/console-client"
)

// CompanyReconciler reconciles a Company object
type CompanyReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=core.mia-platform.eu,resources=companies,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core.mia-platform.eu,resources=consoles,verbs=get;list;watch
// +kubebuilder:rbac:groups=core.mia-platform.eu,resources=companies/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=core.mia-platform.eu,resources=companies/finalizers,verbs=update

// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch

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
			log.Info("Deleting the company from Console is not implemented yet, just deleting the k8s resource")
			r.Delete(ctx, &company)
			log.Info("Company deleted successfully from k8s", "companyName", company.Spec.Name)

			// remove our finalizer from the list and update it.
			controllerutil.RemoveFinalizer(&company, finalizerName)
			if err := r.Update(ctx, &company); err != nil {
				return ctrl.Result{}, err
			}
		}
		// Stop reconciliation as the item is being deleted
		return ctrl.Result{}, nil
	}

	companyName := company.Spec.Name
	companyConsoleRef := company.Spec.ConsoleRef
	var companyConsole corev1alpha1.Console
	if err := r.Get(ctx, types.NamespacedName{
		Name:      companyConsoleRef.Name,
		Namespace: companyConsoleRef.Namespace,
	}, &companyConsole); err != nil {
		log.Error(err, "unable to fetch Console for Company", "consoleRef", companyConsoleRef)
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	var clientIdSecret v1.Secret
	if err := r.Get(ctx, types.NamespacedName{Name: companyConsole.Spec.ClientID.SecretRef, Namespace: companyConsole.Namespace}, &clientIdSecret); err != nil {
		log.Error(err, "unable to fetch ClientID Secret for Company")
		return ctrl.Result{}, err
	}

	var clientKeySecret v1.Secret
	if err := r.Get(ctx, types.NamespacedName{Name: companyConsole.Spec.ClientSecret.SecretRef, Namespace: companyConsole.Namespace}, &clientKeySecret); err != nil {
		log.Error(err, "unable to fetch ClientKey Secret for Company")
		return ctrl.Result{}, err
	}

	consoleClientConfig := consoleclient.ClientConfig{
		BaseURL:      companyConsole.Spec.URL,
		ClientID:     string(clientIdSecret.Data[companyConsole.Spec.ClientID.KeyRef]),
		ClientSecret: string(clientKeySecret.Data[companyConsole.Spec.ClientSecret.KeyRef]),
	}

	consoleClient, err := consoleclient.NewClient(consoleClientConfig, ctx)
	if err != nil {
		log.Error(err, "failed to create Console client")
		return ctrl.Result{}, err
	}

	exists, companyId, err := consoleClient.CompanyExists(ctx, companyName)
	if err != nil {
		log.Error(err, "failed to check if company exists in Console")
		return ctrl.Result{}, err
	}

	if exists {
		log.Info("Company already exists", "companyName", companyName)
		company.Status.CompanyId = companyId
		company.Status.CompanyName = companyName
		company.Status.Exists = true
		if err := r.updateCompanyStatus(ctx, &company, &log); err != nil {
			log.Error(err, "failed to update Company status")
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	newCompany := consoleclient.ConsoleCompany{
		Name:        companyName,
		Description: company.Spec.Description,
	}

	if companyId, err = consoleClient.CreateCompany(ctx, newCompany); err != nil {
		log.Error(err, "failed to create company in Console", "companyName", companyName)
		return ctrl.Result{}, err
	}

	log.Info("Company created successfully in Console", "companyName", companyName, "companyId", companyId)

	if errors := consoleClient.AddCompanyOwners(ctx, company, companyId); len(errors) > 0 {
		for _, err := range errors {
			log.Error(err, "failed to add company owner in Console", "companyName", companyName)
		}
	}

	var clustersMap map[string]string = make(map[string]string)
	for _, cluster := range company.Spec.Clusters {
		var serviceAccountTokenSecret v1.Secret
		if err := r.Get(ctx, types.NamespacedName{Name: cluster.Connection.ServiceAccountToken.SecretRef, Namespace: company.Namespace}, &serviceAccountTokenSecret); err != nil {
			log.Error(err, "unable to fetch cluster service account token Secret for Company")
			continue
		}
		var serviceAccountToken = string(serviceAccountTokenSecret.Data[cluster.Connection.ServiceAccountToken.KeyRef])

		if clusterId, err := consoleClient.AddCompanyCluster(ctx, cluster, companyId, serviceAccountToken); err != nil {
			log.Error(err, "failed to add company cluster in Console", "companyName", companyName)
		} else {
			clustersMap[cluster.ClusterId] = clusterId
			log.Info("Cluster added successfully to Company in Console", "companyName", companyName, "clusterId", clusterId)
		}
	}
	log.Info("Clusters map: ", clustersMap)
	if addEnvironmentsErr := consoleClient.AddCompanyEnvironments(ctx, company.Spec.Environments, companyId, clustersMap); addEnvironmentsErr != nil {
		log.Error(addEnvironmentsErr, "failed to add company environments in Console", "companyName", companyName)
	}

	company.Status.CompanyId = companyId
	company.Status.CompanyName = companyName
	company.Status.Exists = true
	if err := r.updateCompanyStatus(ctx, &company, &log); err != nil {
		log.Error(err, "failed to update Company status")
		return ctrl.Result{}, err
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

	// Re-fetch the latest version of the company before updating status
	// to avoid conflicts
	latest := &corev1alpha1.Company{}
	if err := r.Get(ctx, client.ObjectKeyFromObject(company), latest); err != nil {
		return err
	}

	// Copy the status from the company we want to update
	latest.Status = company.Status

	return r.Status().Update(ctx, latest)
}

// SetupWithManager sets up the controller with the Manager.
func (r *CompanyReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1alpha1.Company{}).
		Named("company").
		Complete(r)
}
