package main

import (
	"fmt"
	"os"
	"strings"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func main() {
	fmt.Println("Hello, creating a basic Kube Job to run mvn on a Liberty Project.")
	// creates the in-cluster config
	config, err := rest.InClusterConfig()
	if err != nil {
		panic(err.Error())
	}
	// creates the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	if len(os.Args) != 3 {
		fmt.Println("Wrong usage of idcbuildtask tool...")
		fmt.Println("Usage:")
		fmt.Println("Argument 1: Project IDP Build Task Name")
		fmt.Println("Argument 2: Project Name")
		os.Exit(1)
	}

	taskName := os.Args[1]
	fmt.Printf("The taskName chosen: %s\n", taskName)

	projectName := os.Args[2]
	fmt.Printf("The projectName chosen: %s\n", projectName)

	buildTaskJob1 := "codewind-liberty-build-job"

	cheWorkspaceID := os.Getenv("CHE_WORKSPACE_ID")
	if cheWorkspaceID == "" {
		fmt.Println("Che Workspace ID not set and unable to run the IDP Kube Job, exiting...")
		os.Exit(1)
	} else {
		fmt.Printf("The Che Workspace ID: %s\n", cheWorkspaceID)
	}

	namespace := GetCurrentNamespace()
	fmt.Printf("The Che Codewind Namespace: %s\n", namespace)

	workspacePVC := GetWorkspacePVC(clientset, namespace, cheWorkspaceID)
	fmt.Printf("The Che Codewind PVC: %s\n", workspacePVC)

	err = CreateBuildTaskKubeJob(clientset, buildTaskJob1, namespace, cheWorkspaceID, workspacePVC, taskName, projectName)
	if err != nil {
		fmt.Println("Failed to create a job, exiting...")
		panic(err.Error())
	}

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
					loop = false
				} else if failed > 0 {
					fmt.Printf("The job %s failed\n", job.Name)
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

	fmt.Println("And that's it folks...")
}

// CreateBuildTaskKubeJob creates a Kubernetes Job
func CreateBuildTaskKubeJob(clientset *kubernetes.Clientset, buildTaskJob string, namespace string, cheWorkspaceID string, workspacePVC string, taskName string, projectName string) error {
	fmt.Printf("Creating job %s\n", buildTaskJob)
	// Create a Kube job to run mvn compile for a Liberty project
	mvnCommand := "echo listing /home/default/app && ls -la /home/default/app && echo copying /home/default/app /tmp/app && cp -rf /home/default/app /tmp/app && cd /tmp/app && echo chown, listing and running mvn in /tmp/app: && chown -fR 1001 /tmp/app && ls -la && mvn -B clean package -DskipTests=true -DlibertyEnv=microclimate -DmicroclimateOutputDir=/tmp/app/mc-target --log-file /home/default/app/maven.package.test.log && echo listing after mvn && ls -la && echo copying tmp/app/mc-target to /home/default/app && cp -rf /tmp/app/mc-target /home/default/app/ && chown -fR 1001 /home/default/app/mc-target && echo listing /home/default/app && ls -la /home/default/app/"

	if taskName == "package" {
		mvnCommand = "echo listing /home/default/app && ls -la /home/default/app && echo copying /home/default/app /tmp/app && cp -rf /home/default/app /tmp/app && cd /tmp/app && echo chown, listing and running mvn in /tmp/app: && chown -fR 1001 /tmp/app && ls -la && mvn -B clean package liberty:install-apps -DskipTests=true -DlibertyEnv=microclimate -DmicroclimateOutputDir=/tmp/app/mc-target --log-file /home/default/app/maven.package.test.log && echo listing after mvn && ls -la && echo copying tmp/app/mc-target to /home/default/app && cp -rf /tmp/app/mc-target /home/default/app/ && chown -fR 1001 /home/default/app/mc-target && echo listing /home/default/app && ls -la /home/default/app/"
	}

	fmt.Printf("Mvn Command: %s\n", mvnCommand)

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      buildTaskJob,
			Namespace: namespace,
		},
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Volumes: []corev1.Volume{
						{
							Name: "liberty-project",
							VolumeSource: corev1.VolumeSource{
								PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
									ClaimName: workspacePVC,
								},
							},
						},
					},
					Containers: []corev1.Container{
						{
							Name:            "maven-build",
							Image:           "docker.io/maven:3.6",
							ImagePullPolicy: corev1.PullAlways,
							Command:         []string{"/bin/sh", "-c"},
							Args:            []string{mvnCommand},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "liberty-project",
									MountPath: "/home/default/app",
									SubPath:   cheWorkspaceID + "/projects/" + projectName,
								},
							},
						},
					},
					RestartPolicy: corev1.RestartPolicyNever,
				},
			},
		},
	}
	kubeJob, err := clientset.BatchV1().Jobs(namespace).Create(job)
	if err != nil {
		return err
	}

	fmt.Printf("The job %s has been created\n", kubeJob.Name)
	return nil
}

// GetWorkspacePVC retrieves the PVC (Persistent Volume Claim) associated with the Che workspace we're deploying Codewind in
func GetWorkspacePVC(clientset *kubernetes.Clientset, namespace string, cheWorkspaceID string) string {
	var pvcName string

	PVCs, err := clientset.CoreV1().PersistentVolumeClaims(namespace).List(metav1.ListOptions{
		LabelSelector: "che.workspace.volume_name=projects,che.workspace_id=" + cheWorkspaceID,
	})
	if err != nil || PVCs == nil {
		fmt.Printf("Error, unable to retrieve PVCs: %v\n", err)
		os.Exit(1)
	} else if len(PVCs.Items) < 1 {
		// We couldn't find the workspace PVC, so need to find an alternative.
		PVCs, err = clientset.CoreV1().PersistentVolumeClaims(namespace).List(metav1.ListOptions{
			LabelSelector: "che.workspace_id=" + cheWorkspaceID,
		})
		if err != nil || PVCs == nil {
			fmt.Printf("Error, unable to retrieve PVCs: %v\n", err)
			os.Exit(1)
		} else if len(PVCs.Items) != 1 {
			pvcName = "claim-che-workspace"
		} else {
			pvcName = PVCs.Items[0].GetName()
		}
	} else {
		pvcName = PVCs.Items[0].GetName()
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
