package restart

import (
	"context"
	"fmt"
	"strings"
	"time"

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

//+kubebuilder:rbac:groups=core,resources=serviceaccounts,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=roles,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=rolebindings,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch;create;update;patch;delete

func (r *PodExpireReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	podExpireLogger.V(1).Info("Reconciling Pod", "req", req)

	pod := &corev1.Pod{}

	if err := r.Get(ctx, req.NamespacedName, pod); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	podExpireLogger.V(1).Info("Pod", "Name", pod.Name, "Namespace", pod.Namespace)

	return ctrl.Result{}, nil
}

func (r *PodExpireReconciler) EvictPod(ctx context.Context, pod *corev1.Pod) error {
	annotations := pod.GetAnnotations()
	var minTime *time.Time

	for key, value := range annotations {
		if strings.HasPrefix(key, "restarter.zncdata.dv/expires-at.") {
			t, err := time.Parse(time.RFC3339, value)
			if err != nil {
				return fmt.Errorf("failed to parse time: %w", err)
			}

			if minTime == nil || t.Before(*minTime) {
				minTime = &t
			}
		}
	}

	if minTime != nil && minTime.Before(time.Now()) {
		err := r.Client.SubResource("eviction").Create(ctx, pod, &policyv1.Eviction{})
		if err != nil {
			return fmt.Errorf("failed to evict pod: %w", err)
		}
	}

	return nil
}

func (r *PodExpireReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Pod{}).Named("podExpire").
		Complete(r)
}
