package argoutil

import (
	"context"

	"github.com/argoproj-labs/argocd-operator/common"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	liveCount = 0
)

// ClientWrapper wraps multiple clients and inherits default methods.
type ClientWrapper struct {
	client.Client
	liveClient client.Client
}

// Ensure ClientWrapper implements client.Client
var _ client.Client = &ClientWrapper{}

// NewClientWrapper initializes a new ClientWrapper.
func NewClientWrapper(defaultClient, liveClient client.Client) *ClientWrapper {
	return &ClientWrapper{
		Client:     defaultClient, // Embedding inherits all default methods
		liveClient: liveClient,
	}
}

func (cw *ClientWrapper) GetLiveCount() int {
	return liveCount
}

// Get: Tries default cache client, falls back liveClient.
func (cw *ClientWrapper) Get(ctx context.Context, key types.NamespacedName, obj client.Object, opts ...client.GetOption) error {
	err := cw.Client.Get(ctx, key, obj)
	if err == nil {
		return nil // Success
	}

	// Use liveClient to do a live look up of resource
	liveCount += 1
	err = cw.liveClient.Get(ctx, key, obj)
	if err == nil {
		// resource present, add the label so that it ends up in cache
		patch := client.MergeFrom(obj.DeepCopyObject().(client.Object))
		if obj.GetLabels() == nil {
			obj.SetLabels(map[string]string{})
		}
		obj.GetLabels()[common.WatchedByOperatorKey] = common.ArgoCDAppName
		cw.Client.Patch(ctx, obj, patch)
	}
	return err
}
