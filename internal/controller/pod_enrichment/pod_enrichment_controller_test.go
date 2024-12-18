package pod_enrichment_test

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/kubectl/pkg/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/zncdatadev/commons-operator/internal/controller/pod_enrichment"
)

var _ = Describe("PodEnrichmentController", func() {
	var (
		pod            *corev1.Pod
		node           *corev1.Node
		fakeClient     client.Client
		fakeReconciler *pod_enrichment.PodEnrichmentReconciler
	)

	BeforeEach(func() {
		fakeClient = fake.NewClientBuilder().WithScheme(scheme.Scheme).Build()
		fakeReconciler = &pod_enrichment.PodEnrichmentReconciler{
			Client: fakeClient,
			Schema: fakeClient.Scheme(),
		}
		pod = &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-pod",
				Namespace: "default",
				Labels: map[string]string{
					pod_enrichment.PodEnrichmentLabelName: pod_enrichment.PodEnrichmentLabelValue,
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
		node = &corev1.Node{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-node",
			},
			Status: corev1.NodeStatus{
				Addresses: []corev1.NodeAddress{
					{
						Type:    corev1.NodeInternalIP,
						Address: "192.168.1.1",
					},
					{
						Type:    corev1.NodeHostName,
						Address: "test-node.example.com",
					},
				},
			},
		}
	})

	Context("Pod Reconciliation with envtest", func() {
		BeforeEach(func() {
			Expect(k8sClient.Create(ctx, node)).To(Succeed())
		})

		It("should update pod annotation with node address", func() {
			Expect(k8sClient.Create(ctx, pod)).To(Succeed())

			By("waiting for pod to be scheduled")
			time.Sleep(time.Second * 2)

			obj := &corev1.Pod{}

			By("pod not scheduled")
			Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(pod), obj)).To(Succeed())
			nodeAddr := obj.Annotations[pod_enrichment.PodenrichmentNodeAddressAnnotationName]
			Expect(nodeAddr).To(BeEmpty())

			// Due to the way envtest works, pods are not scheduled on nodes, so we need e2e tests to verify this
		})
	})

	Context("Pod Reconciliation with fake client", func() {
		var (
			req ctrl.Request
			ctx context.Context
		)

		BeforeEach(func() {
			ctx = context.TODO()
			req = ctrl.Request{
				NamespacedName: types.NamespacedName{
					Name:      pod.Name,
					Namespace: pod.Namespace,
				},
			}
		})

		It("should return error when node does not exist", func() {
			pod.Spec.NodeName = "non-existent-node"
			Expect(fakeClient.Create(ctx, pod)).To(Succeed())

			result, err := fakeReconciler.Reconcile(ctx, req)
			Expect(err).To(HaveOccurred())
			Expect(result.IsZero()).To(BeTrue())
		})

		It("should successfully update pod annotation with node address", func() {
			pod.Spec.NodeName = node.Name
			Expect(fakeClient.Create(ctx, node)).To(Succeed())
			Expect(fakeClient.Create(ctx, pod)).To(Succeed())

			result, err := fakeReconciler.Reconcile(ctx, req)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.IsZero()).To(BeTrue())

			// 验证 Pod 注解是否已更新
			updatedPod := &corev1.Pod{}
			Expect(fakeClient.Get(ctx, req.NamespacedName, updatedPod)).To(Succeed())

			nodeAddr := updatedPod.Annotations[pod_enrichment.PodenrichmentNodeAddressAnnotationName]
			Expect(nodeAddr).To(Equal("test-node.example.com"))
		})

		It("should use fallback address when preferred address type not available", func() {
			node.Status.Addresses = []corev1.NodeAddress{
				{
					Type:    corev1.NodeInternalIP,
					Address: "192.168.1.1",
				},
			}

			pod.Spec.NodeName = node.Name
			Expect(fakeClient.Create(ctx, node)).To(Succeed())
			Expect(fakeClient.Create(ctx, pod)).To(Succeed())

			result, err := fakeReconciler.Reconcile(ctx, req)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.IsZero()).To(BeTrue())

			updatedPod := &corev1.Pod{}
			Expect(fakeClient.Get(ctx, req.NamespacedName, updatedPod)).To(Succeed())

			nodeAddr := updatedPod.Annotations[pod_enrichment.PodenrichmentNodeAddressAnnotationName]
			Expect(nodeAddr).To(Equal("192.168.1.1"))
		})
	})

})
