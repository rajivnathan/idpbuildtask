package main

import (
	"fmt"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CreateBuildTaskKubeJob creates a Kubernetes Job
func CreateBuildTaskKubeJob(buildTaskJob string, taskName string, namespace string, idpClaimName string, projectSubPath string, projectName string) (*batchv1.Job, error) {
	fmt.Printf("Creating job %s\n", buildTaskJob)
	// Create a Kube job to run mvn compile for a Liberty project
	var mvnCommand string
	if taskName == "update" {
		// mvnCommand = "echo listing /data/idp/src && ls -la /data/idp/src && echo copying /data/idp/src to /tmp/app && cp -rf /data/idp/src /tmp/app && echo chown, listing and running mvn in /tmp/app: && id && chown -R 1001 /tmp/app && cd /tmp/app && echo before maven build && ls -la && mvn -B compile war:war -Dmaven.repo.local=/data/idp/cache/.m2/repository -DskipTests=true && echo listing target contents after mvn && ls -la target && echo copying tmp/app/target to /data/idp/output && cp -rf /tmp/app/target /data/idp/output && chown -fR 1001 /data/idp/output && echo listing /data/idp/output after mvn and chown 1001 buildoutput && ls -la /data/idp/output && echo copying artifacts && mkdir -p /data/idp/buildartifacts/ && echo chown the buildartifacts dir && chown -fR 1001 /data/idp/buildartifacts"
		// mvnCommand = "echo listing /data/idp/src && ls -la /data/idp/src && echo chown, listing and running mvn in /data/idp/src && chown -R 1001 /data/idp/src && cd /data/idp/src && echo list contents before maven build && ls -la && mvn -B compile war:war -Dmaven.repo.local=/data/idp/cache/.m2/repository -DskipTests=true && echo listing target contents after mvn && ls -la target && cp target/mpnew-1.0-SNAPSHOT.war /data/idp/buildartifacts/apps  && echo chown the buildartifacts dir && chown 1001 /data/idp/buildartifacts/apps/mpnew-1.0-SNAPSHOT.war && echo list /data/idp/buildartifacts/apps && ls -al /data/idp/buildartifacts/apps"
		mvnCommand = "date && echo chown and running mvn in /data/idp/src && chown -R 1001 /data/idp/src && echo chown done && cd /data/idp/src && mvn -B package -DskipLibertyPackage -Dmaven.repo.local=/data/idp/cache/.m2/repository -DskipTests=true && date && echo maven build finished && ls -la target && echo starting copy of war && date && cp target/mpnew-1.0-SNAPSHOT.war /data/idp/buildartifacts/apps && date && echo chown the buildartifacts dir && chown 1001 /data/idp/buildartifacts/apps/mpnew-1.0-SNAPSHOT.war && echo list /data/idp/buildartifacts/apps && ls -al /data/idp/buildartifacts/apps && date"
	} else {
		// mvnCommand = "echo listing /data/idp/src && ls -la /data/idp/src && echo copying /data/idp/src to /tmp/app && cp -rf /data/idp/src /tmp/app && echo chown, listing and running mvn in /tmp/app: && id && chown -R 1001 /tmp/app && cd /tmp/app && echo before maven build && ls -la && mvn -B clean package -Dmaven.repo.local=/data/idp/cache/.m2/repository -DskipTests=true && echo listing after mvn && ls -la && echo copying tmp/app/target to /data/idp/output && cp -rf /tmp/app/target /data/idp/output && chown -fR 1001 /data/idp/output && echo listing /data/idp/output after mvn and chown 1001 buildoutput && ls -la /data/idp/output && echo rm -rf /data/idp/buildartifacts and copying artifacts && rm -rf /data/idp/buildartifacts && mkdir -p /data/idp/buildartifacts/ && cp -r /data/idp/output/target/liberty/wlp/usr/servers/defaultServer/* /data/idp/buildartifacts/ && cp -r /data/idp/output/target/liberty/wlp/usr/shared/resources/ /data/idp/buildartifacts/ && cp /data/idp/src/src/main/liberty/config/jvmbx.options /data/idp/buildartifacts/jvm.options && echo chown the buildartifacts dir && chown -fR 1001 /data/idp/buildartifacts"
		mvnCommand = "echo listing /data/idp/src && ls -la /data/idp/src && echo chown, listing and running mvn in /data/idp/src && chown -R 1001 /data/idp/src && cd /data/idp/src && echo list contents before maven build && ls -la && mvn -B clean package -Dmaven.repo.local=/data/idp/cache/.m2/repository -DskipTests=true && echo listing after mvn && ls -la && echo copying /data/idp/target to /data/idp/output && cp -rf /data/idp/src/target /data/idp/output && chown -fR 1001 /data/idp/output && echo listing /data/idp/output after mvn and chown 1001 buildoutput && ls -la /data/idp/output && echo rm -rf /data/idp/buildartifacts and copying artifacts && rm -rf /data/idp/buildartifacts && mkdir -p /data/idp/buildartifacts/ && cp -r /data/idp/output/target/liberty/wlp/usr/servers/defaultServer/* /data/idp/buildartifacts/ && cp -r /data/idp/output/target/liberty/wlp/usr/shared/resources/ /data/idp/buildartifacts/ && cp /data/idp/src/src/main/liberty/config/jvmbx.options /data/idp/buildartifacts/jvm.options && echo chown the buildartifacts dir && chown -fR 1001 /data/idp/buildartifacts"
	}

	fmt.Printf("Mvn Command: %s\n", mvnCommand)
	backoffLimit := int32(1)
	parallelism := int32(1)
	// user := int64(1000)
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      buildTaskJob,
			Namespace: namespace,
		},
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					// SecurityContext: &corev1.PodSecurityContext{
					// 	RunAsUser: &user,
					// },
					Volumes: []corev1.Volume{
						{
							Name: "idp-volume",
							VolumeSource: corev1.VolumeSource{
								PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
									ClaimName: idpClaimName,
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
									Name:      "idp-volume",
									MountPath: "/data/idp/",
									SubPath:   projectSubPath,
								},
							},
						},
					},
					RestartPolicy: corev1.RestartPolicyNever,
				},
			},
			BackoffLimit: &backoffLimit,
			Parallelism:  &parallelism,
		},
	}

	return job, nil
}
