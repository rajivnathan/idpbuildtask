package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

//Microprofile is a struct of essential data
type BuildTask struct {
	Name               string
	Image              string
	Namespace          string
	WorkspaceID        string
	PVCName            string
	ServiceAccountName string
	PullSecret         string
	OwnerReferenceName string
	OwnerReferenceUID  types.UID
	Privileged         bool
	Ingress            string
}

func main() {
	if len(os.Args) != 3 {
		fmt.Println("Wrong usage of idcbuildtask tool...")
		fmt.Println("Usage:")
		fmt.Println("Argument 1: Project IDP Build Task Name")
		fmt.Println("Argument 2: Project Name")
		os.Exit(1)
	}

	fmt.Println("Hello, creating a basic Kube Job to run mvn on a Liberty Project.")

	// creates the in-cluster config
	config, err := rest.InClusterConfig()
	if err != nil {
		fmt.Println("Not running in a cluster. Attempting to load kube context from local kubeconfig")
		// initialize client-go clients
		loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
		configOverrides := &clientcmd.ConfigOverrides{}
		kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides)

		clientConfig, err := kubeConfig.ClientConfig()
		if err != nil {
			panic(err.Error())
		}
		config = clientConfig

	}
	// creates the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	taskName := os.Args[1]
	fmt.Printf("The taskName chosen: %s\n", taskName)

	projectName := os.Args[2]
	fmt.Printf("The projectName chosen: %s\n", projectName)

	buildTaskJob1 := "codewind-liberty-build-job"

	namespace := GetCurrentNamespace()
	fmt.Printf("Current namespace: %s\n", namespace)

	idpClaimName := GetIDPPVC(clientset, namespace, "app=idp")
	fmt.Printf("Persistent Volume Claim: %s\n", idpClaimName)

	serviceAccountName := "default"
	fmt.Printf("Service Account: %s\n", serviceAccountName)

	job, err := CreateBuildTaskKubeJob(buildTaskJob1, taskName, namespace, idpClaimName, "projects/"+projectName, projectName)
	if err != nil {
		fmt.Println("There was a problem with the job configuration, exiting...")
		panic(err.Error())
	}

	kubeJob, err := clientset.BatchV1().Jobs(namespace).Create(job)
	if err != nil {
		fmt.Println("Failed to create a job, exiting...")
		panic(err.Error())
	}

	fmt.Printf("The job %s has been created\n", kubeJob.Name)

	// Wait for pods to start running so that we can tail the logs
	fmt.Printf("Waiting for pod to run\n")
	foundRunningPod := false
	for foundRunningPod == false {

		podList, err := clientset.CoreV1().Pods(namespace).List(metav1.ListOptions{
			LabelSelector: "job-name=codewind-liberty-build-job",
			FieldSelector: "status.phase=Running",
		})

		if err != nil {
			continue
		}

		for _, pod := range podList.Items {
			fmt.Printf("Running pod found: %s Retrieving logs...\n\n", pod.Name)
			foundRunningPod = true
		}
	}

	// Print logs before deleting the job
	podList, err := clientset.CoreV1().Pods(namespace).List(metav1.ListOptions{
		LabelSelector: "job-name=codewind-liberty-build-job",
	})

	for _, pod := range podList.Items {
		fmt.Printf("Retrieving logs for pod: %s\n\n", pod.Name)
		req := clientset.CoreV1().Pods(namespace).GetLogs(pod.Name, &corev1.PodLogOptions{
			Follow: true,
		})
		readCloser, err := req.Stream()
		if err != nil {
			fmt.Printf("Unable to retrieve logs for pod: %s\n", pod.Name)
			continue
		}

		defer readCloser.Close()
		_, err = io.Copy(os.Stdout, readCloser)
	}

	// TODO: Set owner references
	var jobSucceeded bool
	// Loop and see if the job either succeeded or failed
	loop := true
	for loop == true {
		jobs, err := clientset.BatchV1().Jobs(namespace).List(metav1.ListOptions{})
		if err != nil {
			panic(err.Error())
		}
		for _, job := range jobs.Items {
			if strings.Contains(job.Name, buildTaskJob1) {
				succeeded := job.Status.Succeeded
				failed := job.Status.Failed
				if succeeded == 1 {
					fmt.Printf("The job %s succeeded\n", job.Name)
					jobSucceeded = true
					loop = false
				} else if failed > 0 {
					fmt.Printf("The job %s failed\n", job.Name)
					jobSucceeded = false
					loop = false
				}
			}
		}
	}

	if loop == false {
		// delete the job
		gracePeriodSeconds := int64(0)
		deletionPolicy := metav1.DeletePropagationForeground
		err := clientset.BatchV1().Jobs(namespace).Delete(buildTaskJob1, &metav1.DeleteOptions{
			PropagationPolicy:  &deletionPolicy,
			GracePeriodSeconds: &gracePeriodSeconds,
		})
		if err != nil {
			panic(err.Error())
		} else {
			fmt.Printf("The job %s has been deleted\n", buildTaskJob1)
		}
	}

	if !jobSucceeded {
		fmt.Println("The job failed, exiting...")
		os.Exit(1)
	}

	// Create the Codewind deployment object
	BuildTaskInstance := BuildTask{
		Name:               "cw-maysunliberty2-6c1b1ce0-cb4c-11e9-be96",
		Image:              "websphere-liberty:19.0.0.3-webProfile7",
		Namespace:          namespace,
		PVCName:            idpClaimName,
		ServiceAccountName: serviceAccountName,
		// OwnerReferenceName: ownerReferenceName,
		// OwnerReferenceUID:  ownerReferenceUID,
		Privileged: true,
	}

	if taskName == "full" {
		// Deploy Application
		deploy := createPFEDeploy(BuildTaskInstance, projectName)
		service := createPFEService(BuildTaskInstance)

		fmt.Println("===============================")
		fmt.Println("Deploying application...")
		_, err = clientset.CoreV1().Services(namespace).Create(&service)
		if err != nil {
			fmt.Printf("Unable to create application service: %v\n", err)
			os.Exit(1)
		}
		_, err = clientset.AppsV1().Deployments(namespace).Create(&deploy)
		if err != nil {
			fmt.Printf("Unable to create application deployment: %v\n", err)
			os.Exit(1)
		}
	}

	fmt.Println("And that's it folks...")
}

// GetIDPPVC retrieves the PVC (Persistent Volume Claim) associated with the Iterative Development Pack
func GetIDPPVC(clientset *kubernetes.Clientset, namespace string, labels string) string {
	var pvcName string

	PVCs, err := clientset.CoreV1().PersistentVolumeClaims(namespace).List(metav1.ListOptions{
		LabelSelector: labels,
	})
	if err != nil || PVCs == nil {
		fmt.Printf("Error, unable to retrieve PVCs: %v\n", err)
		os.Exit(1)
	} else if len(PVCs.Items) == 1 {
		pvcName = PVCs.Items[0].GetName()
	} else {
		// We couldn't find the workspace PVC, use a default value
		pvcName = "claim-che-workspace"
	}

	return pvcName
}

// GetKubeClientConfig retrieves the Kubernetes client config from the cluster
func GetKubeClientConfig() clientcmd.ClientConfig {
	// Retrieve the Kube client config
	clientconfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		clientcmd.NewDefaultClientConfigLoadingRules(),
		&clientcmd.ConfigOverrides{},
	)
	return clientconfig
}

// GetCurrentNamespace gets the current namespace in the Kubernetes context
func GetCurrentNamespace() string {
	// Instantiate loader for kubeconfig file.
	kubeconfig := GetKubeClientConfig()
	namespace, _, err := kubeconfig.Namespace()
	if err != nil {
		panic(err)
	}
	return namespace
}
