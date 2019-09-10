package main

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// createPFEDeploy creates a Kubernetes deployment
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
			SubPath:   "projects/projA/buildartifacts",
		},
		// {
		// 	Name:      "idp-volume",
		// 	MountPath: "/resources",
		// 	SubPath:   "projects/projA/output/target/liberty/wlp/usr/servers/defaultServer/resources",
		// },
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
	// blockOwnerDeletion := true
	// controller := true
	replicas := int32(1)
	labels2 := map[string]string{
		"app":     "javamicroprofiletemplate-selector",
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
			// OwnerReferences: []metav1.OwnerReference{
			// 	{
			// 		APIVersion:         "apps/v1",
			// 		BlockOwnerDeletion: &blockOwnerDeletion,
			// 		Controller:         &controller,
			// 		Kind:               "ReplicaSet",
			// 		Name:               codewind.OwnerReferenceName,
			// 		UID:                codewind.OwnerReferenceUID,
			// 	},
			// },
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
							// Command:      []string{"tail"},
							// Args:         []string{"-f", "/dev/null"},
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
	// blockOwnerDeletion := true
	// controller := true

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
			// OwnerReferences: []metav1.OwnerReference{
			// 	{
			// 		APIVersion:         "apps/v1",
			// 		BlockOwnerDeletion: &blockOwnerDeletion,
			// 		Controller:         &controller,
			// 		Kind:               "ReplicaSet",
			// 		Name:               codewind.OwnerReferenceName,
			// 		UID:                codewind.OwnerReferenceUID,
			// 	},
			// },
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
				"app": "javamicroprofiletemplate-selector",
			},
		},
	}
	return service
}
