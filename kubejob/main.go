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

	arg := os.Args[1]
	fmt.Printf("The command chosen: %s\n", arg)

	buildTaskJob1 := "codewind-liberty-build-job"
	namespace := "eclipse-che"

	err = CreateBuildTaskKubeJob(clientset, buildTaskJob1, namespace, arg)
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

	/*pods, err := clientset.CoreV1().Pods("eclipse-che").List(metav1.ListOptions{})
	// _, err = clientset.CoreV1().Pods("default").Get("example-xxxxx", metav1.GetOptions{})
	if err != nil {
		panic(err.Error())
	}

	for _, pod := range pods.Items {
		if strings.Contains(pod.Name, "codewind-liberty-build-job") {
			fmt.Println(pod.Name)
			fmt.Println(pod.Status.Phase)
			fmt.Println(pod.Status.ContainerStatuses[0].State.Terminated.Reason)
		}
	}*/

	// namespace := GetCurrentNamespace()
	// fmt.Println("Hello,", namespace)
}

// CreateBuildTaskKubeJob creates a Kubernetes Job
func CreateBuildTaskKubeJob(clientset *kubernetes.Clientset, buildTaskJob string, namespace string, command string) error {
	fmt.Printf("Creating job %s\n", buildTaskJob)
	// Create a Kube job to run mvn compile for a Liberty project
	mvnCommand := "cd /home/default/app && ls -la && mvn -B compile -DskipTests=true -DlibertyEnv=microclimate -DmicroclimateOutputDir=/home/default/app/mc-target --log-file /home/default/app/maven.compile.test.log && chown -R 1001 /home/default/app/"

	if command == "package" {
		mvnCommand = "cd /home/default/app && ls -la && mvn -B package -DskipTests=true -DlibertyEnv=microclimate -DmicroclimateOutputDir=/home/default/app/mc-target --log-file /home/default/app/maven.package.test.log && chown -R 1001 /home/default/app/"
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
									ClaimName: "claim-che-workspace6kfl7t0q",
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
									SubPath:   "workspace05bhbslshm9de3jl/projects/arandomjava1",
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
