package services

import (
	"backend/models"
	"context"
	"fmt"
	"log"
	"math/rand"
	"os"
	"strconv"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type KubernetesService struct {
	clientset *kubernetes.Clientset
	namespace string
}

func NewKubernetesService() (*KubernetesService, error) {
	var config *rest.Config
	var err error

	config, err = rest.InClusterConfig()
	if err != nil {
		kubeconfig := os.Getenv("KUBECONFIG")
		if kubeconfig == "" {
			kubeconfig = os.Getenv("HOME") + "/.kube/config"
		}
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			return nil, fmt.Errorf("failed to create k8s config: %v", err)
		}
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create k8s clientset: %v", err)
	}

	namespace := os.Getenv("CTF_NAMESPACE")
	if namespace == "" {
		namespace = "ctf-instances"
	}

	k8sService := &KubernetesService{
		clientset: clientset,
		namespace: namespace,
	}

	if err := k8sService.ensureNamespace(); err != nil {
		return nil, fmt.Errorf("failed to ensure namespace: %v", err)
	}

	return k8sService, nil
}

func (k *KubernetesService) ensureNamespace() error {
	ctx := context.Background()
	_, err := k.clientset.CoreV1().Namespaces().Get(ctx, k.namespace, metav1.GetOptions{})
	if err != nil {
		namespace := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: k.namespace,
				Labels: map[string]string{
					"purpose": "ctf-challenges",
				},
			},
		}
		_, err = k.clientset.CoreV1().Namespaces().Create(ctx, namespace, metav1.CreateOptions{})
		if err != nil {
			return err
		}
		log.Printf("Created namespace: %s", k.namespace)
	}
	return nil
}

func (k *KubernetesService) CreateChallengeInstance(instance *models.Instance, challenge *models.Challenge) error {
	ctx := context.Background()

	podName := fmt.Sprintf("ctf-%d-%d-%d", instance.UserID, challenge.ID, time.Now().Unix()%10000)
	serviceName := fmt.Sprintf("svc-%s", podName)
	externalPort := 30000 + rand.Intn(2000)

	instance.PodName = podName
	instance.ServiceName = serviceName
	instance.ExternalPort = externalPort
	instance.InternalPort = challenge.Port
	instance.Namespace = k.namespace

	if err := k.createDeployment(ctx, instance, challenge); err != nil {
		return fmt.Errorf("failed to create deployment: %v", err)
	}

	if err := k.createService(ctx, instance); err != nil {
		k.deleteDeployment(ctx, podName)
		return fmt.Errorf("failed to create service: %v", err)
	}

	clusterIP := os.Getenv("CLUSTER_IP")
	if clusterIP == "" {
		clusterIP = "192.168.1.34"
	}
	instance.AccessURL = fmt.Sprintf("http://%s:%d", clusterIP, externalPort)

	log.Printf("Created instance for user %d, challenge %d: %s", instance.UserID, instance.ChallengeID, instance.AccessURL)
	return nil
}

func (k *KubernetesService) createDeployment(ctx context.Context, instance *models.Instance, challenge *models.Challenge) error {
	labels := map[string]string{
		"app":          "ctf-challenge",
		"user-id":      strconv.Itoa(instance.UserID),
		"challenge-id": strconv.Itoa(instance.ChallengeID),
		"instance-id":  strconv.Itoa(instance.ID),
	}

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      instance.PodName,
			Namespace: k.namespace,
			Labels:    labels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: int32Ptr(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "challenge",
							Image: challenge.DockerImage,
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: int32(challenge.Port),
									Protocol:      corev1.ProtocolTCP,
								},
							},
							Resources: corev1.ResourceRequirements{
								Limits: corev1.ResourceList{
									corev1.ResourceCPU:    parseQuantity(challenge.CPULimit, "500m"),
									corev1.ResourceMemory: parseQuantity(challenge.MemoryLimit, "512Mi"),
								},
								Requests: corev1.ResourceList{
									corev1.ResourceCPU:    parseQuantity(challenge.CPULimit, "100m"),
									corev1.ResourceMemory: parseQuantity(challenge.MemoryLimit, "256Mi"),
								},
							},
							SecurityContext: &corev1.SecurityContext{
								AllowPrivilegeEscalation: boolPtr(false),
								RunAsNonRoot:             boolPtr(true),
								RunAsUser:                int64Ptr(1000),
								ReadOnlyRootFilesystem:   boolPtr(false),
							},
						},
					},
					RestartPolicy: corev1.RestartPolicyAlways,
					SecurityContext: &corev1.PodSecurityContext{
						RunAsNonRoot: boolPtr(true),
						RunAsUser:    int64Ptr(1000),
					},
				},
			},
		},
	}

	_, err := k.clientset.AppsV1().Deployments(k.namespace).Create(ctx, deployment, metav1.CreateOptions{})
	return err
}

func (k *KubernetesService) createService(ctx context.Context, instance *models.Instance) error {
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      instance.ServiceName,
			Namespace: k.namespace,
			Labels: map[string]string{
				"app":         "ctf-challenge",
				"instance-id": strconv.Itoa(instance.ID),
			},
		},
		Spec: corev1.ServiceSpec{
			Type: corev1.ServiceTypeNodePort,
			Ports: []corev1.ServicePort{
				{
					Port:       int32(instance.InternalPort),
					TargetPort: intstr.FromInt(instance.InternalPort),
					NodePort:   int32(instance.ExternalPort),
					Protocol:   corev1.ProtocolTCP,
				},
			},
			Selector: map[string]string{
				"app":         "ctf-challenge",
				"instance-id": strconv.Itoa(instance.ID),
			},
		},
	}

	_, err := k.clientset.CoreV1().Services(k.namespace).Create(ctx, service, metav1.CreateOptions{})
	return err
}

func (k *KubernetesService) DeleteInstance(instance *models.Instance) error {
	ctx := context.Background()

	if instance.ServiceName != "" {
		err := k.clientset.CoreV1().Services(k.namespace).Delete(ctx, instance.ServiceName, metav1.DeleteOptions{})
		if err != nil {
			log.Printf("Failed to delete service %s: %v", instance.ServiceName, err)
		}
	}

	if instance.PodName != "" {
		err := k.deleteDeployment(ctx, instance.PodName)
		if err != nil {
			log.Printf("Failed to delete deployment %s: %v", instance.PodName, err)
		}
	}

	log.Printf("Deleted instance %d resources", instance.ID)
	return nil
}

func (k *KubernetesService) deleteDeployment(ctx context.Context, name string) error {
	deletePolicy := metav1.DeletePropagationForeground
	return k.clientset.AppsV1().Deployments(k.namespace).Delete(ctx, name, metav1.DeleteOptions{
		PropagationPolicy: &deletePolicy,
	})
}

// ✅ Nouvelle méthode ajoutée ici
func (k *KubernetesService) CleanupChallengeInstance(instance *models.Instance) error {
	return k.DeleteInstance(instance)
}

// ---------- Helper Functions ----------

func int32Ptr(i int32) *int32 {
	return &i
}

func int64Ptr(i int64) *int64 {
	return &i
}

func boolPtr(b bool) *bool {
	return &b
}

func parseQuantity(value string, defaultVal string) resource.Quantity {
	qty, err := resource.ParseQuantity(value)
	if err != nil {
		qty, _ = resource.ParseQuantity(defaultVal)
	}
	return qty
}
