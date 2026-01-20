package matcher

import (
	"context"
	"fmt"

	v1 "agones.dev/agones/pkg/apis/allocation/v1"
	"agones.dev/agones/pkg/client/clientset/versioned"
	"agones.dev/agones/pkg/util/runtime"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
)

func AllocateGameServer() (string, error) {
	config, err := rest.InClusterConfig()
	logger := runtime.NewLoggerWithSource("allocator")
	if err != nil {
		logger.WithError(err).Fatal("Could not create in-cluster config")
	}

	agonesClient, err := versioned.NewForConfig(config)
	if err != nil {
		logger.WithError(err).Fatal("Could not create Agones clientset")
	}

	// Create GameServerAllocation
	allocation := &v1.GameServerAllocation{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "game-alloc-",
			Namespace:    "default",
		},
		Spec: v1.GameServerAllocationSpec{
			Selectors: []v1.GameServerSelector{{
				LabelSelector: metav1.LabelSelector{
					MatchLabels: map[string]string{
						"agones.dev/fleet": "rps-game",
					},
				},
			}},
		},
	}

	result, err := agonesClient.AllocationV1().GameServerAllocations("default").Create(context.TODO(), allocation, metav1.CreateOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to allocate GameServer: %w", err)
	}

	// Extract the IP and port
	if len(result.Status.Ports) == 0 {
		return "", fmt.Errorf("failed to allocate GameServer: no ports available in the allocated GameServer")
	}
	ip := result.Status.Address
	port := result.Status.Ports[0].Port
	return fmt.Sprintf("http://%s:%d", ip, port), nil
}
