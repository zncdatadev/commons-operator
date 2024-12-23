package restart

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/zncdatadev/operator-go/pkg/constants"
	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	podExpiresLogger = ctrl.Log.WithName("controllers").WithName("PodExpires")
)

type PodExpiresReconciler struct {
	client.Client
	Schema *runtime.Scheme
}

// +kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=pods/eviction,verbs=create

func (r *PodExpiresReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	pod := &corev1.Pod{}

	if err := r.Get(ctx, req.NamespacedName, pod); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	podExpiresLogger.V(5).Info("Reconcile Pod", "Pod", pod.Name, "Namespace", pod.Namespace)

	if pod.DeletionTimestamp != nil {
		podExpiresLogger.V(5).Info("Pod is being deleted, skip it", "Pod", pod.Name, "Namespace", pod.Namespace)
		return ctrl.Result{}, nil
	}

	annotations := pod.GetAnnotations()

	if annotations == nil {
		podExpiresLogger.V(5).Info("No annotations, skip it", "Pod", pod.Name, "Namespace", pod.Namespace)
		return ctrl.Result{}, nil
	}
	var watched bool
	var minTime *time.Time
	for key, value := range annotations {
		if strings.HasPrefix(key, constants.PrefixLabelRestarterExpiresAt) {
			watched = true
			t, err := time.Parse(time.RFC3339, value)
			if err != nil {
				return ctrl.Result{}, fmt.Errorf("failed to parse time: %w", err)
			}

			if minTime == nil || t.Before(*minTime) {
				minTime = &t
			}
		}
	}

	if !watched {
		podExpiresLogger.V(5).Info("No expiry annotations, skip it", "Pod", pod.Name, "Namespace", pod.Namespace)
		return ctrl.Result{}, nil
	}

	podExpiresLogger.Info("Reconcile pod with expiry annotations", "Pod", pod.Name, "Namespace", pod.Namespace, "annotations", annotations)

	if minTime == nil {
		podExpiresLogger.V(1).Info("No expiry time, skip it", "Pod", pod.Name, "Namespace", pod.Namespace)
		return ctrl.Result{}, nil
	}

	if minTime.Before(time.Now()) {

		// naming eviction object with pod name
		eviction := &policyv1.Eviction{
			ObjectMeta: metav1.ObjectMeta{
				Name:      pod.Name,
				Namespace: pod.Namespace,
			},
		}

		err := r.Client.SubResource("eviction").Create(ctx, pod, eviction)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to evict pod: %w", err)
		}
		podExpiresLogger.Info("Evict pod immediately", "Pod", pod.Name, "Namespace", pod.Namespace)
		return ctrl.Result{}, nil
	}

	delay_time := time.Until(*minTime)
	podExpiresLogger.Info("Evict pod after delay", "Pod", pod.Name, "Namespace", pod.Namespace, "Delay", delay_time)

	return ctrl.Result{RequeueAfter: delay_time}, nil
}

func (r *PodExpiresReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Pod{}).Named("podExpires").
		Complete(r)
}
