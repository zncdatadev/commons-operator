package restart

import (
	"context"

	appv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	RestartertLabelName  = "restarter.zncdata.dev/enable"
	RestartertLabelValue = "true"
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

	restartLogger.V(1).Info("Reconciling StatefulSet", "req", req)

	sts := &appv1.StatefulSet{}

	if err := r.Get(ctx, req.NamespacedName, sts); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	restartLogger.Info("StatefulSet", "Name", sts.Name, "Namespace", sts.Namespace)

	h := &StatefulSetHandler{
		Client: r.Client,
		Sts:    sts.DeepCopy(),
	}

	err := h.UpdateRef(ctx)
	if err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *StatefulSetReconciler) SetupWithManager(mgr ctrl.Manager) error {

	pred, err := predicate.LabelSelectorPredicate(metav1.LabelSelector{
		MatchLabels: map[string]string{
			RestartertLabelName: RestartertLabelValue,
		},
	})
	if err != nil {
		return err
	}

	mapFunc := handler.MapFunc(func(ctx context.Context, a client.Object) []reconcile.Request {
		list := &appv1.StatefulSetList{}
		err := r.List(ctx, list, client.MatchingLabels{RestartertLabelName: RestartertLabelValue})
		if err != nil {
			restartLogger.Error(err, "Failed to list StatefulSets")
		}

		var requests []reconcile.Request
		for _, item := range list.Items {
			requests = append(requests, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Namespace: item.Namespace,
					Name:      item.Name,
				},
			})
		}
		return requests
	})

	return ctrl.NewControllerManagedBy(mgr).
		For(&appv1.StatefulSet{}, builder.WithPredicates(pred)).
		Watches(&corev1.ConfigMap{}, handler.EnqueueRequestsFromMapFunc(mapFunc)).
		Watches(&corev1.Secret{}, handler.EnqueueRequestsFromMapFunc(mapFunc)).
		Complete(r)
}

type StatefulSetHandler struct {
	Client client.Client
	Sts    *appv1.StatefulSet
}

func (h *StatefulSetHandler) GetRefSecrets(ctx context.Context) ([]corev1.Secret, error) {
	secrets := []corev1.Secret{}
	for _, volume := range h.Sts.Spec.Template.Spec.Volumes {
		if volume.Secret != nil {
			secret, err := h.GetSecret(ctx, volume.Secret.SecretName, h.Sts.Namespace)
			if err != nil {
				return nil, err
			}
			secrets = append(secrets, *secret)
		}
	}
	for _, container := range h.Sts.Spec.Template.Spec.InitContainers {
		for _, env := range container.Env {
			if env.ValueFrom != nil && env.ValueFrom.SecretKeyRef != nil {
				secret, err := h.GetSecret(ctx, env.ValueFrom.SecretKeyRef.Name, h.Sts.Namespace)
				if err != nil {
					return nil, err
				}
				secrets = append(secrets, *secret)
			}
		}
	}
	for _, container := range h.Sts.Spec.Template.Spec.Containers {
		for _, env := range container.Env {
			if env.ValueFrom != nil && env.ValueFrom.SecretKeyRef != nil {
				secret, err := h.GetSecret(ctx, env.ValueFrom.SecretKeyRef.Name, h.Sts.Namespace)
				if err != nil {
					return nil, err
				}
				secrets = append(secrets, *secret)
			}
		}
	}
	return secrets, nil
}

func (h *StatefulSetHandler) GetSecret(ctx context.Context, name, namespace string) (*corev1.Secret, error) {
	obj := &corev1.Secret{}
	err := h.Client.Get(ctx, types.NamespacedName{Namespace: namespace, Name: name}, obj)
	if err != nil {
		return nil, err
	}
	return obj, nil
}

func (h *StatefulSetHandler) GetConfigMap(ctx context.Context, name, namespace string) (*corev1.ConfigMap, error) {
	obj := &corev1.ConfigMap{}
	err := h.Client.Get(ctx, types.NamespacedName{Namespace: namespace, Name: name}, obj)
	if err != nil {
		return nil, err
	}
	return obj, nil
}

func (h *StatefulSetHandler) GetRefConfigMaps(ctx context.Context) ([]corev1.ConfigMap, error) {
	configMaps := []corev1.ConfigMap{}
	for _, volume := range h.Sts.Spec.Template.Spec.Volumes {
		if volume.ConfigMap != nil {
			configMap, err := h.GetConfigMap(ctx, volume.ConfigMap.Name, h.Sts.Namespace)
			if err != nil {
				return nil, err
			}
			configMaps = append(configMaps, *configMap)
		}
	}

	for _, container := range h.Sts.Spec.Template.Spec.InitContainers {
		for _, env := range container.Env {
			if env.ValueFrom != nil && env.ValueFrom.ConfigMapKeyRef != nil {
				configMap, err := h.GetConfigMap(ctx, env.ValueFrom.ConfigMapKeyRef.Name, h.Sts.Namespace)
				if err != nil {
					return nil, err
				}
				configMaps = append(configMaps, *configMap)
			}
		}
	}

	for _, container := range h.Sts.Spec.Template.Spec.Containers {
		for _, env := range container.Env {
			if env.ValueFrom != nil && env.ValueFrom.ConfigMapKeyRef != nil {
				configMap, err := h.GetConfigMap(ctx, env.ValueFrom.ConfigMapKeyRef.Name, h.Sts.Namespace)
				if err != nil {
					return nil, err
				}
				configMaps = append(configMaps, *configMap)
			}
		}
	}

	return configMaps, nil
}

func (h *StatefulSetHandler) UpdateRef(ctx context.Context) error {
	configMaps, err := h.GetRefConfigMaps(ctx)
	if err != nil {
		return err
	}
	annotations := h.Sts.Spec.Template.GetAnnotations()

	if annotations == nil {
		annotations = make(map[string]string)
	}

	for _, configMap := range configMaps {
		annotationName := "configmap.restarter.zncdata.dev/" + configMap.Name
		annotationValue := string(configMap.UID) + "/" + configMap.ResourceVersion
		annotations[annotationName] = annotationValue
	}

	secrets, err := h.GetRefSecrets(ctx)
	if err != nil {
		return err
	}

	for _, secret := range secrets {
		annotationName := "secret.restarter.zncdata.dev/" + secret.Name
		annotationValue := string(secret.UID) + "/" + secret.ResourceVersion
		annotations[annotationName] = annotationValue
	}

	h.Sts.Spec.Template.SetAnnotations(annotations)

	err = h.Client.Update(ctx, h.Sts)

	if err != nil {
		return err
	}

	return nil
}
