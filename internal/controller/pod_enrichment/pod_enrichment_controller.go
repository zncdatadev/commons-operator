package pod_enrichment

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/util/retry"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

const (
	PodEnrichmentLabelName  = "enrichment.kubedoop.dev/enable"
	PodEnrichmentLabelValue = "true"

	PodenrichmentNodeAddressAnnotationName = "enrichment.kubedoop.dev/node-address"
)

var (
	OrderedAddressTypes = []corev1.NodeAddressType{
		corev1.NodeHostName,
		corev1.NodeExternalDNS,
		corev1.NodeInternalDNS,
		corev1.NodeExternalIP,
		corev1.NodeInternalIP,
	}
)

var (
	logger = ctrl.Log.WithName("controllers").WithName("PodEnrichment")
)

type PodEnrichmentReconciler struct {
	client.Client
	Schema *runtime.Scheme
}

// +kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch;update;patch
// +kubebuilder:rbac:groups=core,resources=nodes,verbs=get;list;watch

func (r *PodEnrichmentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	pod := &corev1.Pod{}
	if err := r.Get(ctx, req.NamespacedName, pod); err != nil {
		if apierrors.IsNotFound(err) {
			logger.V(5).Info("Pod not found", "pod", req.NamespacedName)
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
	} else if !result.IsZero() {
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

type PodHandler struct {
	Client client.Client
	Pod    *corev1.Pod
}

func NewPodHandler(client client.Client, pod *corev1.Pod) *PodHandler {
	return &PodHandler{
		Client: client,
		Pod:    pod,
	}
}

func (p *PodHandler) scheduled() bool {
	if p.Pod.Spec.NodeName == "" {
		logger.Info("Pod is not scheduled", "pod", p.Pod.Name, "namespace", p.Pod.Namespace)
		return false
	}
	return true
}

func (p *PodHandler) getNodeAddress(ctx context.Context) (string, error) {
	node := &corev1.Node{}

	if err := p.Client.Get(ctx, client.ObjectKey{Name: p.Pod.Spec.NodeName}, node); err != nil {
		return "", err
	}

	addressMap := make(map[corev1.NodeAddressType]string)

	for _, address := range node.Status.Addresses {
		addressMap[address.Type] = address.Address
	}

	for _, addressType := range OrderedAddressTypes {
		if address, ok := addressMap[addressType]; ok {
			return address, nil
		}
	}

	return "", fmt.Errorf("Node %s has no address", node.Name)
}

func (p *PodHandler) updateNodeAddrToPodMeta(ctx context.Context, nodeAddress string) error {
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		// get the latest version of the pod
		if err := p.Client.Get(ctx, client.ObjectKey{Namespace: p.Pod.Namespace, Name: p.Pod.Name}, p.Pod); err != nil {
			if apierrors.IsNotFound(err) {
				logger.Info("Pod not found, skipping", "pod", p.Pod.Name, "namespace", p.Pod.Namespace)
				return nil
			}
			return err
		}

		annotations := p.Pod.GetAnnotations()
		if annotations == nil {
			annotations = make(map[string]string)
		}

		annotations[PodenrichmentNodeAddressAnnotationName] = nodeAddress
		p.Pod.SetAnnotations(annotations)

		return p.Client.Update(ctx, p.Pod)
	})
}

func (p *PodHandler) Reconcile(ctx context.Context) (ctrl.Result, error) {
	if !p.scheduled() {
		return ctrl.Result{RequeueAfter: time.Second * 2}, nil
	}

	nodeAddress, err := p.getNodeAddress(ctx)
	if err != nil {
		return ctrl.Result{}, err
	}

	if err := p.updateNodeAddrToPodMeta(ctx, nodeAddress); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}
