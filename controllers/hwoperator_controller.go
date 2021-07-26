/*
Copyright 2021.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1beta1 "k8s.io/api/networking/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	hwoperatorcomv1 "github.com/jbebe/hwoperator/api/v1"
)

// HwOperatorReconciler reconciles a HwOperator object
type HwOperatorReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=hwoperator.com,resources=hwoperators,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=hwoperator.com,resources=hwoperators/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=hwoperator.com,resources=hwoperators/finalizers,verbs=update
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the HwOperator object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.9.2/pkg/reconcile
func (r *HwOperatorReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)
	reqLogger := log.Log.WithValues("Request.Namespace", req.Namespace, "Request.Name", req.Name)
	reqLogger.Info("Reconciling HwOperator")

	// Fetch HwOperator instance
	operatorInstance := &hwoperatorcomv1.HwOperator{}
	err := r.Client.Get(context.TODO(), req.NamespacedName, operatorInstance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return ctrl.Result{}, err
	}

	// Check if this Deployment already exists
	found := &appsv1.Deployment{}
	err = r.Client.Get(context.TODO(), types.NamespacedName{Name: operatorInstance.Name, Namespace: operatorInstance.Namespace}, found)
	var result *ctrl.Result

	// Check deployment
	deploymentInstance := r.CreateFrontendDeployment(operatorInstance)
	result, err = r.EnsureDeployment(req, operatorInstance, deploymentInstance)
	if result != nil {
		return *result, err
	}

	// Check service
	serviceInstance := r.CreateFrontendService(operatorInstance)
	result, err = r.EnsureService(req, operatorInstance, serviceInstance)
	if result != nil {
		return *result, err
	}

	// Check ingress
	ingressInstance := r.CreateFrontendIngress(operatorInstance, serviceInstance)
	result, err = r.EnsureIngress(req, operatorInstance, ingressInstance)
	if result != nil {
		return *result, err
	}

	// Deployment and Service already exists - don't requeue
	reqLogger.Info("Skip reconcile: Deployment and service already exists",
		"Deployment.Namespace", found.Namespace, "Deployment.Name", found.Name)

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *HwOperatorReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&hwoperatorcomv1.HwOperator{}).
		Complete(r)
}

func (r *HwOperatorReconciler) CreateFrontendService(v *hwoperatorcomv1.HwOperator) *corev1.Service {
	// Build a Service and deploys
	s := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "frontend-service",
			Namespace: v.Namespace,
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"tier": "frontend",
			},
			Ports: []corev1.ServicePort{{
				Protocol:   corev1.ProtocolTCP,
				Port:       80,
				TargetPort: intstr.FromInt(80),
				NodePort:   30685,
			}},
			Type: corev1.ServiceTypeNodePort,
		},
	}

	controllerutil.SetControllerReference(v, s, r.Scheme)
	return s
}

func (r *HwOperatorReconciler) CreateFrontendIngress(
	v *hwoperatorcomv1.HwOperator,
	service *corev1.Service,
) *networkingv1beta1.Ingress {
	// Build a Service and deploys
	pathTypePrefix := networkingv1beta1.PathTypePrefix
	ingress := &networkingv1beta1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "frontend-ingress",
			Namespace: v.Namespace,
			Annotations: map[string]string{
				"kubernetes.io/ingress.class": "nginx",
				"cert-manager.io/issuer":      "letsencrypt",
			},
		},
		Spec: networkingv1beta1.IngressSpec{
			TLS: []networkingv1beta1.IngressTLS{{
				Hosts:      []string{v.Spec.Host},
				SecretName: "frontend-secret",
			}},
			Rules: []networkingv1beta1.IngressRule{{
				Host: v.Spec.Host,
				IngressRuleValue: networkingv1beta1.IngressRuleValue{
					HTTP: &networkingv1beta1.HTTPIngressRuleValue{
						Paths: []networkingv1beta1.HTTPIngressPath{{
							Path:     "/",
							PathType: &pathTypePrefix,
							Backend: networkingv1beta1.IngressBackend{
								ServiceName: service.Name,
								ServicePort: intstr.FromInt(80),
							},
						}},
					},
				},
			}},
		},
	}

	controllerutil.SetControllerReference(v, ingress, r.Scheme)
	return ingress
}

func (r *HwOperatorReconciler) CreateFrontendDeployment(v *hwoperatorcomv1.HwOperator) *appsv1.Deployment {
	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "nginx-frontend",
			Namespace: v.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &v.Spec.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"tier": "frontend",
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"tier": "frontend",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Image:           v.Spec.Image,
						ImagePullPolicy: corev1.PullAlways,
						Name:            "nginx-frontend",
						Ports: []corev1.ContainerPort{{
							ContainerPort: 80,
							Name:          "frontend",
						}},
						Env: []corev1.EnvVar{{
							Name:  "NGINX_HOST",
							Value: v.Spec.Host,
						}},
					}},
				},
			},
		},
	}

	controllerutil.SetControllerReference(v, dep, r.Scheme)
	return dep
}

func (r *HwOperatorReconciler) EnsureService(request ctrl.Request,
	instance *hwoperatorcomv1.HwOperator,
	s *corev1.Service,
) (*ctrl.Result, error) {

	// See if service already exists and create if it doesn't
	found := &appsv1.Deployment{}
	err := r.Client.Get(context.TODO(), types.NamespacedName{
		Name:      s.Name,
		Namespace: instance.Namespace,
	}, found)
	if err != nil && errors.IsNotFound(err) {

		// Create the service
		log.Log.Info("Creating a new Service", "Service.Namespace", s.Namespace, "Service.Name", s.Name)
		err = r.Client.Create(context.TODO(), s)

		if err != nil {
			// Service creation failed
			log.Log.Error(err, "Failed to create new Service", "Service.Namespace", s.Namespace, "Service.Name", s.Name)
			return &ctrl.Result{}, err
		} else {
			// Service creation was successful
			return nil, nil
		}
	} else if err != nil {
		// Error that isn't due to the service not existing
		log.Log.Error(err, "Failed to get Service")
		return &ctrl.Result{}, err
	}

	return nil, nil
}

func (r *HwOperatorReconciler) EnsureDeployment(
	request ctrl.Request, instance *hwoperatorcomv1.HwOperator, dep *appsv1.Deployment,
) (*ctrl.Result, error) {

	// See if deployment already exists and create if it doesn't
	found := &appsv1.Deployment{}
	err := r.Client.Get(context.TODO(), types.NamespacedName{
		Name:      dep.Name,
		Namespace: instance.Namespace,
	}, found)
	if err != nil && errors.IsNotFound(err) {
		// Create the deployment
		log.Log.Info("Creating a new Deployment", "Deployment.Namespace",
			dep.Namespace, "Deployment.Name", dep.Name)
		err = r.Client.Create(context.TODO(), dep)

		if err != nil {
			// Deployment failed
			log.Log.Error(err, "Failed to create new Deployment",
				"Deployment.Namespace", dep.Namespace, "Deployment.Name", dep.Name)
			return &ctrl.Result{}, err
		} else {
			// Deployment was successful
			return nil, nil
		}
	} else if err != nil {
		// Error that isn't due to the deployment not existing
		log.Log.Error(err, "Failed to get Deployment")
		return &ctrl.Result{}, err
	}

	return nil, nil
}

func (r *HwOperatorReconciler) EnsureIngress(
	request ctrl.Request,
	instance *hwoperatorcomv1.HwOperator,
	ingr *networkingv1beta1.Ingress,
) (*ctrl.Result, error) {

	// See if ingress already exists and create if it doesn't
	found := &networkingv1beta1.Ingress{}
	err := r.Client.Get(context.TODO(), types.NamespacedName{
		Name:      ingr.Name,
		Namespace: instance.Namespace,
	}, found)
	if err != nil && errors.IsNotFound(err) {
		// Create the ingress
		log.Log.Info("Creating a new Ingress", "Ingress.Namespace",
			ingr.Namespace, "Ingress.Name", ingr.Name)
		err = r.Client.Create(context.TODO(), ingr)

		if err != nil {
			// Ingress failed
			log.Log.Error(err, "Failed to create new Ingress",
				"Ingress.Namespace", ingr.Namespace, "Ingress.Name", ingr.Name)
			return &ctrl.Result{}, err
		} else {
			// Ingress was successful
			return nil, nil
		}
	} else if err != nil {
		// Error that isn't due to the ingress not existing
		log.Log.Error(err, "Failed to get Ingress")
		return &ctrl.Result{}, err
	}

	return nil, nil
}
