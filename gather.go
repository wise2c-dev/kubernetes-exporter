package main

import (
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/pkg/apis/extensions/v1beta1"
	"k8s.io/client-go/rest"
)

// Data kubernate rest data
type Data struct {
}

// Discovery kubernate discovery client
type Discovery struct {
	client kubernetes.Interface
	role   string
}

// GathData kubernates data
type GathData struct {
	pods        *v1.PodList
	nodes       *v1.NodeList
	services    *v1.ServiceList
	endpoints   *v1.EndpointsList
	deployments *v1beta1.DeploymentList
	stacks      map[Stack]*[]v1beta1.Deployment
}

// Run fetch kubernates data
func (d *Discovery) Run() *GathData {

	fmt.Println("kubernate gather running")

	deployments, err := d.client.ExtensionsV1beta1().Deployments(api.NamespaceAll).List(v1.ListOptions{})
	if err != nil {
		fmt.Println(err)
		return nil
	}

	pods, err := d.client.Core().Pods(api.NamespaceAll).List(v1.ListOptions{})
	if err != nil {
		fmt.Println(err)
		return nil
	}

	nodes, err := d.client.Core().Nodes().List(v1.ListOptions{})
	if err != nil {
		fmt.Println(err)
		return nil
	}

	services, err := d.client.Core().Services(api.NamespaceAll).List(v1.ListOptions{})
	if err != nil {
		fmt.Println(err)
		return nil
	}

	endpoints, err := d.client.Core().Endpoints(api.NamespaceAll).List(v1.ListOptions{})
	if err != nil {
		fmt.Println(err)
		return nil
	}

	stacks := map[Stack]*[]v1beta1.Deployment{}

	for _, deployment := range deployments.Items {
		stack := Stack{
			Name:      stackName(deployment),
			Namespace: deployment.Namespace,
		}

		if deployments, ok := stacks[stack]; ok {
			*deployments = append(*deployments, deployment)
			stacks[stack] = deployments
		} else {
			stacks[stack] = &[]v1beta1.Deployment{deployment}
		}

	}

	fmt.Println("kubernate gather finished")

	return &GathData{
		pods:        pods,
		nodes:       nodes,
		services:    services,
		endpoints:   endpoints,
		deployments: deployments,
		stacks:      stacks,
	}

}

// New new discovery instance
func (e *Exporter) New() (*Discovery, error) {

	var (
		kcfg *rest.Config
	)

	kcfg = &rest.Config{
		Host: e.APIServer.String(),
		TLSClientConfig: rest.TLSClientConfig{
			CAFile:   e.TLSConfig.CAFile,
			CertFile: e.TLSConfig.CertFile,
			KeyFile:  e.TLSConfig.KeyFile,
		},
		Insecure: e.TLSConfig.InsecureSkipVerify,
	}

	token := e.BearerToken

	if e.BearerTokenFile != "" {
		bf, err1 := ioutil.ReadFile(e.BearerTokenFile)
		if err1 != nil {
			return nil, err1
		}
		token = string(bf)
	}

	kcfg.BearerToken = token

	kcfg.UserAgent = "prometheus/kubernates-exporter"

	c, err2 := kubernetes.NewForConfig(kcfg)
	if err2 != nil {
		return nil, err2
	}

	return &Discovery{
		client: c,
	}, nil

}

func (e *Exporter) gatherData(ch chan<- prometheus.Metric) (*GathData, error) {

	discovery, err := e.New()
	if err != nil {
		fmt.Println(0, err)
		return nil, err
	}

	data := discovery.Run()
	return data, nil

}

// Stack group of deployment
type Stack struct {
	Name      string
	Namespace string
}

func stackName(deployment v1beta1.Deployment) string {
	return strings.Split(deployment.Name, "-")[0]
}
