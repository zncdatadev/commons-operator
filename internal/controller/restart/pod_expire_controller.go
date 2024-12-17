package restart

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/zncdatadev/operator-go/pkg/constants"
	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	podExpireLogger = ctrl.Log.WithName("controllers").WithName("PodExpire")
)

type PodExpireReconciler struct {
	client.Client
	Schema *runtime.Scheme
}

// +kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=pods/eviction,verbs=create

func (r *PodExpireReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	pod := &corev1.Pod{}

	if err := r.Get(ctx, req.NamespacedName, pod); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	annotations := pod.GetAnnotations()

	if annotations == nil {
		podExpireLogger.V(5).Info("No expiry annotations, skip it", "Pod", pod.Name, "Namespace", pod.Namespace)
		return ctrl.Result{}, nil
	}

	podExpireLogger.V(5).Info("Found expiry annotations", "Pod", pod.Name, "Namespace", pod.Namespace)

	var minTime *time.Time
	for key, value := range annotations {
		if strings.HasPrefix(key, constants.PrefixLabelRestarterExpiresAt) {
			t, err := time.Parse(time.RFC3339, value)
			if err != nil {
				return ctrl.Result{}, fmt.Errorf("failed to parse time: %w", err)
			}

			if minTime == nil || t.Before(*minTime) {
				minTime = &t
			}
		}
	}

	if minTime == nil || minTime.Before(time.Now()) {
		err := r.Client.SubResource("eviction").Create(ctx, pod, &policyv1.Eviction{})
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to evict pod: %w", err)
		}
		podExpireLogger.Info("Evict pod immediately", "Pod", pod.Name, "Namespace", pod.Namespace)
		return ctrl.Result{}, nil
	}

	delay_time := time.Until(*minTime)
	podExpireLogger.Info("Evict pod after delay", "Pod", pod.Name, "Namespace", pod.Namespace, "Delay", delay_time)

	return ctrl.Result{RequeueAfter: delay_time}, nil
}

func (r *PodExpireReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Pod{}).Named("podExpire").
		Complete(r)
}
