package pod_enrichment

import (
	"context"
	"errors"
	"time"

	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
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

func (p *PodHandler) GetNodeName() string {
	return p.Pod.Spec.NodeName
}

func (p *PodHandler) Scheduled() bool {
	if p.Pod.Spec.NodeName == "" {
		logger.Info("Pod is not scheduled", "pod", p.Pod.Name, "namespace", p.Pod.Namespace)
		return false
	}
	return true
}

func (p *PodHandler) GetNodeAddress(ctx context.Context) (string, error) {
	node := &corev1.Node{}

	if err := p.Client.Get(ctx, client.ObjectKey{Name: p.GetNodeName()}, node); err != nil {
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

	return "", errors.New("node address not found")
}

func (p *PodHandler) UpdateNodeAddrToPodMeta(ctx context.Context, nodeAddress string) error {

	annonations := p.Pod.GetAnnotations()

	if annonations == nil {
		annonations = make(map[string]string)
	}

	annonations["enrichment.zncdata.dev/node-address"] = nodeAddress

	p.Pod.SetAnnotations(annonations)
	if err := p.Client.Update(ctx, p.Pod); err != nil {
		return err
	}

	return nil
}

func (p *PodHandler) Reconcile(ctx context.Context) (ctrl.Result, error) {
	if !p.Scheduled() {
		return ctrl.Result{RequeueAfter: time.Second * 2}, nil
	}

	nodeAddress, err := p.GetNodeAddress(ctx)
	if err != nil {
		return ctrl.Result{}, err
	}

	if err := p.UpdateNodeAddrToPodMeta(ctx, nodeAddress); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}
