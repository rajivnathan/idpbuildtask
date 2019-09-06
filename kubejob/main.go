package main

import (
	"fmt"
	"os"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

//Codewind is a struct of essential data
type Codewind struct {
	PFEName            string
	PerformanceName    string
	PFEImage           string
	PerformanceImage   string
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
	// runTaskJob1 := "codewind-liberty-run-job"

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
	fmt.Printf("The IDP Vol PVC: %s\n", workspacePVC)

	serviceAccountName := GetWorkspaceServiceAccount(clientset, namespace, cheWorkspaceID)
	fmt.Printf("The Che Service Account: %s\n", serviceAccountName)

	// Get the Owner reference name and uid
	ownerReferenceName, ownerReferenceUID := GetOwnerReferences(clientset, namespace, cheWorkspaceID)
	fmt.Printf("The Che ownerReferenceName: %s\n", ownerReferenceName)
	fmt.Printf("The Che ownerReferenceUID: %s\n", ownerReferenceUID)

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

	// Create the Codewind deployment object
	codewindInstance := Codewind{
		PFEName:            "cw-maysunliberty2-6c1b1ce0-cb4c-11e9-be96",
		PFEImage:           "websphere-liberty:19.0.0.3-webProfile7",
		Namespace:          namespace,
		WorkspaceID:        cheWorkspaceID,
		PVCName:            workspacePVC,
		ServiceAccountName: serviceAccountName,
		OwnerReferenceName: ownerReferenceName,
		OwnerReferenceUID:  ownerReferenceUID,
		Privileged:         true,
	}

	if taskName == "full" {
		// Deploy Application
		service := createPFEService(codewindInstance)
		deploy := createPFEDeploy(codewindInstance)

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

	// err = CreateRunTaskKubeJob(clientset, runTaskJob1, namespace, cheWorkspaceID, workspacePVC, taskName, projectName)
	// if err != nil {
	// 	fmt.Println("Failed to create a job, exiting...")
	// 	panic(err.Error())
	// }

	// // Loop and see if the job either succeeded or failed
	// loop = true
	// for loop == true {
	// 	jobs, err := clientset.BatchV1().Jobs(namespace).List(metav1.ListOptions{})
	// 	if err != nil {
	// 		panic(err.Error())
	// 	}
	// 	for _, job := range jobs.Items {
	// 		if strings.Contains(job.Name, runTaskJob1) {
	// 			succeeded := job.Status.Succeeded
	// 			failed := job.Status.Failed
	// 			if succeeded == 1 {
	// 				fmt.Printf("The job %s succeeded\n", job.Name)
	// 				loop = false
	// 			} else if failed > 0 {
	// 				fmt.Printf("The job %s failed\n", job.Name)
	// 				loop = false
	// 			}
	// 		}
	// 	}
	// }

	// if loop == false {
	// 	// delete the job
	// 	gracePeriodSeconds := int64(0)
	// 	deletionPolicy := metav1.DeletePropagationForeground
	// 	err := clientset.BatchV1().Jobs(namespace).Delete(runTaskJob1, &metav1.DeleteOptions{
	// 		PropagationPolicy:  &deletionPolicy,
	// 		GracePeriodSeconds: &gracePeriodSeconds,
	// 	})
	// 	if err != nil {
	// 		panic(err.Error())
	// 	} else {
	// 		fmt.Printf("The job %s has been deleted\n", runTaskJob1)
	// 	}
	// }

	fmt.Println("And that's it folks...")
}

// CreateBuildTaskKubeJob creates a Build Task Kubernetes Job
func CreateBuildTaskKubeJob(clientset *kubernetes.Clientset, jobName string, namespace string, cheWorkspaceID string, workspacePVC string, taskName string, projectName string) error {
	fmt.Printf("Creating job %s\n", jobName)
	// Create a Kube job to run mvn compile for a Liberty project
	// mvnCommand := "echo listing /home/default/app && ls -la /home/default/app && echo copying /home/default/app /tmp/app && cp -rf /home/default/app /tmp/app && cd /tmp/app && echo chown, listing and running mvn in /tmp/app: && chown -fR 1001 /tmp/app && ls -la && mvn -B clean package -Dmaven.repo.local=/tmp/app/.m2/repository -DskipTests=true -DlibertyEnv=microclimate -DmicroclimateOutputDir=/tmp/app/mc-target --log-file /home/default/app/maven.package.test.log && echo listing after mvn && ls -la && echo copying tmp/app/mc-target to /home/default/app && cp -rf /tmp/app/mc-target /home/default/app/ && chown -fR 1001 /home/default/app/mc-target && echo listing /home/default/app && ls -la /home/default/app/"

	mvnCommand := "echo changing dir to /home/default/app && cd /home/default/app && echo chown, listing and running mvn in /home/default/app: && chown -fR 1001 /home/default/app && ls -la && mvn -B clean package -Dmaven.repo.local=/home/default/app/.m2/repository -DskipTests=true -DlibertyEnv=microclimate -DmicroclimateOutputDir=/home/default/app/buildoutput --log-file /home/default/app/maven.package.test.log && chown -fR 1001 /home/default/app/buildoutput && echo listing /home/default/app after mvn and chown 1001 buildoutput && ls -la && echo rm -rf /home/default/app/buildartifacts and copying artifacts && rm -rf /home/default/app/buildartifacts && mkdir -p /home/default/app/buildartifacts/ && cp -r /home/default/app/buildoutput/liberty/wlp/usr/servers/defaultServer/* /home/default/app/buildartifacts/ && cp -r /home/default/app/buildoutput/liberty/wlp/usr/shared/resources/ /home/default/app/buildartifacts/ && cp /home/default/app/src/main/liberty/config/jvmbx.options /home/default/app/buildartifacts/jvm.options && echo chown the buildartifacts dir && chown -fR 1001 /home/default/app/buildartifacts"

	if taskName == "inc" {
		mvnCommand = "echo changing dir to /home/default/app && cd /home/default/app && echo chown, listing and running mvn in /home/default/app: && chown -fR 1001 /home/default/app && ls -la && mvn -B clean package -Dmaven.repo.local=/home/default/app/.m2/repository -DskipTests=true -DlibertyEnv=microclimate -DmicroclimateOutputDir=/home/default/app/buildoutput --log-file /home/default/app/maven.package.test.log && chown -fR 1001 /home/default/app/buildoutput && echo listing /home/default/app after mvn and chown 1001 buildoutput && ls -la && echo copying artifacts && cp -rf /home/default/app/buildoutput/liberty/wlp/usr/servers/defaultServer/apps/* /home/default/app/buildartifacts/apps/ && echo chown the buildartifacts apps dir && chown -fR 1001 /home/default/app/buildartifacts/apps"
	}

	fmt.Printf("Mvn Command: %s\n", mvnCommand)

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jobName,
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
									// SubPath:   cheWorkspaceID + "/projects/" + projectName,
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

// // CreateRunTaskKubeJob creates a Run Task Kubernetes Job
// func CreateRunTaskKubeJob(clientset *kubernetes.Clientset, buildTaskJob string, namespace string, cheWorkspaceID string, workspacePVC string, taskName string, projectName string) error {
// 	fmt.Printf("Creating job %s\n", buildTaskJob)

// 	runCommand := "echo changing dir to /home/default/app && cd /home/default/app && "

// 	fmt.Printf("Mvn Command: %s\n", mvnCommand)

// 	job := &batchv1.Job{
// 		ObjectMeta: metav1.ObjectMeta{
// 			Name:      buildTaskJob,
// 			Namespace: namespace,
// 		},
// 		Spec: batchv1.JobSpec{
// 			Template: corev1.PodTemplateSpec{
// 				Spec: corev1.PodSpec{
// 					Volumes: []corev1.Volume{
// 						{
// 							Name: "liberty-project",
// 							VolumeSource: corev1.VolumeSource{
// 								PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
// 									ClaimName: workspacePVC,
// 								},
// 							},
// 						},
// 					},
// 					Containers: []corev1.Container{
// 						{
// 							Name:            "maven-build",
// 							Image:           "docker.io/maven:3.6",
// 							ImagePullPolicy: corev1.PullAlways,
// 							Command:         []string{"/bin/sh", "-c"},
// 							Args:            []string{mvnCommand},
// 							VolumeMounts: []corev1.VolumeMount{
// 								{
// 									Name:      "liberty-project",
// 									MountPath: "/home/default/app",
// 									// SubPath:   cheWorkspaceID + "/projects/" + projectName,
// 								},
// 							},
// 						},
// 					},
// 					RestartPolicy: corev1.RestartPolicyNever,
// 				},
// 			},
// 		},
// 	}
// 	kubeJob, err := clientset.BatchV1().Jobs(namespace).Create(job)
// 	if err != nil {
// 		return err
// 	}

// 	fmt.Printf("The job %s has been created\n", kubeJob.Name)
// 	return nil
// }

// GetWorkspacePVC retrieves the PVC (Persistent Volume Claim) associated with the Che workspace we're deploying Codewind in
func GetWorkspacePVC(clientset *kubernetes.Clientset, namespace string, cheWorkspaceID string) string {
	var pvcName string

	PVCs, err := clientset.CoreV1().PersistentVolumeClaims(namespace).List(metav1.ListOptions{
		LabelSelector: "artifact=codewindidpvolume",
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

// GetOwnerReferences retrieves the owner reference name and UID, allowing us to tie any Codewind resources to the Che workspace
// Enabling the Kubernetes garbage collector clean everything up when the workspace is deleted
func GetOwnerReferences(clientset *kubernetes.Clientset, namespace string, cheWorkspaceID string) (string, types.UID) {
	// Get the Workspace pod
	var ownerReferenceName string
	var ownerReferenceUID types.UID

	workspacePod, err := clientset.CoreV1().Pods(namespace).List(metav1.ListOptions{
		LabelSelector: "che.original_name=che-workspace-pod,che.workspace_id=" + cheWorkspaceID,
	})
	if err != nil {
		fmt.Printf("Error: Unable to retrieve the workspace pod %v\n", err)
		os.Exit(1)
	}
	// Retrieve the owner reference name and UID from the workspace pod. This will allow Codewind to be garbage collected by Kube
	ownerReferenceName = workspacePod.Items[0].GetOwnerReferences()[0].Name
	ownerReferenceUID = workspacePod.Items[0].GetOwnerReferences()[0].UID

	return ownerReferenceName, ownerReferenceUID
}

// GetWorkspaceServiceAccount retrieves the Service Account associated with the Che workspace we're deploying Codewind in
func GetWorkspaceServiceAccount(clientset *kubernetes.Clientset, namespace string, cheWorkspaceID string) string {
	var serviceAccountName string

	// Retrieve the workspace service account labeled with the Che Workspace ID
	workspacePod, err := clientset.CoreV1().Pods(namespace).List(metav1.ListOptions{
		LabelSelector: "che.original_name=che-workspace-pod,che.workspace_id=" + cheWorkspaceID,
	})
	if err != nil || workspacePod == nil {
		fmt.Printf("Error retrieving the Che workspace pod %v\n", err)
		os.Exit(1)
	} else if len(workspacePod.Items) != 1 {
		// Default to che-workspace as the Service Account name if one couldn't be found
		serviceAccountName = "che-workspace"
	} else {
		serviceAccountName = workspacePod.Items[0].Spec.ServiceAccountName
	}

	return serviceAccountName

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

// createPFEDeploy creates a Kubernetes deploy for Codewind, marking the Che workspace as its owner
func createPFEDeploy(codewind Codewind) appsv1.Deployment {
	labels := map[string]string{
		"chart":   "javamicroprofiletemplate-1.0.0",
		"release": codewind.PFEName,
	}

	volumes, volumeMounts := setPFEVolumes(codewind)
	envVars := setPFEEnvVars(codewind)

	return generateDeployment(codewind, "javamicroprofiletemplate", codewind.PFEImage, volumes, volumeMounts, envVars, labels)
}

// createPFEService creates a Kubernetes service for Codewind, exposing port 9191
func createPFEService(codewind Codewind) corev1.Service {
	labels := map[string]string{
		"chart":   "javamicroprofiletemplate-1.0.0",
		"release": codewind.PFEName,
	}
	return generateService(codewind, labels)
}

// setPFEVolumes returns the 3 volumes & corresponding volume mounts required by the PFE container:
// project workspace, buildah volume, and the docker registry secret (the latter of which is optional)
func setPFEVolumes(codewind Codewind) ([]corev1.Volume, []corev1.VolumeMount) {

	volumes := []corev1.Volume{
		{
			Name: "idp-volume",
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: codewind.PVCName,
				},
			},
		},
	}

	volumeMounts := []corev1.VolumeMount{
		{
			Name:      "idp-volume",
			MountPath: "/config",
			SubPath:   "buildartifacts/",
		},
	}

	return volumes, volumeMounts
}

func setPFEEnvVars(codewind Codewind) []corev1.EnvVar {
	booleanTrue := bool(true)

	return []corev1.EnvVar{
		{
			Name:  "PORT",
			Value: "9080",
		},
		{
			Name:  "APPLICATION_NAME",
			Value: "cw-maysunliberty2-6c1b1ce0-cb4c-11e9-be96",
		},
		{
			Name:  "PROJECT_NAME",
			Value: "maysunliberty2",
		},
		{
			Name:  "LOG_FOLDER",
			Value: "maysunliberty2-6c1b1ce0-cb4c-11e9-be96-bfc50f05726d",
		},
		{
			Name:  "IN_K8",
			Value: "true",
		},
		{
			Name: "IBM_APM_SERVER_URL",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: "apm-server-config",
					},
					Key:      "ibm_apm_server_url",
					Optional: &booleanTrue,
				},
			},
		},
		{
			Name: "IBM_APM_KEYFILE",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: "apm-server-config",
					},
					Key:      "ibm_apm_keyfile_password",
					Optional: &booleanTrue,
				},
			},
		},
		{
			Name: "IBM_APM_INGRESS_URL",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: "apm-server-config",
					},
					Key:      "ibm_apm_ingress_url",
					Optional: &booleanTrue,
				},
			},
		},
		{
			Name: "IBM_APM_KEYFILE_PASSWORD",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: "apm-server-config",
					},
					Key:      "ibm_apm_keyfile_password",
					Optional: &booleanTrue,
				},
			},
		},
		{
			Name: "IBM_APM_ACCESS_TOKEN",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: "apm-server-config",
					},
					Key:      "ibm_apm_access_token",
					Optional: &booleanTrue,
				},
			},
		},
	}
}

// generateDeployment returns a Kubernetes deployment object with the given name for the given image.
// Additionally, volume/volumemounts and env vars can be specified.
func generateDeployment(codewind Codewind, name string, image string, volumes []corev1.Volume, volumeMounts []corev1.VolumeMount, envVars []corev1.EnvVar, labels map[string]string) appsv1.Deployment {
	blockOwnerDeletion := true
	controller := true
	replicas := int32(1)
	labels2 := map[string]string{
		"app":     "libertyidp-selector",
		"version": "current",
		"release": codewind.PFEName,
	}
	deployment := appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      codewind.PFEName,
			Namespace: codewind.Namespace,
			Labels:    labels,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion:         "apps/v1",
					BlockOwnerDeletion: &blockOwnerDeletion,
					Controller:         &controller,
					Kind:               "ReplicaSet",
					Name:               codewind.OwnerReferenceName,
					UID:                codewind.OwnerReferenceUID,
				},
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels2,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels2,
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: codewind.ServiceAccountName,
					Volumes:            volumes,
					Containers: []corev1.Container{
						{
							Name:            name,
							Image:           image,
							ImagePullPolicy: corev1.PullAlways,
							SecurityContext: &corev1.SecurityContext{
								Privileged: &codewind.Privileged,
							},
							VolumeMounts: volumeMounts,
							Command:      []string{"/usr/bin/tail"},
							Args:         []string{"-f", "/dev/null"},
							Lifecycle: &corev1.Lifecycle{
								PostStart: &corev1.Handler{
									Exec: &corev1.ExecAction{
										Command: []string{"/bin/bash", "-c", "/opt/ibm/wlp/bin/server start"},
									},
								},
							},
							Env: envVars,
						},
					},
				},
			},
		},
	}
	return deployment
}

// generateService returns a Kubernetes service object with the given name, exposed over the specified port
// for the container with the given labels.
func generateService(codewind Codewind, labels map[string]string) corev1.Service {
	blockOwnerDeletion := true
	controller := true

	port1 := 9080
	port2 := 9443

	service := corev1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      codewind.PFEName,
			Namespace: codewind.Namespace,
			Labels:    labels,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion:         "apps/v1",
					BlockOwnerDeletion: &blockOwnerDeletion,
					Controller:         &controller,
					Kind:               "ReplicaSet",
					Name:               codewind.OwnerReferenceName,
					UID:                codewind.OwnerReferenceUID,
				},
			},
		},
		Spec: corev1.ServiceSpec{
			Type: corev1.ServiceTypeNodePort,
			Ports: []corev1.ServicePort{
				{
					Port: int32(port1),
					Name: "http",
				},
				{
					Port: int32(port2),
					Name: "https",
				},
			},
			Selector: map[string]string{
				"app": "libertyidp-selector",
			},
		},
	}
	return service
}
