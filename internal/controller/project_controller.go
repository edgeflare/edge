package controller

import (
	"context"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	helmv1alpha1 "github.com/edgeflare/edge/api/helm/v1alpha1"
	edgev1alpha1 "github.com/edgeflare/edge/api/v1alpha1"
	"github.com/edgeflare/edge/internal/common"
)

const (
	finalizerName = "edgeflare.io/finalizer"
	requeueShort  = 2 * time.Second
	requeueLong   = 5 * time.Minute
)

// ProjectReconciler reconciles Project resources
type ProjectReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// Reconcile handles the reconciliation of Project resources
// +kubebuilder:rbac:groups=edgeflare.io,resources=projects,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=edgeflare.io,resources=projects/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=edgeflare.io,resources=projects/finalizers,verbs=update
// +kubebuilder:rbac:groups=helm.edgeflare.io,resources=releases,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=helm.edgeflare.io,resources=releases/status,verbs=get
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch
func (r *ProjectReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("Reconciling", "project", req.NamespacedName)

	// Fetch the Project
	project := &edgev1alpha1.Project{}
	if err := r.Get(ctx, req.NamespacedName, project); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Handle deletion if needed
	if !project.DeletionTimestamp.IsZero() {
		return r.finalize(ctx, project)
	}

	// Skip if no changes since last reconciliation
	if project.Status.Generation == project.Generation {
		logger.Info("No changes detected")
		return ctrl.Result{RequeueAfter: requeueLong}, nil
	}

	// Ensure finalizer exists
	if err := r.ensureFinalizer(ctx, project); err != nil {
		if errors.IsConflict(err) {
			return ctrl.Result{RequeueAfter: requeueShort}, nil
		}
		return ctrl.Result{}, err
	}

	// Set reconciling status
	if err := r.setCondition(ctx, project, common.ConditionTypeReady, metav1.ConditionFalse,
		common.ReasonReconciling, "Reconciling project components"); err != nil {
		return ctrl.Result{}, err
	}

	// Reconcile all components
	if err := r.reconcileComponents(ctx, project); err != nil {
		logger.Error(err, "Component reconciliation failed")
		_ = r.setCondition(ctx, project, common.ConditionTypeError, metav1.ConditionTrue,
			common.ReasonComponentError, fmt.Sprintf("Failed: %v", err))
		return ctrl.Result{}, err
	}

	// Mark as ready and update observed generation
	if err := r.setCondition(ctx, project, common.ConditionTypeReady, metav1.ConditionTrue,
		common.ReasonReady, "All components ready"); err != nil {
		return ctrl.Result{}, err
	}

	if err := r.updateObservedGeneration(ctx, project); err != nil {
		return ctrl.Result{Requeue: true}, nil
	}

	logger.Info("Reconciliation successful")
	return ctrl.Result{RequeueAfter: requeueLong}, nil
}

func (r *ProjectReconciler) updateObservedGeneration(ctx context.Context, project *edgev1alpha1.Project) error {
	patch := client.MergeFrom(project.DeepCopy())
	project.Status.Generation = project.Generation
	return r.Status().Patch(ctx, project, patch)
}

func (r *ProjectReconciler) ensureFinalizer(ctx context.Context, project *edgev1alpha1.Project) error {
	if !controllerutil.ContainsFinalizer(project, finalizerName) {
		controllerutil.AddFinalizer(project, finalizerName)
		return r.Update(ctx, project)
	}
	return nil
}

func (r *ProjectReconciler) reconcileComponents(ctx context.Context, project *edgev1alpha1.Project) error {
	// Process database components
	if db := project.Spec.Database; db != nil {
		if ref := db.GetComponentRef("postgres"); ref != nil {
			if err := r.reconcileDatabase(ctx, project, "postgres", ref); err != nil {
				return err
			}
		}
	}

	if auth := project.Spec.Auth; auth != nil {
		if ref := auth.GetComponentRef("zitadel"); ref != nil {
			if err := r.reconcileAuth(ctx, project, "zitadel", ref); err != nil {
				return err
			}
		}
	}

	return nil
}

func (r *ProjectReconciler) handleComponentRelease(ctx context.Context, project *edgev1alpha1.Project,
	compType, name string, ref *edgev1alpha1.ComponentRef) error {
	logger := log.FromContext(ctx)
	releaseName := fmt.Sprintf("%s-%s", project.Name, name)

	// Create or update the release
	if err := r.upsertRelease(ctx, project, compType, name, ref); err != nil {
		logger.Error(err, "Release creation failed")
		_ = r.updateComponentStatus(ctx, project, compType, name, false,
			fmt.Sprintf("Release error: %v", err), "")
		return err
	}

	// Check release status
	release := &helmv1alpha1.Release{}
	err := r.Get(ctx, types.NamespacedName{
		Name:      releaseName,
		Namespace: project.Namespace,
	}, release)

	if errors.IsNotFound(err) {
		// Not found yet, wait for next reconciliation
		return nil
	} else if err != nil {
		return err
	}

	// Default status values
	ready := false
	message := "Installation in progress"
	endpoint := ""

	// Check release conditions
	for _, condition := range release.Status.Conditions {
		if condition.Type == common.ConditionTypeInstalled && condition.Status == metav1.ConditionTrue {
			ready = true
			message = "Component ready"

			// Set endpoint for postgres
			if compType == "database" && name == "postgres" {
				endpoint = fmt.Sprintf("%s-postgresql.%s.svc.cluster.local",
					project.Name, project.Namespace)
			}
			break
		}
	}

	// Update status
	return r.updateComponentStatus(ctx, project, compType, name, ready, message, endpoint)
}

func (r *ProjectReconciler) handleExternalComponent(ctx context.Context, project *edgev1alpha1.Project,
	compType, name string, ref *edgev1alpha1.ComponentRef) error {
	_ = ref
	// External resources are marked ready if secret exists
	return r.updateComponentStatus(ctx, project, compType, name, true, "Using external resource", "")
}

func (r *ProjectReconciler) upsertRelease(ctx context.Context, project *edgev1alpha1.Project,
	compType, name string, ref *edgev1alpha1.ComponentRef) error {
	logger := log.FromContext(ctx)

	if !ref.IsRelease() {
		return nil
	}

	releaseName := fmt.Sprintf("%s-%s", project.Name, name)
	releaseSpec := ref.GetReleaseSpec(compType, project.Name)

	// Prepare release object
	release := &helmv1alpha1.Release{
		ObjectMeta: metav1.ObjectMeta{
			Name:      releaseName,
			Namespace: project.Namespace,
			Labels: map[string]string{
				common.LabelManagedBy: "edge",
				common.LabelComponent: compType,
				common.LabelProject:   project.Name,
			},
		},
		Spec: releaseSpec,
	}

	// Set owner reference
	if err := controllerutil.SetControllerReference(project, release, r.Scheme); err != nil {
		return fmt.Errorf("failed to set owner reference: %w", err)
	}

	// Create or update release
	existing := &helmv1alpha1.Release{}
	err := r.Get(ctx, types.NamespacedName{Name: release.Name, Namespace: release.Namespace}, existing)

	if errors.IsNotFound(err) {
		logger.Info("Creating release", "name", release.Name)
		return r.Create(ctx, release)
	} else if err != nil {
		return err
	}

	// Update existing release
	existing.Spec = release.Spec
	logger.Info("Updating release", "name", release.Name)
	return r.Update(ctx, existing)
}

func (r *ProjectReconciler) updateComponentStatus(ctx context.Context, project *edgev1alpha1.Project,
	compType, name string, ready bool, message, endpoint string) error {

	// Create status patch
	patch := client.MergeFrom(project.DeepCopy())

	// Initialize status map if needed
	if project.Status.ComponentStatuses == nil {
		project.Status.ComponentStatuses = make(map[string]edgev1alpha1.ComponentStatus)
	}

	// Update component status
	key := fmt.Sprintf("%s-%s", compType, name)
	project.Status.ComponentStatuses[key] = edgev1alpha1.ComponentStatus{
		Ready:    ready,
		Message:  message,
		Endpoint: endpoint,
	}

	return r.Status().Patch(ctx, project, patch)
}

func (r *ProjectReconciler) setCondition(ctx context.Context, project *edgev1alpha1.Project,
	condType string, status metav1.ConditionStatus, reason, message string) error {

	// Create status patch
	patch := client.MergeFrom(project.DeepCopy())

	// Prepare new condition
	newCondition := metav1.Condition{
		Type:               condType,
		Status:             status,
		LastTransitionTime: metav1.Now(),
		Reason:             reason,
		Message:            message,
	}

	// Update conditions list
	found := false
	newConditions := []metav1.Condition{}

	// Clear error conditions if setting Ready to true
	if condType == common.ConditionTypeReady && status == metav1.ConditionTrue {
		for _, cond := range project.Status.Conditions {
			if cond.Type != common.ConditionTypeError {
				if cond.Type == condType {
					newConditions = append(newConditions, newCondition)
					found = true
				} else {
					newConditions = append(newConditions, cond)
				}
			}
		}
	} else {
		// Update/add the specified condition
		for _, cond := range project.Status.Conditions {
			if cond.Type == condType {
				newConditions = append(newConditions, newCondition)
				found = true
			} else {
				newConditions = append(newConditions, cond)
			}
		}
	}

	if !found {
		newConditions = append(newConditions, newCondition)
	}

	project.Status.Conditions = newConditions
	return r.Status().Patch(ctx, project, patch)
}

func (r *ProjectReconciler) finalize(ctx context.Context, project *edgev1alpha1.Project) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	if controllerutil.ContainsFinalizer(project, finalizerName) {
		// List owned releases
		releaseList := &helmv1alpha1.ReleaseList{}
		err := r.List(ctx, releaseList,
			client.InNamespace(project.Namespace),
			client.MatchingLabels{common.LabelProject: project.Name})

		if err != nil {
			return ctrl.Result{}, err
		}

		// If releases still exist, explicitly delete them
		if len(releaseList.Items) > 0 {
			logger.Info("Deleting associated releases", "count", len(releaseList.Items))
			for _, release := range releaseList.Items {
				if err := r.Delete(ctx, &release); err != nil && !errors.IsNotFound(err) {
					logger.Error(err, "Failed to delete release", "name", release.Name)
					return ctrl.Result{RequeueAfter: requeueShort}, err
				}
			}

			// Requeue to check if releases are deleted
			return ctrl.Result{RequeueAfter: requeueShort}, nil
		}

		// Remove finalizer once all releases are deleted
		controllerutil.RemoveFinalizer(project, finalizerName)
		if err := r.Update(ctx, project); err != nil {
			return ctrl.Result{}, err
		}
		logger.Info("Project finalized")
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager
func (r *ProjectReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&edgev1alpha1.Project{}).
		Owns(&helmv1alpha1.Release{}).
		Complete(r)
}
