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
)

var _ = Describe("PodExpireReconciler", func() {
	var (
		reconciler *restart.PodExpireReconciler
		ctx        = context.TODO()
		pod        *corev1.Pod
	)

	BeforeEach(func() {

		reconciler = &restart.PodExpireReconciler{
			Client: k8sClient,
			Schema: scheme.Scheme,
		}

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

		err := k8sClient.Create(ctx, pod)
		Expect(err).NotTo(HaveOccurred())

		existPod := &corev1.Pod{}
		err = k8sClient.Get(ctx, ctrlclient.ObjectKeyFromObject(pod), existPod)
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		// cleanup resources
		err := k8sClient.Delete(ctx, pod)
		Expect(ctrlclient.IgnoreNotFound(err)).NotTo(HaveOccurred())
	})

	It("should evict expired Pod correctly", func() {
		req := ctrl.Request{
			NamespacedName: ctrlclient.ObjectKeyFromObject(pod),
		}
		_, err := reconciler.Reconcile(ctx, req)
		Expect(err).NotTo(HaveOccurred())

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

	It("should not evict unexpired Pod", func() {
		By("Set expiration time to 1 hour in the future")
		pod.Annotations = map[string]string{
			fmt.Sprintf("%svolume_id", constants.PrefixLabelRestarterExpiresAt): time.Now().Add(1 * time.Hour).Format(time.RFC3339),
		}
		err := k8sClient.Update(ctx, pod)
		Expect(err).NotTo(HaveOccurred())

		By("Call Reconcile")
		req := ctrl.Request{
			NamespacedName: ctrlclient.ObjectKeyFromObject(pod),
		}
		_, err = reconciler.Reconcile(ctx, req)
		Expect(err).NotTo(HaveOccurred())

		By("Checking that the Pod is not evicted")
		Consistently(func() bool {
			nonEvictedPod := &corev1.Pod{}
			err := k8sClient.Get(ctx, ctrlclient.ObjectKeyFromObject(pod), nonEvictedPod)
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
		err := k8sClient.Update(ctx, pod)
		Expect(err).NotTo(HaveOccurred())

		By("Reconcile until the Pod is evicted")
		Eventually(func() bool {
			for i := 0; i < 5; i++ {
				result, err := reconciler.Reconcile(ctx, ctrl.Request{
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
			err := k8sClient.Get(ctx, ctrlclient.ObjectKeyFromObject(pod), evictedPod)
			if err != nil {
				// If NotFound error, Pod has been deleted
				return ctrlclient.IgnoreNotFound(err) == nil
			}
			// Check if DeletionTimestamp is set
			return evictedPod.DeletionTimestamp != nil
		}, time.Second*10, time.Millisecond*500).Should(BeTrue())
	})
})
