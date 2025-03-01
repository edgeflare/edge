/*
Copyright 2025 edgeflare.io.

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

package helm

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"

	"helm.sh/helm/v3/pkg/release"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	helmv1alpha1 "github.com/edgeflare/edge/api/helm/v1alpha1"
	"github.com/edgeflare/edge/internal/common"
	"github.com/edgeflare/edge/internal/util/helm"
)

type ReleaseReconciler struct {
	client.Client
	Scheme     *runtime.Scheme
	HelmClient *helm.Client
}

const (
	finalizerName = "helm.edgeflare.io/finalizer"
)

// Reconcile handles the reconciliation loop for helm releases.
// +kubebuilder:rbac:groups=helm.edgeflare.io,resources=releases,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=helm.edgeflare.io,resources=releases/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=helm.edgeflare.io,resources=releases/finalizers,verbs=update
func (r *ReleaseReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("Starting reconciliation", "namespace", req.Namespace, "name", req.Name)

	// Get the Release resource
	release := &helmv1alpha1.Release{}
	if err := r.Get(ctx, req.NamespacedName, release); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Initialize HelmClient if not already done
	if r.HelmClient == nil {
		helmClient, err := helm.NewClient()
		if err != nil {
			return r.handleError(ctx, release, err)
		}
		r.HelmClient = helmClient
	}

	// Handle deletion if the resource is being deleted
	if !release.DeletionTimestamp.IsZero() {
		return r.handleDeletion(ctx, release)
	}

	// Add finalizer if not present
	if !controllerutil.ContainsFinalizer(release, finalizerName) {
		controllerutil.AddFinalizer(release, finalizerName)
		if err := r.Update(ctx, release); err != nil {
			return ctrl.Result{}, err
		}
		// Return here as the update will trigger another reconciliation
		return ctrl.Result{}, nil
	}

	// Skip reconciliation if no changes detected
	if !r.shouldReconcile(release) {
		logger.Info("No changes detected, skipping reconciliation")
		return ctrl.Result{}, nil
	}

	// Install/upgrade the helm release
	releaseSpec := helm.ReleaseSpec{
		Name:          release.Name,
		ChartURL:      release.Spec.ChartURL,
		Namespace:     release.Namespace,
		ValuesContent: release.Spec.ValuesContent,
	}

	releaseResult, err := r.HelmClient.Install(ctx, releaseSpec)
	if err != nil {
		return r.handleError(ctx, release, err)
	}

	// Update the release status after successful installation/upgrade
	if err := r.updateReleaseState(ctx, release, releaseResult); err != nil {
		logger.Error(err, "Failed to update release state")
		return ctrl.Result{}, err
	}

	logger.Info("Reconciliation completed successfully")
	return ctrl.Result{}, nil
}

// handleDeletion manages the deletion process for a helm release.
// This improved version ensures finalizers are properly removed even if uninstall fails.
func (r *ReleaseReconciler) handleDeletion(ctx context.Context, release *helmv1alpha1.Release) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("Handling deletion", "name", release.Name, "namespace", release.Namespace)

	// Only process if our finalizer is present
	if controllerutil.ContainsFinalizer(release, finalizerName) {
		// Try to verify if the release exists in Helm
		releases, err := r.HelmClient.ListReleases(ctx, release.Namespace)

		// If listing releases fails, log error but continue with finalizer removal
		if err != nil {
			logger.Error(err, "Failed to list Helm releases, proceeding with finalizer removal")
			// Update status to indicate error before removing finalizer
			release.Status.Conditions = []metav1.Condition{
				{
					Type:               common.ConditionTypeError,
					Status:             metav1.ConditionTrue,
					LastTransitionTime: metav1.Now(),
					Reason:             "UninstallError",
					Message:            fmt.Sprintf("Failed to list Helm releases: %v", err),
				},
			}
			if updateErr := r.Status().Update(ctx, release); updateErr != nil {
				logger.Error(updateErr, "Failed to update release status")
			}
		} else {
			// Check if release exists in Helm
			releaseExists := false
			for _, rel := range releases {
				if rel == release.Name {
					releaseExists = true
					break
				}
			}

			// Attempt uninstallation if release exists
			if releaseExists {
				if err := r.HelmClient.Uninstall(ctx, release.Name, release.Namespace); err != nil {
					logger.Error(err, "Failed to uninstall Helm release, proceeding with finalizer removal")
					// Update status to indicate uninstall error before removing finalizer
					release.Status.Conditions = []metav1.Condition{
						{
							Type:               common.ConditionTypeError,
							Status:             metav1.ConditionTrue,
							LastTransitionTime: metav1.Now(),
							Reason:             "UninstallError",
							Message:            fmt.Sprintf("Uninstall failed: %v", err),
						},
					}
					if updateErr := r.Status().Update(ctx, release); updateErr != nil {
						logger.Error(updateErr, "Failed to update release status")
					}
				} else {
					logger.Info("Successfully uninstalled Helm release")
				}
			} else {
				logger.Info("Helm release not found, skipping uninstallation")
			}
		}

		// Remove finalizer regardless of uninstall outcome to prevent stuck resources
		controllerutil.RemoveFinalizer(release, finalizerName)
		if err := r.Update(ctx, release); err != nil {
			logger.Error(err, "Failed to remove finalizer")
			return ctrl.Result{}, err
		}
		logger.Info("Successfully removed finalizer")
	}

	return ctrl.Result{}, nil
}

// shouldReconcile checks if reconciliation is needed based on changes to values or chart version.
func (r *ReleaseReconciler) shouldReconcile(release *helmv1alpha1.Release) bool {
	if release.Annotations == nil {
		return true
	}

	// Check for changes in values content
	currentHash := valuesHash(release.Spec.ValuesContent)
	lastHash := release.Annotations[common.AnnotationValuesHash]
	if lastHash != currentHash {
		return true
	}

	// Check for changes in chart version
	currentVersion := chartVersion(release.Spec.ChartURL)
	lastVersion := release.Annotations[common.AnnotationChartVersion]
	return lastVersion != currentVersion
}

// updateReleaseState updates the release CR with current state after install/upgrade.
func (r *ReleaseReconciler) updateReleaseState(ctx context.Context, release *helmv1alpha1.Release, releaseResult *release.Release) error {
	// Initialize annotations if nil
	if release.Annotations == nil {
		release.Annotations = make(map[string]string)
	}

	// Update annotations with current state
	release.Annotations[common.AnnotationValuesHash] = valuesHash(release.Spec.ValuesContent)
	release.Annotations[common.AnnotationRevision] = strconv.Itoa(releaseResult.Version)
	release.Annotations[common.AnnotationChartVersion] = chartVersion(release.Spec.ChartURL)

	// Initialize labels if nil
	if release.Labels == nil {
		release.Labels = make(map[string]string)
	}

	// Update labels
	release.Labels[common.LabelVersion] = releaseResult.Chart.AppVersion()
	release.Labels[common.LabelManagedBy] = "edge"

	// Update the resource
	if err := r.Update(ctx, release); err != nil {
		return err
	}

	// Update status fields
	release.Status.Conditions = []metav1.Condition{
		{
			Type:               common.ConditionTypeInstalled,
			Status:             metav1.ConditionTrue,
			LastTransitionTime: metav1.Now(),
			Reason:             "InstallationSucceeded",
			Message:            "Helm release installed/upgraded successfully",
		},
	}

	release.Status.HelmStatus = releaseResult.Info.Status
	release.Status.FirstDeployed = releaseResult.Info.FirstDeployed.String()
	release.Status.LastDeployed = releaseResult.Info.LastDeployed.String()
	release.Status.Deleted = releaseResult.Info.Deleted.String()

	return r.Status().Update(ctx, release)
}

// handleError updates the release status with error information.
func (r *ReleaseReconciler) handleError(ctx context.Context, release *helmv1alpha1.Release, err error) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Error(err, "Reconciliation failed")

	// Update status with error information
	release.Status.Conditions = []metav1.Condition{
		{
			Type:               common.ConditionTypeError,
			Status:             metav1.ConditionTrue,
			LastTransitionTime: metav1.Now(),
			Reason:             "InstallationFailed",
			Message:            err.Error(),
		},
	}

	if updateErr := r.Status().Update(ctx, release); updateErr != nil {
		logger.Error(updateErr, "Failed to update error status")
		return ctrl.Result{}, updateErr
	}

	return ctrl.Result{}, err
}

// SetupWithManager sets up the controller with the Manager.
func (r *ReleaseReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&helmv1alpha1.Release{}).
		Named("helm-release").
		Complete(r)
}

// valuesHash generates a hash of the values content for change detection.
func valuesHash(values string) string {
	hash := sha256.Sum256([]byte(values))
	return hex.EncodeToString(hash[:])
}

// chartVersion extracts the version from a chart URL.
func chartVersion(chartURL string) string {
	parts := strings.Split(chartURL, ":")
	if len(parts) > 1 {
		return parts[len(parts)-1]
	}
	return ""
}
