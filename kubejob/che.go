package main

import (
	"fmt"
	"os"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
)

func getCheWorkspaceID() string {
	cheWorkspaceID := os.Getenv("CHE_WORKSPACE_ID")
	if cheWorkspaceID == "" {
		fmt.Println("Che Workspace ID not set and unable to run the IDP Kube Job, exiting...")
	} else {
		fmt.Printf("The Che Workspace ID: %s\n", cheWorkspaceID)
	}
	return cheWorkspaceID
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
