package restart_test

import (
	"context"
	"time"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/kubectl/pkg/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/zncdatadev/commons-operator/internal/controller/restart"
)

var _ = Describe("StatefulsetController", func() {
	var (
		sts    *appv1.StatefulSet
		cm     *corev1.ConfigMap
		secret *corev1.Secret
		ns     *corev1.Namespace
	)

	BeforeEach(func() {
		// generate random namespace
		ns = &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-ns-" + uuid.New().String()[:8],
			},
		}

		sts = &appv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-sts",
				Namespace: ns.Name,
				Labels: map[string]string{
					restart.RestartertLabelName: restart.RestartertLabelValue,
				},
			},
			Spec: appv1.StatefulSetSpec{
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"app": "test",
					},
				},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"app": "test",
						},
					},
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

		cm = &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-cm",
				Namespace: ns.Name,
			},
			Data: map[string]string{"key": "value"},
		}

		secret = &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-secret",
				Namespace: ns.Name,
			},
			Data: map[string][]byte{"key": []byte("value")},
		}

	})

	Context("Update Statefulsets with envtest", func() {
		BeforeEach(func() {
			// create ns
			Expect(k8sClient.Create(ctx, ns)).To(Succeed())
		})

		AfterEach(func() {
			// cleanup resource
			Expect(client.IgnoreNotFound(k8sClient.Delete(ctx, sts))).To(Succeed())
			Expect(client.IgnoreNotFound(k8sClient.Delete(ctx, cm))).To(Succeed())
			Expect(client.IgnoreNotFound(k8sClient.Delete(ctx, secret))).To(Succeed())

			// cleanup namespace
			Expect(client.IgnoreNotFound(k8sClient.Delete(ctx, ns))).To(Succeed())
		})

		It("should update annotations when statefulset has container ref configmap", func() {
			Expect(k8sClient.Create(ctx, cm)).To(Succeed())

			sts.Spec.Template.Spec.Containers[0].Env = []corev1.EnvVar{
				{
					Name: "TEST",
					ValueFrom: &corev1.EnvVarSource{
						ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: cm.Name,
							},
							Key: "key",
						},
					},
				},
			}

			Expect(k8sClient.Create(ctx, sts)).To(Succeed())

			latestCm := &corev1.ConfigMap{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: cm.Name, Namespace: cm.Namespace}, latestCm)).To(Succeed())
			expectedValue := restart.GenStatefulSetRestartAnnotationValue(latestCm)

			Eventually(func() bool {
				updatedSts := &appv1.StatefulSet{}
				if err := k8sClient.Get(ctx, client.ObjectKeyFromObject(sts), updatedSts); err != nil {
					return false
				}
				return updatedSts.Spec.Template.Annotations[restart.GenStatefulSetRestartAnnotationKey(cm)] == expectedValue
			}, time.Second*5, time.Millisecond*500).Should(BeTrue())
		})

		It("should update annotations when statefulset has container ref secret", func() {
			Expect(k8sClient.Create(ctx, secret)).To(Succeed())

			sts.Spec.Template.Spec.Containers[0].Env = []corev1.EnvVar{
				{
					Name: "TEST",
					ValueFrom: &corev1.EnvVarSource{
						SecretKeyRef: &corev1.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: secret.Name,
							},
							Key: "key",
						},
					},
				},
			}

			Expect(k8sClient.Create(ctx, sts)).To(Succeed())

			latestSecret := &corev1.Secret{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: secret.Name, Namespace: secret.Namespace}, latestSecret)).To(Succeed())
			expectedValue := restart.GenStatefulSetRestartAnnotationValue(latestSecret)

			Eventually(func() bool {
				updatedSts := &appv1.StatefulSet{}
				if err := k8sClient.Get(ctx, client.ObjectKeyFromObject(sts), updatedSts); err != nil {
					return false
				}
				return updatedSts.Spec.Template.Annotations[restart.GenStatefulSetRestartAnnotationKey(secret)] == expectedValue
			}, time.Second*5, time.Millisecond*500).Should(BeTrue())
		})

		It("should update annotations when statefulset has volume mount configmap", func() {
			Expect(k8sClient.Create(ctx, cm)).To(Succeed())

			sts.Spec.Template.Spec.Volumes = []corev1.Volume{
				{
					Name: "config-volume",
					VolumeSource: corev1.VolumeSource{
						ConfigMap: &corev1.ConfigMapVolumeSource{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: cm.Name,
							},
						},
					},
				},
			}

			sts.Spec.Template.Spec.Containers[0].VolumeMounts = []corev1.VolumeMount{
				{
					Name:      "config-volume",
					MountPath: "/etc/config",
				},
			}

			Expect(k8sClient.Create(ctx, sts)).To(Succeed())

			latestCm := &corev1.ConfigMap{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: cm.Name, Namespace: cm.Namespace}, latestCm)).To(Succeed())
			expectedValue := restart.GenStatefulSetRestartAnnotationValue(latestCm)

			Eventually(func() bool {
				updatedSts := &appv1.StatefulSet{}
				if err := k8sClient.Get(ctx, client.ObjectKeyFromObject(sts), updatedSts); err != nil {
					return false
				}
				return updatedSts.Spec.Template.Annotations[restart.GenStatefulSetRestartAnnotationKey(cm)] == expectedValue
			}, time.Second*5, time.Millisecond*500).Should(BeTrue())
		})

		It("should update annotations when statefulset has volume mount secret", func() {
			Expect(k8sClient.Create(ctx, secret)).To(Succeed())

			sts.Spec.Template.Spec.Volumes = []corev1.Volume{
				{
					Name: "secret-volume",
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							SecretName: secret.Name,
						},
					},
				},
			}

			sts.Spec.Template.Spec.Containers[0].VolumeMounts = []corev1.VolumeMount{
				{
					Name:      "secret-volume",
					MountPath: "/etc/secret",
				},
			}

			Expect(k8sClient.Create(ctx, sts)).To(Succeed())

			latestSecret := &corev1.Secret{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: secret.Name, Namespace: secret.Namespace}, latestSecret)).To(Succeed())
			expectedValue := restart.GenStatefulSetRestartAnnotationValue(latestSecret)

			Eventually(func() bool {
				updatedSts := &appv1.StatefulSet{}
				if err := k8sClient.Get(ctx, client.ObjectKeyFromObject(sts), updatedSts); err != nil {
					return false
				}
				return updatedSts.Spec.Template.Annotations[restart.GenStatefulSetRestartAnnotationKey(secret)] == expectedValue
			}, time.Second*5, time.Millisecond*500).Should(BeTrue())
		})

		It("should update annotations when statefulset has container ref configmap and secret", func() {
			Expect(k8sClient.Create(ctx, cm)).To(Succeed())
			Expect(k8sClient.Create(ctx, secret)).To(Succeed())

			sts.Spec.Template.Spec.Containers[0].Env = []corev1.EnvVar{
				{
					Name: "TEST",
					ValueFrom: &corev1.EnvVarSource{
						ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: cm.Name,
							},
							Key: "key",
						},
					},
				},
				{
					Name: "TEST_SECRET",
					ValueFrom: &corev1.EnvVarSource{
						SecretKeyRef: &corev1.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: secret.Name,
							},
							Key: "key",
						},
					},
				},
			}

			Expect(k8sClient.Create(ctx, sts)).To(Succeed())

			latestCm := &corev1.ConfigMap{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: cm.Name, Namespace: cm.Namespace}, latestCm)).To(Succeed())
			expectedCmValue := restart.GenStatefulSetRestartAnnotationValue(latestCm)

			latestSecret := &corev1.Secret{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: secret.Name, Namespace: secret.Namespace}, latestSecret)).To(Succeed())
			expectedSecretValue := restart.GenStatefulSetRestartAnnotationValue(latestSecret)

			Eventually(func() bool {
				updatedSts := &appv1.StatefulSet{}
				if err := k8sClient.Get(ctx, client.ObjectKeyFromObject(sts), updatedSts); err != nil {
					return false
				}
				return updatedSts.Spec.Template.Annotations[restart.GenStatefulSetRestartAnnotationKey(cm)] == expectedCmValue &&
					updatedSts.Spec.Template.Annotations[restart.GenStatefulSetRestartAnnotationKey(secret)] == expectedSecretValue
			}, time.Second*5, time.Millisecond*500).Should(BeTrue())
		})

		It("should update annotations when statefulset has container ref configmap and secret with updated", func() {
			Expect(k8sClient.Create(ctx, cm)).To(Succeed())
			Expect(k8sClient.Create(ctx, secret)).To(Succeed())

			sts.Spec.Template.Spec.Containers[0].Env = []corev1.EnvVar{
				{
					Name: "TEST",
					ValueFrom: &corev1.EnvVarSource{
						ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: cm.Name,
							},
							Key: "key",
						},
					},
				},
				{
					Name: "TEST_SECRET",
					ValueFrom: &corev1.EnvVarSource{
						SecretKeyRef: &corev1.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: secret.Name,
							},
							Key: "key",
						},
					},
				},
			}

			Expect(k8sClient.Create(ctx, sts)).To(Succeed())

			latestCm := &corev1.ConfigMap{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: cm.Name, Namespace: cm.Namespace}, latestCm)).To(Succeed())
			expectedCmValue := restart.GenStatefulSetRestartAnnotationValue(latestCm)

			latestSecret := &corev1.Secret{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: secret.Name, Namespace: secret.Namespace}, latestSecret)).To(Succeed())
			expectedSecretValue := restart.GenStatefulSetRestartAnnotationValue(latestSecret)

			Eventually(func() bool {
				updatedSts := &appv1.StatefulSet{}
				if err := k8sClient.Get(ctx, client.ObjectKeyFromObject(sts), updatedSts); err != nil {
					return false
				}
				return updatedSts.Spec.Template.Annotations[restart.GenStatefulSetRestartAnnotationKey(cm)] == expectedCmValue &&
					updatedSts.Spec.Template.Annotations[restart.GenStatefulSetRestartAnnotationKey(secret)] == expectedSecretValue
			}, time.Second*5, time.Millisecond*500).Should(BeTrue())

			By("update configmap")
			cm.Data["key"] = "new-value"
			Expect(k8sClient.Update(ctx, cm)).To(Succeed())

			By("update secret")
			secret.Data["key"] = []byte("new-value")
			Expect(k8sClient.Update(ctx, secret)).To(Succeed())

			latestCm = &corev1.ConfigMap{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: cm.Name, Namespace: cm.Namespace}, latestCm)).To(Succeed())
			newExpectedCmValue := restart.GenStatefulSetRestartAnnotationValue(latestCm)

			latestSecret = &corev1.Secret{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: secret.Name, Namespace: secret.Namespace}, latestSecret)).To(Succeed())
			newExpectedSecretValue := restart.GenStatefulSetRestartAnnotationValue(latestSecret)

			Eventually(func() bool {
				updatedSts := &appv1.StatefulSet{}
				if err := k8sClient.Get(ctx, client.ObjectKeyFromObject(sts), updatedSts); err != nil {
					return false
				}
				return updatedSts.Spec.Template.Annotations[restart.GenStatefulSetRestartAnnotationKey(cm)] == newExpectedCmValue &&
					updatedSts.Spec.Template.Annotations[restart.GenStatefulSetRestartAnnotationKey(secret)] == newExpectedSecretValue
			}, time.Second*5, time.Millisecond*500).Should(BeTrue())
		})

	})

	Context("Update Statefulsets with fake client", func() {
		var (
			c client.Client
			r *restart.StatefulSetReconciler
		)

		BeforeEach(func() {
			ctx = context.Background()
			c = fake.NewClientBuilder().WithScheme(scheme.Scheme).Build()
			r = &restart.StatefulSetReconciler{
				Client: c,
				Schema: c.Scheme(),
			}

			Expect(c.Create(ctx, ns)).To(Succeed())

			Expect(c.Create(ctx, cm)).To(Succeed())
			Expect(c.Create(ctx, secret)).To(Succeed())
		})

		AfterEach(func() {
			// cleanup resource
			Expect(client.IgnoreNotFound(c.Delete(ctx, sts))).To(Succeed())
			Expect(client.IgnoreNotFound(c.Delete(ctx, cm))).To(Succeed())
			Expect(client.IgnoreNotFound(c.Delete(ctx, secret))).To(Succeed())

			// cleanup namespace
			Expect(client.IgnoreNotFound(c.Delete(ctx, ns))).To(Succeed())
		})
		It("should ignore when statefulset not found", func() {
			req := types.NamespacedName{
				Name:      sts.Name,
				Namespace: ns.Name,
			}
			result, err := r.Reconcile(ctx, ctrl.Request{NamespacedName: req})
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(ctrl.Result{}))
		})

		It("should skip when statefulset has no configmap/secret refs", func() {
			req := types.NamespacedName{
				Name:      sts.Name,
				Namespace: ns.Name,
			}
			result, err := r.Reconcile(ctx, ctrl.Request{NamespacedName: req})
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(ctrl.Result{}))
		})

		It("should update annotations when statefulset has configmap refs", func() {
			sts.Spec.Template.Spec.Containers[0].Env = []corev1.EnvVar{
				{
					Name: "TEST",
					ValueFrom: &corev1.EnvVarSource{
						ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: cm.Name,
							},
							Key: "key",
						},
					},
				},
			}
			Expect(c.Create(ctx, sts)).To(Succeed())

			req := types.NamespacedName{
				Name:      sts.Name,
				Namespace: ns.Name,
			}
			result, err := r.Reconcile(ctx, ctrl.Request{NamespacedName: req})
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(ctrl.Result{}))

			updatedSts := &appv1.StatefulSet{}
			Expect(c.Get(ctx, req, updatedSts)).To(Succeed())

			latestCm := &corev1.ConfigMap{}
			Expect(c.Get(ctx, types.NamespacedName{Name: cm.Name, Namespace: cm.Namespace}, latestCm)).To(Succeed())
			expectedValue := restart.GenStatefulSetRestartAnnotationValue(latestCm)
			Expect(updatedSts.Spec.Template.Annotations[restart.GenStatefulSetRestartAnnotationKey(cm)]).To(Equal(expectedValue))
		})

		It("should update annotations when statefulset has secret refs", func() {
			sts.Spec.Template.Spec.Containers[0].Env = []corev1.EnvVar{
				{
					Name: "TEST",
					ValueFrom: &corev1.EnvVarSource{
						SecretKeyRef: &corev1.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: secret.Name,
							},
							Key: "key",
						},
					},
				},
			}
			Expect(c.Create(ctx, sts)).To(Succeed())

			req := types.NamespacedName{
				Name:      sts.Name,
				Namespace: ns.Name,
			}
			result, err := r.Reconcile(ctx, ctrl.Request{NamespacedName: req})
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(ctrl.Result{}))

			updatedSts := &appv1.StatefulSet{}
			Expect(c.Get(ctx, req, updatedSts)).To(Succeed())

			latestSecret := &corev1.Secret{}
			Expect(c.Get(ctx, types.NamespacedName{Name: secret.Name, Namespace: secret.Namespace}, latestSecret)).To(Succeed())
			expectedValue := restart.GenStatefulSetRestartAnnotationValue(latestSecret)
			Expect(updatedSts.Spec.Template.Annotations[restart.GenStatefulSetRestartAnnotationKey(secret)]).To(Equal(expectedValue))
		})

		It("should update annotations when statefulset has configmap volume mount", func() {
			existCm := &corev1.ConfigMap{}
			Expect(c.Get(ctx, types.NamespacedName{Name: cm.Name, Namespace: cm.Namespace}, existCm)).To(Succeed())
			Expect(existCm).NotTo(BeNil())

			sts.Spec.Template.Spec.Volumes = []corev1.Volume{
				{
					Name: "config-volume",
					VolumeSource: corev1.VolumeSource{
						ConfigMap: &corev1.ConfigMapVolumeSource{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: cm.Name,
							},
						},
					},
				},
			}
			sts.Spec.Template.Spec.Containers[0].VolumeMounts = []corev1.VolumeMount{
				{
					Name:      "config-volume",
					MountPath: "/etc/config",
				},
			}
			Expect(c.Create(ctx, sts)).To(Succeed())

			req := types.NamespacedName{
				Name:      sts.Name,
				Namespace: ns.Name,
			}
			result, err := r.Reconcile(ctx, ctrl.Request{NamespacedName: req})
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(ctrl.Result{}))

			updatedSts := &appv1.StatefulSet{}
			Expect(c.Get(ctx, req, updatedSts)).To(Succeed())

			latestCm := &corev1.ConfigMap{}
			Expect(c.Get(ctx, types.NamespacedName{Name: cm.Name, Namespace: cm.Namespace}, latestCm)).To(Succeed())
			expectedValue := restart.GenStatefulSetRestartAnnotationValue(latestCm)
			Expect(updatedSts.Spec.Template.Annotations[restart.GenStatefulSetRestartAnnotationKey(cm)]).To(Equal(expectedValue))
		})

		It("should update annotations when statefulset has secret volume mount", func() {
			sts.Spec.Template.Spec.Volumes = []corev1.Volume{
				{
					Name: "secret-volume",
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							SecretName: secret.Name,
						},
					},
				},
			}
			sts.Spec.Template.Spec.Containers[0].VolumeMounts = []corev1.VolumeMount{
				{
					Name:      "secret-volume",
					MountPath: "/etc/secret",
				},
			}
			Expect(c.Create(ctx, sts)).To(Succeed())

			req := types.NamespacedName{
				Name:      sts.Name,
				Namespace: ns.Name,
			}
			result, err := r.Reconcile(ctx, ctrl.Request{NamespacedName: req})
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(ctrl.Result{}))

			updatedSts := &appv1.StatefulSet{}
			Expect(c.Get(ctx, req, updatedSts)).To(Succeed())

			latestSecret := &corev1.Secret{}
			Expect(c.Get(ctx, types.NamespacedName{Name: secret.Name, Namespace: secret.Namespace}, latestSecret)).To(Succeed())
			expectedValue := restart.GenStatefulSetRestartAnnotationValue(latestSecret)
			Expect(updatedSts.Spec.Template.Annotations[restart.GenStatefulSetRestartAnnotationKey(secret)]).To(Equal(expectedValue))
		})

		It("should update annotations when statefulset has configmap and secret refs", func() {
			sts.Spec.Template.Spec.Containers[0].Env = []corev1.EnvVar{
				{
					Name: "TEST",
					ValueFrom: &corev1.EnvVarSource{
						ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: cm.Name,
							},
							Key: "key",
						},
					},
				},
				{
					Name: "TEST_SECRET",
					ValueFrom: &corev1.EnvVarSource{
						SecretKeyRef: &corev1.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: secret.Name,
							},
							Key: "key",
						},
					},
				},
			}
			Expect(c.Create(ctx, sts)).To(Succeed())

			req := types.NamespacedName{
				Name:      sts.Name,
				Namespace: ns.Name,
			}
			result, err := r.Reconcile(ctx, ctrl.Request{NamespacedName: req})
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(ctrl.Result{}))

			updatedSts := &appv1.StatefulSet{}
			Expect(c.Get(ctx, req, updatedSts)).To(Succeed())

			latestCm := &corev1.ConfigMap{}
			Expect(c.Get(ctx, types.NamespacedName{Name: cm.Name, Namespace: cm.Namespace}, latestCm)).To(Succeed())
			expectedCmValue := restart.GenStatefulSetRestartAnnotationValue(latestCm)
			Expect(updatedSts.Spec.Template.Annotations[restart.GenStatefulSetRestartAnnotationKey(cm)]).To(Equal(expectedCmValue))

			latestSecret := &corev1.Secret{}
			Expect(c.Get(ctx, types.NamespacedName{Name: secret.Name, Namespace: secret.Namespace}, latestSecret)).To(Succeed())
			expectedSecretValue := restart.GenStatefulSetRestartAnnotationValue(latestSecret)
			Expect(updatedSts.Spec.Template.Annotations[restart.GenStatefulSetRestartAnnotationKey(secret)]).To(Equal(expectedSecretValue))
		})

		It("should update annotations when statefulset has volume ref configmap and secret with updated", func() {
			sts.Spec.Template.Spec.Volumes = []corev1.Volume{
				{
					Name: "config-volume",
					VolumeSource: corev1.VolumeSource{
						ConfigMap: &corev1.ConfigMapVolumeSource{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: cm.Name,
							},
						},
					},
				},
				{
					Name: "secret-volume",
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							SecretName: secret.Name,
						},
					},
				},
			}

			sts.Spec.Template.Spec.Containers[0].VolumeMounts = []corev1.VolumeMount{
				{
					Name:      "config-volume",
					MountPath: "/etc/config",
				},
				{
					Name:      "secret-volume",
					MountPath: "/etc/secret",
				},
			}
			Expect(c.Create(ctx, sts)).To(Succeed())

			req := types.NamespacedName{
				Name:      sts.Name,
				Namespace: ns.Name,
			}
			result, err := r.Reconcile(ctx, ctrl.Request{NamespacedName: req})
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(ctrl.Result{}))

			updatedSts := &appv1.StatefulSet{}
			Expect(c.Get(ctx, req, updatedSts)).To(Succeed())

			latestCm := &corev1.ConfigMap{}
			Expect(c.Get(ctx, types.NamespacedName{Name: cm.Name, Namespace: cm.Namespace}, latestCm)).To(Succeed())
			expectedCmValue := restart.GenStatefulSetRestartAnnotationValue(latestCm)
			Expect(updatedSts.Spec.Template.Annotations[restart.GenStatefulSetRestartAnnotationKey(cm)]).To(Equal(expectedCmValue))

			latestSecret := &corev1.Secret{}
			Expect(c.Get(ctx, types.NamespacedName{Name: secret.Name, Namespace: secret.Namespace}, latestSecret)).To(Succeed())
			expectedSecretValue := restart.GenStatefulSetRestartAnnotationValue(latestSecret)
			Expect(updatedSts.Spec.Template.Annotations[restart.GenStatefulSetRestartAnnotationKey(secret)]).To(Equal(expectedSecretValue))

			By("update configmap")
			cm.Data["key"] = "new-value"
			Expect(c.Update(ctx, cm)).To(Succeed())

			By("update configmap")
			secret.Data["key"] = []byte("new-value")
			Expect(c.Update(ctx, secret)).To(Succeed())

			result, err = r.Reconcile(ctx, ctrl.Request{NamespacedName: req})
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(ctrl.Result{}))

			updatedSts = &appv1.StatefulSet{}
			Expect(c.Get(ctx, req, updatedSts)).To(Succeed())

			latestCm = &corev1.ConfigMap{}
			Expect(c.Get(ctx, types.NamespacedName{Name: cm.Name, Namespace: cm.Namespace}, latestCm)).To(Succeed())
			newExpectedCmValue := restart.GenStatefulSetRestartAnnotationValue(latestCm)
			Expect(updatedSts.Spec.Template.Annotations[restart.GenStatefulSetRestartAnnotationKey(cm)]).To(Equal(newExpectedCmValue))

			latestSecret = &corev1.Secret{}
			Expect(c.Get(ctx, types.NamespacedName{Name: secret.Name, Namespace: secret.Namespace}, latestSecret)).To(Succeed())
			newExpectedSecretValue := restart.GenStatefulSetRestartAnnotationValue(latestSecret)
			Expect(updatedSts.Spec.Template.Annotations[restart.GenStatefulSetRestartAnnotationKey(secret)]).To(Equal(newExpectedSecretValue))
		})
	})
})
