package main

import (
	"context"
	"fmt"

	"log"
	"time"

	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"k8s.io/client-go/rest"
)

func main () {
	for {
		RemoveDeployments()
		time.Sleep(60000000000)
	}
}

func RemoveDeployments() {


	config, err := rest.InClusterConfig()
	if err != nil {
		log.Fatalf("Error creating in-cluster config: %v", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatalf("Error creating clientset: %v", err)
	}
	namespace := "default"

	deployments,err := clientset.AppsV1().Deployments(namespace).List(context.TODO(),metav1.ListOptions{})
	if err != nil {
		fmt.Println(err)
	}

	for _,deployment := range deployments.Items {
		isRunning,exists := deployment.ObjectMeta.Annotations["isRunning"]
		if !exists {
			fmt.Println("Annotation not found")
			continue
		}
		if isRunning == "False"  {
			lastRun,timeExists := deployment.ObjectMeta.Annotations["LastOpened"]

			lastOpenedTime, err := time.Parse(time.RFC3339, lastRun)
			if err != nil {
				fmt.Println(err)
			}
			elapsed := time.Since(lastOpenedTime)
			if elapsed > 10*time.Minute || !timeExists{
				DeleteDeployment(clientset, namespace, deployment.Name)
			}
		}
	}
}

func DeleteDeployment(client *kubernetes.Clientset, namespace,name string) {
	err := client.AppsV1().Deployments(namespace).Delete(context.TODO(),name,metav1.DeleteOptions{})
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println("Removed Deployment")

	err = client.CoreV1().Services(namespace).Delete(context.TODO(),name,metav1.DeleteOptions{})
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println("Removed Service")

	ingress,err := client.NetworkingV1().Ingresses(namespace).Get(context.TODO(), "user-ingress", metav1.GetOptions{})
	if err != nil {
		fmt.Println(err)
	}
	paths := ingress.Spec.Rules[0].HTTP.Paths

	updatedPaths := []networkingv1.HTTPIngressPath{}
	for _,path := range paths {
		if path.Backend.Service.Name == name {
			continue
		}
		updatedPaths = append(updatedPaths, path)
	}

	ingress.Spec.Rules[0].HTTP.Paths = updatedPaths

	_, err = client.NetworkingV1().Ingresses(namespace).Update(context.TODO(), ingress, metav1.UpdateOptions{})
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println("Removed from Ingress")
}