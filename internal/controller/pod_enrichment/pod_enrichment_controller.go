package pod_enrichment

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

const (
	PodEnrichmentLabelName  = "enrichment.kubedoop.dev/enable"
	PodEnrichmentLabelValue = "true"
)

var (
	logger = ctrl.Log.WithName("controllers").WithName("PodEnrichment")
)

type PodEnrichmentReconciler struct {
	client.Client
	Schema *runtime.Scheme
}

// +kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch;update;patch

func (r *PodEnrichmentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	logger.Info("Reconciling PodEnrichment")

	pod := &corev1.Pod{}
	if err := r.Get(ctx, req.NamespacedName, pod); err != nil {
		if apierrors.IsNotFound(err) {
			logger.Info("Pod not found", "pod", req.NamespacedName)
			return ctrl.Result{}, nil
		}
		logger.Error(err, "Failed to get Pod", "pod", req.NamespacedName)
		return ctrl.Result{}, err
	}

	logger.Info("Pod found", "name", pod.Name, "namespace", pod.Namespace)

	handler := NewPodHandler(r.Client, pod)

	if result, err := handler.Reconcile(ctx); err != nil {
		logger.Error(err, "Failed to reconcile Pod", "pod", req.NamespacedName)
		return result, err
	} else if result.Requeue || result.RequeueAfter > 0 {
		return result, nil
	}

	logger.Info("Pod reconciled", "pod", req.NamespacedName)

	return ctrl.Result{}, nil
}

func (r *PodEnrichmentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	pred, err := predicate.LabelSelectorPredicate(metav1.LabelSelector{
		MatchLabels: map[string]string{
			PodEnrichmentLabelName: PodEnrichmentLabelValue,
		},
	})
	if err != nil {
		return err
	}
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Pod{}).Named("podEnrichment").
		WithEventFilter(pred).
		Complete(r)
}
