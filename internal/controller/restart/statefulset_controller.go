package restart

import (
	"context"

	appv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var (
	restartLogger = ctrl.Log.WithName("controllers").WithName("StatefulSetRestart")
)

const (
	RestartertLabelName  = "restarter.kubedoop.dev/enable"
	RestartertLabelValue = "true"

	ConfigMapRestartAnnotationPrefix = "configmap.restarter.kubedoop.dev/"
	SecretRestartAnnotationPrefix    = "secret.restarter.kubedoop.dev/"
)

type StatefulSetReconciler struct {
	client.Client

	Schema *runtime.Scheme
}

// +kubebuilder:rbac:groups=apps,resources=statefulsets,verbs=get;list;watch;patch
// +kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;watch
// +kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch
// +kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch;update;patch

func (r *StatefulSetReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	sts := &appv1.StatefulSet{}

	if err := r.Get(ctx, req.NamespacedName, sts); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	restartLogger.Info("StatefulSet", "Name", sts.Name, "Namespace", sts.Namespace)

	h := &StatefulSetHandler{
		Client: r.Client,
		Sts:    sts.DeepCopy(),
	}

	err := h.updateRef(ctx)
	if err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *StatefulSetReconciler) SetupWithManager(mgr ctrl.Manager) error {
	labelPredicate := predicate.NewTypedPredicateFuncs(func(obj client.Object) bool {
		return obj.GetLabels()[RestartertLabelName] == RestartertLabelValue
	})
	return ctrl.NewControllerManagedBy(mgr).
		For(&appv1.StatefulSet{}, builder.WithPredicates(labelPredicate)).
		Watches(&corev1.ConfigMap{}, handler.EnqueueRequestsFromMapFunc(r.findAffectedStatefulSets)).
		Watches(&corev1.Secret{}, handler.EnqueueRequestsFromMapFunc(r.findAffectedStatefulSets)).
		Complete(r)
}

func (r *StatefulSetReconciler) findAffectedStatefulSets(ctx context.Context, obj client.Object) []reconcile.Request {
	list := &appv1.StatefulSetList{}
	if err := r.List(ctx, list, client.MatchingLabels{RestartertLabelName: RestartertLabelValue}); err != nil {
		restartLogger.Error(err, "Failed to list StatefulSets")
		return nil
	}

	requests := make([]reconcile.Request, 0, len(list.Items))
	for _, sts := range list.Items {
		requests = append(requests, reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      sts.Name,
				Namespace: sts.Namespace,
			},
		})
	}
	return requests
}

type StatefulSetHandler struct {
	Client client.Client
	Sts    *appv1.StatefulSet
}

func (h *StatefulSetHandler) getRefSecretRefs() []string {
	secrets := make([]string, 0)
	for _, volume := range h.Sts.Spec.Template.Spec.Volumes {
		if volume.Secret != nil {
			secrets = append(secrets, volume.Secret.SecretName)
		}
	}
	for _, container := range h.Sts.Spec.Template.Spec.InitContainers {
		for _, env := range container.Env {
			if env.ValueFrom != nil && env.ValueFrom.SecretKeyRef != nil {
				secrets = append(secrets, env.ValueFrom.SecretKeyRef.Name)
			}
		}
	}
	for _, container := range h.Sts.Spec.Template.Spec.Containers {
		for _, env := range container.Env {
			if env.ValueFrom != nil && env.ValueFrom.SecretKeyRef != nil {
				secrets = append(secrets, env.ValueFrom.SecretKeyRef.Name)
			}
		}
	}
	return secrets
}

func (h *StatefulSetHandler) getRefConfigMapRefs() []string {
	configMaps := make([]string, 0)
	for _, volume := range h.Sts.Spec.Template.Spec.Volumes {
		if volume.ConfigMap != nil {
			return append(configMaps, volume.ConfigMap.Name)
		}
	}

	for _, container := range h.Sts.Spec.Template.Spec.InitContainers {
		for _, env := range container.Env {
			if env.ValueFrom != nil && env.ValueFrom.ConfigMapKeyRef != nil {
				return append(configMaps, env.ValueFrom.ConfigMapKeyRef.Name)
			}
		}
	}

	for _, container := range h.Sts.Spec.Template.Spec.Containers {
		for _, env := range container.Env {
			if env.ValueFrom != nil && env.ValueFrom.ConfigMapKeyRef != nil {
				return append(configMaps, env.ValueFrom.ConfigMapKeyRef.Name)
			}
		}
	}

	return configMaps
}

func (h *StatefulSetHandler) getSecret(ctx context.Context, name, namespace string) (*corev1.Secret, error) {
	obj := &corev1.Secret{}
	err := h.Client.Get(ctx, types.NamespacedName{Namespace: namespace, Name: name}, obj)
	if err != nil {
		return nil, err
	}
	return obj, nil
}

func (h *StatefulSetHandler) getConfigMap(ctx context.Context, name, namespace string) (*corev1.ConfigMap, error) {
	obj := &corev1.ConfigMap{}
	err := h.Client.Get(ctx, types.NamespacedName{Namespace: namespace, Name: name}, obj)
	if err != nil {
		return nil, err
	}
	return obj, nil
}

func (h *StatefulSetHandler) updateRef(ctx context.Context) error {
	annotations := make(map[string]string)

	configMapRefs := h.getRefConfigMapRefs()
	for _, configMap := range configMapRefs {
		obj, err := h.getConfigMap(ctx, configMap, h.Sts.Namespace)
		if err != nil {
			return err
		}
		annotationName := GenStatefulSetRestartAnnotationKey(obj)
		annotationValue := GenStatefulSetRestartAnnotationValue(obj)
		annotations[annotationName] = annotationValue
	}

	secretRefs := h.getRefSecretRefs()
	for _, secret := range secretRefs {
		obj, err := h.getSecret(ctx, secret, h.Sts.Namespace)
		if err != nil {
			return err
		}
		annotationName := GenStatefulSetRestartAnnotationKey(obj)
		annotations[annotationName] = GenStatefulSetRestartAnnotationValue(obj)
	}

	if len(annotations) == 0 {
		restartLogger.V(5).Info("No ConfigMap or Secret references found, skip it", "StatefulSet", h.Sts.Name, "Namespace", h.Sts.Namespace)
		return nil
	}

	patch := client.MergeFrom(h.Sts.DeepCopy())
	if h.Sts.Annotations == nil {
		h.Sts.Annotations = make(map[string]string)
	}

	for k, v := range annotations {
		h.Sts.Annotations[k] = v
	}

	restartLogger.Info("Update StatefulSet annotations", "Name", h.Sts.Name, "Namespace", h.Sts.Namespace, "Annotations", annotations)
	return h.Client.Patch(ctx, h.Sts, patch)
}

func GenStatefulSetRestartAnnotationValue(obj client.Object) string {
	return string(obj.GetUID()) + "/" + obj.GetResourceVersion()
}

func GenStatefulSetRestartAnnotationKey(obj client.Object) string {
	switch obj := obj.(type) {
	case *corev1.ConfigMap:
		return ConfigMapRestartAnnotationPrefix + obj.Name
	case *corev1.Secret:
		return SecretRestartAnnotationPrefix + obj.Name
	default:
		panic("unsupported object type for restart annotation")
	}
}
