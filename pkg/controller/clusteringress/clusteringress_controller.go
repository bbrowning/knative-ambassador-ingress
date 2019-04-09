package clusteringress

import (
	"context"
	"fmt"
	"os"
	"reflect"

	"github.com/bbrowning/knative-ambassador-ingress/pkg/controller/clusteringress/resources"
	"github.com/knative/pkg/logging"
	"github.com/knative/serving/pkg/apis/networking"
	networkingv1alpha1 "github.com/knative/serving/pkg/apis/networking/v1alpha1"
	"github.com/knative/serving/pkg/utils"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	ambassadorNamespaceEnvVar = "AMBASSADOR_NAMESPACE"
	ambassadorIngressClass    = "ambassador.ingress.networking.knative.dev"
)

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
	ctx := context.TODO()
	logger := logging.FromContext(ctx)
	ambassadorNamespace, found := os.LookupEnv(ambassadorNamespaceEnvVar)
	if !found {
		logger.Fatalf("%s must be set", ambassadorNamespaceEnvVar)
	}
	if len(ambassadorNamespace) == 0 {
		logger.Fatalf("%s must not be empty", ambassadorNamespaceEnvVar)
	}
	return &ReconcileClusterIngress{
		client:              mgr.GetClient(),
		scheme:              mgr.GetScheme(),
		ambassadorNamespace: ambassadorNamespace,
	}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("clusteringress-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource ClusterIngress
	// TODO: Add a filter based on IngressClassAnnotationKey from kn/serving
	err = c.Watch(&source.Kind{Type: &networkingv1alpha1.ClusterIngress{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// TODO: watch K8s Services but w/o checking ownerref...
	// // Watch for changes to secondary resource Services and requeue
	// // the owner ClusterIngress
	// err = c.Watch(&source.Kind{Type: &corev1.Service{}}, &handler.EnqueueRequestForOwner{
	// 	IsController: true,
	// 	OwnerType:    &networkingv1alpha1.ClusterIngress{},
	// })
	// if err != nil {
	// 	return err
	// }

	return nil
}

var _ reconcile.Reconciler = &ReconcileClusterIngress{}

// ReconcileClusterIngress reconciles a ClusterIngress object
type ReconcileClusterIngress struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client              client.Client
	scheme              *runtime.Scheme
	ambassadorNamespace string
}

// Reconcile reads that state of the cluster for a ClusterIngress object and makes changes based on the state read
// and what is in the ClusterIngress.Spec
// TODO(user): Modify this Reconcile function to implement your Controller logic.  This example creates
// a Pod as an example
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileClusterIngress) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	ctx := context.TODO()
	logger := logging.FromContext(ctx)

	// Fetch the ClusterIngress instance
	original := &networkingv1alpha1.ClusterIngress{}
	err := r.client.Get(context.TODO(), request.NamespacedName, original)
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

	// Only reconcile Ambassador ClusterIngress objects
	ingressClass := original.ObjectMeta.Annotations[networking.IngressClassAnnotationKey]
	if ingressClass != ambassadorIngressClass {
		return reconcile.Result{}, nil
	}

	// Don't modify the informer's copy
	ci := original.DeepCopy()

	err = r.reconcile(ctx, ci)
	if equality.Semantic.DeepEqual(original.Status, ci.Status) {
		// If we didn't change anything then don't call updateStatus.
		// This is important because the copy we loaded from the informer's
		// cache may be stale and we don't want to overwrite a prior update
		// to status with this stale state.
	} else if _, err := r.updateStatus(ctx, ci); err != nil {
		logger.Warnw("Failed to update clusterIngress status", err)
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}

// Update the Status of the ClusterIngress.  Caller is responsible for checking
// for semantic differences before calling.
func (r *ReconcileClusterIngress) updateStatus(ctx context.Context, desired *networkingv1alpha1.ClusterIngress) (*networkingv1alpha1.ClusterIngress, error) {
	ci := &networkingv1alpha1.ClusterIngress{}
	err := r.client.Get(ctx, types.NamespacedName{Name: desired.Name, Namespace: desired.Namespace}, ci)
	if err != nil {
		return nil, err
	}

	// If there's nothing to update, just return.
	if reflect.DeepEqual(ci.Status, desired.Status) {
		return ci, nil
	}
	// Don't modify the informers copy
	existing := ci.DeepCopy()
	existing.Status = desired.Status
	err = r.client.Status().Update(ctx, existing)
	return existing, err
}

func (r *ReconcileClusterIngress) reconcile(ctx context.Context, ci *networkingv1alpha1.ClusterIngress) error {
	logger := logging.FromContext(ctx)
	if ci.GetDeletionTimestamp() != nil {
		return r.reconcileDeletion(ctx, ci)
	}

	// We may be reading a version of the object that was stored at an older version
	// and may not have had all of the assumed defaults specified.  This won't result
	// in this getting written back to the API Server, but lets downstream logic make
	// assumptions about defaulting.
	ci.SetDefaults(ctx)

	ci.Status.InitializeConditions()

	svc, err := resources.MakeService(ctx, ci, r.ambassadorNamespace)
	if err != nil {
		return err
	}

	logger.Infof("Reconciling clusterIngress :%v", ci)
	logger.Info("Creating/Updating Ambassador config on K8s Service")
	if err := r.reconcileService(ctx, ci, svc); err != nil {
		return err
	}

	ci.Status.MarkNetworkConfigured()
	ci.Status.MarkLoadBalancerReady(getLBStatus())
	ci.Status.ObservedGeneration = ci.Generation

	logger.Info("ClusterIngress successfully synced")
	return nil
}

func (r *ReconcileClusterIngress) reconcileService(ctx context.Context, ci *networkingv1alpha1.ClusterIngress,
	desired *corev1.Service) error {
	logger := logging.FromContext(ctx)

	// TODO: Owner refs
	// // Set ClusterIngress instance as the owner and controller
	// if err := controllerutil.SetControllerReference(instance, service, r.scheme); err != nil {
	// 	return reconcile.Result{}, err
	// }

	// Check if this Service already exists
	svc := &corev1.Service{}
	err := r.client.Get(ctx, types.NamespacedName{Name: desired.Name, Namespace: desired.Namespace}, svc)
	if err != nil && errors.IsNotFound(err) {
		err = r.client.Create(ctx, desired)
		if err != nil {
			logger.Errorw("Failed to create Ambassador config on K8s Service", err)
			return err
		}
		logger.Infof("Created Ambassador config on K8s Service %q in namespace %q", desired.Name, desired.Namespace)
	} else if err != nil {
		return err
	} else if !equality.Semantic.DeepEqual(svc.Spec, desired.Spec) || !equality.Semantic.DeepEqual(svc.ObjectMeta.Annotations, desired.ObjectMeta.Annotations) {
		// Don't modify the informers copy
		existing := svc.DeepCopy()
		existing.Spec = desired.Spec
		existing.ObjectMeta.Annotations = desired.ObjectMeta.Annotations
		err = r.client.Update(ctx, existing)
		if err != nil {
			logger.Errorw("Failed to update Ambassador config on K8s Service", err)
			return err
		}
	}

	return nil
}

func (r *ReconcileClusterIngress) reconcileDeletion(ctx context.Context, ci *networkingv1alpha1.ClusterIngress) error {
	// TODO: something with a finalizer
	return nil
}

func getLBStatus() []networkingv1alpha1.LoadBalancerIngressStatus {
	// TODO: something better...
	return []networkingv1alpha1.LoadBalancerIngressStatus{
		{DomainInternal: fmt.Sprintf("ambassador.default.svc.%s", utils.GetClusterDomainName())},
	}
}
