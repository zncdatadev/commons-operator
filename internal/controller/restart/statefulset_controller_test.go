package restart_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/zncdatadev/commons-operator/internal/controller/restart"
)

var _ = Describe("StatefulsetController", func() {
	var (
		ctx    context.Context
		scheme *runtime.Scheme
		c      client.Client
		r      *restart.StatefulSetReconciler
	)

	BeforeEach(func() {
		ctx = context.Background()
		scheme = runtime.NewScheme()
		Expect(appv1.AddToScheme(scheme)).To(Succeed())
		Expect(corev1.AddToScheme(scheme)).To(Succeed())
		c = fake.NewClientBuilder().WithScheme(scheme).Build()
		r = &restart.StatefulSetReconciler{
			Client: c,
			Schema: scheme,
		}
	})

	Context("Reconcile", func() {
		It("should ignore when statefulset not found", func() {
			req := types.NamespacedName{
				Name:      "test-sts",
				Namespace: "default",
			}
			result, err := r.Reconcile(ctx, ctrl.Request{NamespacedName: req})
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(ctrl.Result{}))
		})

		It("should skip when statefulset has no configmap/secret refs", func() {
			sts := &appv1.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-sts",
					Namespace: "default",
					Labels: map[string]string{
						restart.RestartertLabelName: restart.RestartertLabelValue,
					},
				},
				Spec: appv1.StatefulSetSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name:  "test",
									Image: "nginx",
								},
							},
						},
					},
				},
			}
			Expect(c.Create(ctx, sts)).To(Succeed())

			req := types.NamespacedName{
				Name:      "test-sts",
				Namespace: "default",
			}
			result, err := r.Reconcile(ctx, ctrl.Request{NamespacedName: req})
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(ctrl.Result{}))
		})

		It("should update annotations when statefulset has configmap refs", func() {
			cm := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-cm",
					Namespace: "default",
				},
				Data: map[string]string{"key": "value"},
			}
			Expect(c.Create(ctx, cm)).To(Succeed())

			sts := &appv1.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-sts",
					Namespace: "default",
					Labels: map[string]string{
						restart.RestartertLabelName: restart.RestartertLabelValue,
					},
				},
				Spec: appv1.StatefulSetSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name:  "test",
									Image: "nginx",
									Env: []corev1.EnvVar{
										{
											Name: "TEST",
											ValueFrom: &corev1.EnvVarSource{
												ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
													LocalObjectReference: corev1.LocalObjectReference{
														Name: "test-cm",
													},
													Key: "key",
												},
											},
										},
									},
								},
							},
						},
					},
				},
			}
			Expect(c.Create(ctx, sts)).To(Succeed())

			req := types.NamespacedName{
				Name:      "test-sts",
				Namespace: "default",
			}
			result, err := r.Reconcile(ctx, ctrl.Request{NamespacedName: req})
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(ctrl.Result{}))

			updatedSts := &appv1.StatefulSet{}
			Expect(c.Get(ctx, req, updatedSts)).To(Succeed())
			Expect(updatedSts.Annotations).To(HaveKey("configmap.restarter.kubedoop.dev/test-cm"))
		})

		It("should update annotations when statefulset has secret refs", func() {
			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-secret",
					Namespace: "default",
				},
				Data: map[string][]byte{"key": []byte("value")},
			}
			Expect(c.Create(ctx, secret)).To(Succeed())

			sts := &appv1.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-sts",
					Namespace: "default",
					Labels: map[string]string{
						restart.RestartertLabelName: restart.RestartertLabelValue,
					},
				},
				Spec: appv1.StatefulSetSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name:  "test",
									Image: "nginx",
									Env: []corev1.EnvVar{
										{
											Name: "TEST",
											ValueFrom: &corev1.EnvVarSource{
												SecretKeyRef: &corev1.SecretKeySelector{
													LocalObjectReference: corev1.LocalObjectReference{
														Name: "test-secret",
													},
													Key: "key",
												},
											},
										},
									},
								},
							},
						},
					},
				},
			}
			Expect(c.Create(ctx, sts)).To(Succeed())

			req := types.NamespacedName{
				Name:      "test-sts",
				Namespace: "default",
			}
			result, err := r.Reconcile(ctx, ctrl.Request{NamespacedName: req})
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(ctrl.Result{}))

			updatedSts := &appv1.StatefulSet{}
			Expect(c.Get(ctx, req, updatedSts)).To(Succeed())
			Expect(updatedSts.Annotations).To(HaveKey("secret.restarter.kubedoop.dev/test-secret"))
		})

		It("should update annotations when statefulset has configmap volume mount", func() {
			cm := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-cm-volume",
					Namespace: "default",
				},
				Data: map[string]string{"key": "value"},
			}
			Expect(c.Create(ctx, cm)).To(Succeed())

			sts := &appv1.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-sts",
					Namespace: "default",
					Labels: map[string]string{
						restart.RestartertLabelName: restart.RestartertLabelValue,
					},
				},
				Spec: appv1.StatefulSetSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Volumes: []corev1.Volume{
								{
									Name: "config-volume",
									VolumeSource: corev1.VolumeSource{
										ConfigMap: &corev1.ConfigMapVolumeSource{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: "test-cm-volume",
											},
										},
									},
								},
							},
							Containers: []corev1.Container{
								{
									Name:  "test",
									Image: "nginx",
									VolumeMounts: []corev1.VolumeMount{
										{
											Name:      "config-volume",
											MountPath: "/etc/config",
										},
									},
								},
							},
						},
					},
				},
			}
			Expect(c.Create(ctx, sts)).To(Succeed())

			req := types.NamespacedName{
				Name:      "test-sts",
				Namespace: "default",
			}
			result, err := r.Reconcile(ctx, ctrl.Request{NamespacedName: req})
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(ctrl.Result{}))

			updatedSts := &appv1.StatefulSet{}
			Expect(c.Get(ctx, req, updatedSts)).To(Succeed())
			Expect(updatedSts.Annotations).To(HaveKey("configmap.restarter.kubedoop.dev/test-cm-volume"))
		})

		It("should update annotations when statefulset has secret volume mount", func() {
			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-secret-volume",
					Namespace: "default",
				},
				Data: map[string][]byte{"key": []byte("value")},
			}
			Expect(c.Create(ctx, secret)).To(Succeed())

			sts := &appv1.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-sts",
					Namespace: "default",
					Labels: map[string]string{
						restart.RestartertLabelName: restart.RestartertLabelValue,
					},
				},
				Spec: appv1.StatefulSetSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Volumes: []corev1.Volume{
								{
									Name: "secret-volume",
									VolumeSource: corev1.VolumeSource{
										Secret: &corev1.SecretVolumeSource{
											SecretName: "test-secret-volume",
										},
									},
								},
							},
							Containers: []corev1.Container{
								{
									Name:  "test",
									Image: "nginx",
									VolumeMounts: []corev1.VolumeMount{
										{
											Name:      "secret-volume",
											MountPath: "/etc/secret",
										},
									},
								},
							},
						},
					},
				},
			}
			Expect(c.Create(ctx, sts)).To(Succeed())

			req := types.NamespacedName{
				Name:      "test-sts",
				Namespace: "default",
			}
			result, err := r.Reconcile(ctx, ctrl.Request{NamespacedName: req})
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(ctrl.Result{}))

			updatedSts := &appv1.StatefulSet{}
			Expect(c.Get(ctx, req, updatedSts)).To(Succeed())
			Expect(updatedSts.Annotations).To(HaveKey("secret.restarter.kubedoop.dev/test-secret-volume"))
		})
	})
})
