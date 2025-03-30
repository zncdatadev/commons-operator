package restart_test

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/zncdatadev/commons-operator/internal/controller/restart"
	"github.com/zncdatadev/operator-go/pkg/constants"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubectl/pkg/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var _ = Describe("PodExpiresReconciler", func() {
	var (
		fakeReconciler *restart.PodExpiresReconciler
		pod            *corev1.Pod
		fakeClient     ctrlclient.Client
	)

	BeforeEach(func() {
		// Create a Pod with an expired timestamp
		pod = &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-pod",
				Namespace: "default",
				Annotations: map[string]string{
					fmt.Sprintf("%svolume_id", constants.PrefixLabelRestarterExpiresAt): time.Now().Add(-1 * time.Hour).Format(time.RFC3339),
				},
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:  "test-container",
						Image: "busybox",
					},
				},
			},
		}

	})

	Context("Pod expiration with envtest", func() {

		It("should evict expired Pod correctly", func() {
			Expect(k8sClient.Create(ctx, pod)).NotTo(HaveOccurred())

			By("Waiting for the Pod to be evicted or deleted")
			Eventually(func() bool {
				evictedPod := &corev1.Pod{}
				err := k8sClient.Get(ctx, ctrlclient.ObjectKeyFromObject(pod), evictedPod)
				if err != nil {
					// If Pod does not exist, return true
					return ctrlclient.IgnoreNotFound(err) == nil
				}
				// If DeletionTimestamp is not nil, pod is being deleted
				return evictedPod.DeletionTimestamp != nil
			}, time.Second*5, time.Millisecond*500).Should(BeTrue())
		})

		It("should evict expired Pod after 5 seconds", func() {
			By("Set expiration time to 5 seconds in the future")
			pod.Annotations = map[string]string{
				fmt.Sprintf("%svolume_id", constants.PrefixLabelRestarterExpiresAt): time.Now().Add(5 * time.Second).Format(time.RFC3339),
			}
			Expect(k8sClient.Create(ctx, pod)).NotTo(HaveOccurred())

			By("Check pod is not evicted")
			evictedPod := &corev1.Pod{}
			Expect(k8sClient.Get(ctx, ctrlclient.ObjectKeyFromObject(pod), evictedPod)).NotTo(HaveOccurred())
			Expect(evictedPod.DeletionTimestamp).To(BeNil())

			By("Waiting for the Pod to be evicted")
			Eventually(func() bool {
				evictedPod := &corev1.Pod{}
				err := k8sClient.Get(ctx, ctrlclient.ObjectKeyFromObject(pod), evictedPod)
				if err != nil {
					// If Pod does not exist, return true
					return ctrlclient.IgnoreNotFound(err) == nil
				}
				// If DeletionTimestamp is not nil, pod is being deleted
				return evictedPod.DeletionTimestamp != nil
			}, time.Second*10, time.Millisecond*500).Should(BeTrue())
		})
	})

	Context("Pod expiration with fake client", func() {
		BeforeEach(func() {
			ctx = context.Background()
			fakeClient = fakeclient.NewClientBuilder().WithScheme(scheme.Scheme).Build()
			fakeReconciler = &restart.PodExpiresReconciler{
				Client: fakeClient,
				Schema: fakeClient.Scheme(),
			}

			err := fakeClient.Create(ctx, pod)
			Expect(err).NotTo(HaveOccurred())

			existPod := &corev1.Pod{}
			err = fakeClient.Get(ctx, ctrlclient.ObjectKeyFromObject(pod), existPod)
			Expect(err).NotTo(HaveOccurred())
		})

		AfterEach(func() {
			// cleanup resources
			err := fakeClient.Delete(ctx, pod)
			Expect(ctrlclient.IgnoreNotFound(err)).NotTo(HaveOccurred())
		})

		It("should evict expired Pod correctly", func() {
			req := ctrl.Request{
				NamespacedName: ctrlclient.ObjectKeyFromObject(pod),
			}
			_, err := fakeReconciler.Reconcile(ctx, req)
			Expect(err).NotTo(HaveOccurred())

			By("Waiting for the Pod to be evicted or deleted")
			Eventually(func() bool {
				evictedPod := &corev1.Pod{}
				err := fakeClient.Get(ctx, ctrlclient.ObjectKeyFromObject(pod), evictedPod)
				if err != nil {
					// If Pod does not exist, return true
					return ctrlclient.IgnoreNotFound(err) == nil
				}
				// If DeletionTimestamp is not nil, pod is being deleted
				return evictedPod.DeletionTimestamp != nil
			}, time.Second*5, time.Millisecond*500).Should(BeTrue())
		})

		It("should not evict unexpired Pod", func() {
			By("Set expiration time to 1 hour in the future")
			pod.Annotations = map[string]string{
				fmt.Sprintf("%svolume_id", constants.PrefixLabelRestarterExpiresAt): time.Now().Add(1 * time.Hour).Format(time.RFC3339),
			}
			err := fakeClient.Update(ctx, pod)
			Expect(err).NotTo(HaveOccurred())

			By("Call Reconcile")
			req := ctrl.Request{
				NamespacedName: ctrlclient.ObjectKeyFromObject(pod),
			}
			_, err = fakeReconciler.Reconcile(ctx, req)
			Expect(err).NotTo(HaveOccurred())

			By("Checking that the Pod is not evicted")
			Consistently(func() bool {
				nonEvictedPod := &corev1.Pod{}
				err := fakeClient.Get(ctx, ctrlclient.ObjectKeyFromObject(pod), nonEvictedPod)
				if err != nil {
					return false
				}
				return nonEvictedPod.DeletionTimestamp == nil
			}, time.Second*5, time.Millisecond*500).Should(BeTrue())
		})

		It("should evict expired Pod after 5 seconds", func() {
			By("Set expiration time to 5 seconds in the future")
			pod.Annotations = map[string]string{
				fmt.Sprintf("%svolume_id", constants.PrefixLabelRestarterExpiresAt): time.Now().Add(5 * time.Second).Format(time.RFC3339),
			}
			err := fakeClient.Update(ctx, pod)
			Expect(err).NotTo(HaveOccurred())

			By("Reconcile until the Pod is evicted")
			Eventually(func() bool {
				for i := 0; i < 5; i++ {
					result, err := fakeReconciler.Reconcile(ctx, ctrl.Request{
						NamespacedName: ctrlclient.ObjectKeyFromObject(pod),
					})
					if err != nil {
						return false
					}

					if result.IsZero() {
						// Reconcile done
						break
					}

					time.Sleep(result.RequeueAfter)
				}
				return true
			}, time.Second*10, time.Millisecond*500).Should(BeTrue())

			By("Verifying the Pod has been evicted")
			Eventually(func() bool {
				evictedPod := &corev1.Pod{}
				err := fakeClient.Get(ctx, ctrlclient.ObjectKeyFromObject(pod), evictedPod)
				if err != nil {
					// If NotFound error, Pod has been deleted
					return ctrlclient.IgnoreNotFound(err) == nil
				}
				// Check if DeletionTimestamp is set
				return evictedPod.DeletionTimestamp != nil
			}, time.Second*10, time.Millisecond*500).Should(BeTrue())
		})
	})
})
