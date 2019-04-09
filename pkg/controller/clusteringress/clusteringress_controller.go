package clusteringress

import (
	"context"

	networkingv1alpha1 "github.com/knative/serving/pkg/apis/networking/v1alpha1"
	routev1 "github.com/openshift/api/route/v1"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller_clusteringress")

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new ClusterIngress Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileClusterIngress{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("clusteringress-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource ClusterIngress
	err = c.Watch(&source.Kind{Type: &networkingv1alpha1.ClusterIngress{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// namespace vs cluster scoped...
	//
	// // TODO(user): Modify this to be the types you create that are owned by the primary resource
	// // Watch for changes to secondary resource Pods and requeue the owner ClusterIngress
	// err = c.Watch(&source.Kind{Type: &routev1.Route{}}, &handler.EnqueueRequestForOwner{
	// 	IsController: true,
	// 	OwnerType:    &networkingv1alpha1.ClusterIngress{},
	// })
	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileClusterIngress{}

// ReconcileClusterIngress reconciles a ClusterIngress object
type ReconcileClusterIngress struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a ClusterIngress object and makes changes based on the state read
// and what is in the ClusterIngress.Spec
// TODO(user): Modify this Reconcile function to implement your Controller logic.  This example creates
// a Pod as an example
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileClusterIngress) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling ClusterIngress")

	// Fetch the ClusterIngress instance
	instance := &networkingv1alpha1.ClusterIngress{}
	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	for _, rule := range instance.Spec.Rules {

	}

	// Define a new Route object
	route := newRouteForCR(instance)

	// Can't set owner reference - ClusterIngress is cluster-scoped,
	// Route is namespace scoped
	//
	// // Set ClusterIngress instance as the owner and controller
	// if err := controllerutil.SetControllerReference(instance, route, r.scheme); err != nil {
	// 	return reconcile.Result{}, err
	// }

	// Check if this Route already exists
	found := &routev1.Route{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: route.Name, Namespace: route.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Creating a new Route", "Route.Namespace", route.Namespace, "Route.Name", route.Name)
		err = r.client.Create(context.TODO(), route)
		if err != nil {
			return reconcile.Result{}, err
		}

		// Route created successfully - don't requeue
		return reconcile.Result{}, nil
	} else if err != nil {
		return reconcile.Result{}, err
	}

	// Route already exists - don't requeue
	reqLogger.Info("Skip reconcile: Route already exists", "Route.Namespace", found.Namespace, "Route.Name", found.Name)
	return reconcile.Result{}, nil
}

// newPodForCR returns a busybox pod with the same name/namespace as the cr
func newRouteForCR(cr *networkingv1alpha1.ClusterIngress) *routev1.Route {
	labels := map[string]string{
		"clusteringress": cr.Name,
	}
	var weight = int32(100)
	return &routev1.Route{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.Name + "-knative-route",
			Namespace: "knative-serving",
			Labels:    labels,
		},
		Spec: routev1.RouteSpec{
			To: routev1.RouteTargetReference{
				Kind:   "Service",
				Name:   "activator-service",
				Weight: &weight,
			},
		},
	}
}
