package restart_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/client-go/rest"
	"k8s.io/kubectl/pkg/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	"github.com/zncdatadev/commons-operator/internal/controller/restart"
)

var (
	cfg       *rest.Config
	k8sClient ctrlclient.Client
	ctx       context.Context
	cancel    context.CancelFunc
	testEnv   *envtest.Environment
)

func TestReconciler(t *testing.T) {

	testK8sVersion := os.Getenv("ENVTEST_K8S_VERSION")
	if testK8sVersion == "" {
		logf.Log.Info("ENVTEST_K8S_VERSION is not set, using default version")
		testK8sVersion = "1.26.1"
	}
	if asserts := os.Getenv("KUBEBUILDER_ASSETS"); asserts == "" {
		logf.Log.Info("KUBEBUILDER_ASSETS is not set, using default version " + testK8sVersion)
		err := os.Setenv("KUBEBUILDER_ASSETS", filepath.Join("..", "..", "..", "bin", "k8s", fmt.Sprintf("%s-%s-%s", testK8sVersion, runtime.GOOS, runtime.GOARCH)))
		if err != nil {
			t.Errorf("Failed to set KUBEBUILDER_ASSETS")
		}
	}

	RegisterFailHandler(Fail)
	RunSpecs(t, "Reconciler Suite")
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))
	ctx, cancel = context.WithCancel(context.Background())
	By("bootstrapping test environment")

	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "..", "..", "config", "crd", "bases")},
		ErrorIfCRDPathMissing: true,
	}

	var err error

	cfg, err = testEnv.Start()
	Expect(err).ToNot(HaveOccurred())
	Expect(cfg).ToNot(BeNil())

	k8sClient, err = ctrlclient.New(cfg, ctrlclient.Options{Scheme: scheme.Scheme})
	Expect(err).ToNot(HaveOccurred())
	Expect(k8sClient).ToNot(BeNil())

	k8sManager, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme:  scheme.Scheme,
		Metrics: metricsserver.Options{BindAddress: ":28080"},
	})
	Expect(err).ToNot(HaveOccurred())

	err = (&restart.StatefulSetReconciler{
		Client: k8sManager.GetClient(),
		Schema: k8sManager.GetScheme(),
	}).SetupWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())

	err = (&restart.PodExpiresReconciler{
		Client: k8sManager.GetClient(),
		Schema: k8sManager.GetScheme(),
	}).SetupWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())

	go func() {
		defer GinkgoRecover()
		err = k8sManager.Start(ctx)
		Expect(err).ToNot(HaveOccurred(), "failed to run manager")
	}()

})

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	cancel()
	err := testEnv.Stop()
	Expect(err).ToNot(HaveOccurred())
})
