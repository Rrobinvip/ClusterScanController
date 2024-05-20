/*
Copyright 2024.

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

package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	kbatch "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	scanv1 "github.com/rrobinvip/ClusterScanController/api/v1"
)

// ClusterScanReconciler reconciles a ClusterScan object
type ClusterScanReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func (r *ClusterScanReconciler) createOrUpdateJob(ctx context.Context, clusterScan *scanv1.ClusterScan) error {
	job := &kbatch.Job{}
	jobName := fmt.Sprintf("%s-scan", clusterScan.Name)
	jobNamespace := clusterScan.Namespace

	err := r.Get(ctx, types.NamespacedName{Name: jobName, Namespace: jobNamespace}, job)
	if err != nil && errors.IsNotFound(err) {
		// Define a new Job because it does not exist
		job = constructJobForClusterScan(clusterScan)
		if err := r.Create(ctx, job); err != nil {
			return err
		}
	} else if err == nil {
		// Update existing Job if necessary
		updatedJob := constructJobForClusterScan(clusterScan)
		job.Spec = updatedJob.Spec
		if err := r.Update(ctx, job); err != nil {
			return err
		}
	}

	return nil
}

func constructJobForClusterScan(clusterScan *scanv1.ClusterScan) *kbatch.Job {
	return &kbatch.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-scan", clusterScan.Name),
			Namespace: clusterScan.Namespace,
		},
		Spec: kbatch.JobSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:    "terrascan",
							Image:   "foo/terrascan", //TODO: real image
							Command: []string{"terrascan", "scan", "-i", "kubernetes", "-d", "/configs", "--output", "json"},
						},
					},
					RestartPolicy: corev1.RestartPolicyNever,
				},
			},
		},
	}
}

func (r *ClusterScanReconciler) createOrUpdateCronJob(ctx context.Context, clusterScan *scanv1.ClusterScan) error {
	cronJob := &kbatch.CronJob{}
	cronJobName := fmt.Sprintf("%s-scan", clusterScan.Name)
	cronJobNamespace := clusterScan.Namespace

	err := r.Get(ctx, types.NamespacedName{Name: cronJobName, Namespace: cronJobNamespace}, cronJob)
	if err != nil && errors.IsNotFound(err) {
		// Define a new CronJob because it does not exist
		cronJob = constructCronJobForClusterScan(clusterScan)
		if err := r.Create(ctx, cronJob); err != nil {
			return err
		}
	} else if err == nil {
		// Update existing CronJob if necessary
		updatedCronJob := constructCronJobForClusterScan(clusterScan)
		cronJob.Spec = updatedCronJob.Spec
		if err := r.Update(ctx, cronJob); err != nil {
			return err
		}
	}

	return nil
}

func constructCronJobForClusterScan(clusterScan *scanv1.ClusterScan) *kbatch.CronJob {
	return &kbatch.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-scan", clusterScan.Name),
			Namespace: clusterScan.Namespace,
		},
		Spec: kbatch.CronJobSpec{
			Schedule: clusterScan.Spec.Schedule,
			JobTemplate: kbatch.JobTemplateSpec{
				Spec: kbatch.JobSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name:    "terrascan",
									Image:   "foo/terrascan", //TODO: real image
									Command: []string{"terrascan", "scan", "-i", "kubernetes", "-d", "/configs", "--output", "json"},
								},
							},
							RestartPolicy: corev1.RestartPolicyOnFailure,
						},
					},
				},
			},
		},
	}
}

func (r *ClusterScanReconciler) updateClusterScanStatus(ctx context.Context, clusterScan *scanv1.ClusterScan) error {

	clusterScan.Status.LastScanTime = &metav1.Time{Time: time.Now()}
	// TODO: dynamically assign ScanStatus and result
	clusterScan.Status.ScanStatus = "Completed"
	clusterScan.Status.Result = (json.RawMessage{})

	return r.Status().Update(ctx, clusterScan)
}

//+kubebuilder:rbac:groups=scan.clusterscan.io.clusterscan.io,resources=clusterscans,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=scan.clusterscan.io.clusterscan.io,resources=clusterscans/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=scan.clusterscan.io.clusterscan.io,resources=clusterscans/finalizers,verbs=update
//+kubebuilder:rbac:groups=scan,resources=jobs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=scan,resources=jobs/status,verbs=get

func (r *ClusterScanReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Fetch the ClusterScan instanec
	var clusterScan scanv1.ClusterScan
	if err := r.Get(ctx, req.NamespacedName, &clusterScan); err != nil {
		logger.Error(err, "Unable to fetch clusterScan instance")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	var err error
	switch clusterScan.Spec.ScanFrequencyPolicy {
	case scanv1.OneOff:
		err = r.createOrUpdateJob(ctx, &clusterScan)
	case scanv1.Recurring:
		err = r.createOrUpdateCronJob(ctx, &clusterScan)
	default:
		logger.Error(err, "Invalid ScanFrequencyPolicy", "Policy", clusterScan.Spec.ScanFrequencyPolicy)
		return ctrl.Result{}, err
	}

	if err != nil {
		return ctrl.Result{}, err
	}

	if err := r.updateClusterScanStatus(ctx, &clusterScan); err != nil {
		logger.Error(err, "Failed to update ClusterScan status")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ClusterScanReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&scanv1.ClusterScan{}).
		Complete(r)
}
