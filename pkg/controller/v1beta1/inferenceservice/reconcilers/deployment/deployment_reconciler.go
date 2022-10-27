/*
Copyright 2021 The KServe Authors.

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

package deployment

import (
	"context"
	"github.com/google/go-cmp/cmp"
	"github.com/kserve/kserve/pkg/controller/v1beta1/inferenceservice/utils"
	"strings"

	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/kserve/kserve/pkg/apis/serving/v1beta1"
	"github.com/kserve/kserve/pkg/constants"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"knative.dev/pkg/kmp"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var log = logf.Log.WithName("DeploymentReconciler")

// DeploymentReconciler reconciles the raw kubernetes deployment resource
type DeploymentReconciler struct {
	client       client.Client
	scheme       *runtime.Scheme
	Deployment   *appsv1.Deployment
	componentExt *v1beta1.ComponentExtensionSpec
}

func NewDeploymentReconciler(client client.Client,
	scheme *runtime.Scheme,
	componentMeta metav1.ObjectMeta,
	componentExt *v1beta1.ComponentExtensionSpec,
	podSpec *corev1.PodSpec) *DeploymentReconciler {
	return &DeploymentReconciler{
		client:       client,
		scheme:       scheme,
		Deployment:   createRawDeployment(componentMeta, componentExt, podSpec),
		componentExt: componentExt,
	}
}

func createRawDeployment(componentMeta metav1.ObjectMeta,
	componentExt *v1beta1.ComponentExtensionSpec,
	podSpec *corev1.PodSpec) *appsv1.Deployment {
	podMetadata := componentMeta
	podMetadata.Labels["app"] = constants.GetRawServiceLabel(componentMeta.Name)
	setDefaultPodSpec(podSpec)
	deployment := &appsv1.Deployment{
		ObjectMeta: componentMeta,
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": constants.GetRawServiceLabel(componentMeta.Name),
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: podMetadata,
				Spec:       *podSpec,
			},
		},
	}
	setDefaultDeploymentSpec(&deployment.Spec)
	return deployment
}

// checkDeploymentExist checks if the deployment exists?
func (r *DeploymentReconciler) checkDeploymentExist(client client.Client) (constants.CheckResultType, *appsv1.Deployment, error) {
	//get deployment
	existingDeployment := &appsv1.Deployment{}
	err := client.Get(context.TODO(), types.NamespacedName{
		Namespace: r.Deployment.ObjectMeta.Namespace,
		Name:      r.Deployment.ObjectMeta.Name,
	}, existingDeployment)
	if err != nil {
		if apierr.IsNotFound(err) {
			return constants.CheckResultCreate, nil, nil
		}
		return constants.CheckResultUnknown, nil, err
	}
	// existed, check equivalence
	// for HPA scaling, we should ignore Replicas of Deployment
	// for metadata injection by other controllers, we should ignore annotations and labels and compare them separately
	ignoreFields := cmpopts.IgnoreFields(appsv1.DeploymentSpec{}, "Replicas", "Template.ObjectMeta.Annotations", "Template.ObjectMeta.Labels", "Template.Spec.DeprecatedServiceAccount", "Template.Spec.ServiceAccountName")
	if diff, err := kmp.SafeDiff(r.Deployment.Spec, existingDeployment.Spec, ignoreFields); err != nil {
		return constants.CheckResultUnknown, nil, err
	} else if diff != "" {
		log.Info("Deployment Updated", "Diff", diff)
		return constants.CheckResultUpdate, existingDeployment, nil
	}

	// only compare relevant annotations and labels between the deployment spec templates
	if diff, err := compareMetadata(r.Deployment.Spec.Template, existingDeployment.Spec.Template); err != nil {
		return constants.CheckResultUnknown, nil, err
	} else if diff != "" {
		log.Info("Deployment Updated", "Metadata Diff", diff)
		return constants.CheckResultUpdate, existingDeployment, nil
	}
	return constants.CheckResultExisted, existingDeployment, nil
}

func compareMetadata(template corev1.PodTemplateSpec, existingDeploymentSpecTemplate corev1.PodTemplateSpec) (string, error) {
	log.Info("Comparing Metadata")
	metadataComparer := cmp.Comparer(
		func(x, y corev1.PodTemplateSpec) bool {
			// filter only relevant annotations
			relevantAnnotationsInX := filterRelevantAnnotations(x.ObjectMeta.Annotations)
			relevantAnnotationsInY := filterRelevantAnnotations(y.ObjectMeta.Annotations)

			// compare only relevant annotations
			log.Info("Comparing Metadata", "X Annotations", relevantAnnotationsInX)
			log.Info("Comparing Metadata", "Y Annotations", relevantAnnotationsInY)
			if diff, _ := kmp.SafeDiff(relevantAnnotationsInX, relevantAnnotationsInY); diff != "" {
				return false
			}

			// filter only relevant labels
			relevantLabelsInX := filterRelevantLabels(x.ObjectMeta.Labels)
			relevantLabelsInY := filterRelevantLabels(y.ObjectMeta.Labels)

			// compare only relevant labels
			log.Info("Comparing Metadata", "X Labels", relevantLabelsInX)
			log.Info("Comparing Metadata", "Y Labels", relevantLabelsInY)
			if diff, _ := kmp.SafeDiff(relevantLabelsInX, relevantLabelsInY); diff != "" {
				return false
			}

			return true
		})

	// if metadata contains annotations or labels that aren't relevant then use custom comparer
	filterMetadata := cmp.FilterValues(
		func(x, y corev1.PodTemplateSpec) bool {
			// check if either pod template spec x or y has any irrelevant annotations
			if isIrrelevantAnnotationInX := irrelevantAnnotationInAnnotations(x.ObjectMeta.Annotations); isIrrelevantAnnotationInX {
				return true
			}
			if isIrrelevantAnnotationInY := irrelevantAnnotationInAnnotations(y.ObjectMeta.Annotations); isIrrelevantAnnotationInY {
				return true
			}

			// check if either pod template spec x or y has any irrelevant labels
			if isIrrelevantLabelInX := irrelevantLabelInLabels(x.ObjectMeta.Labels); isIrrelevantLabelInX {
				return true
			}
			if isIrrelevantLabelInY := irrelevantLabelInLabels(y.ObjectMeta.Labels); isIrrelevantLabelInY {
				return true
			}
			return false
		}, metadataComparer)

	diff, err := kmp.SafeDiff(template, existingDeploymentSpecTemplate, filterMetadata)

	return diff, err
}

func irrelevantAnnotationInAnnotations(annotations map[string]string) bool {
	for k := range annotations {
		if relevantAnnotation := utils.StringInSlice(k, constants.InferenceServiceAnnotations) || metadataIsOwnedByKubernetes(k); !relevantAnnotation {
			return true
		}
	}
	return false
}

func irrelevantLabelInLabels(labels map[string]string) bool {
	for k := range labels {
		if relevantLabel := utils.StringInSlice(k, constants.InferenceServiceLabels) || metadataIsOwnedByKubernetes(k); !relevantLabel {
			return true
		}
	}
	return false
}

func filterRelevantAnnotations(annotations map[string]string) (relevantAnnotations map[string]string) {
	relevantAnnotations = map[string]string{}
	for k := range annotations {
		if relevantAnnotation := utils.StringInSlice(k, constants.InferenceServiceAnnotations) || metadataIsOwnedByKubernetes(k); relevantAnnotation {
			relevantAnnotations[k] = annotations[k]
		}
	}
	return
}

func filterRelevantLabels(labels map[string]string) (relevantLabels map[string]string) {
	relevantLabels = map[string]string{}
	for k := range labels {
		if relevantLabel := utils.StringInSlice(k, constants.InferenceServiceLabels) || metadataIsOwnedByKubernetes(k); relevantLabel {
			relevantLabels[k] = labels[k]
		}
	}
	return
}

func metadataIsOwnedByKubernetes(key string) bool {
	return strings.Contains(key, "kubernetes.io")
}

func setDefaultPodSpec(podSpec *corev1.PodSpec) {
	if podSpec.DNSPolicy == "" {
		podSpec.DNSPolicy = corev1.DNSClusterFirst
	}
	if podSpec.RestartPolicy == "" {
		podSpec.RestartPolicy = corev1.RestartPolicyAlways
	}
	if podSpec.TerminationGracePeriodSeconds == nil {
		TerminationGracePeriodSeconds := int64(corev1.DefaultTerminationGracePeriodSeconds)
		podSpec.TerminationGracePeriodSeconds = &TerminationGracePeriodSeconds
	}
	if podSpec.SecurityContext == nil {
		podSpec.SecurityContext = &corev1.PodSecurityContext{}
	}
	if podSpec.SchedulerName == "" {
		podSpec.SchedulerName = corev1.DefaultSchedulerName
	}
	for i := range podSpec.Containers {
		container := &podSpec.Containers[i]
		if container.TerminationMessagePath == "" {
			container.TerminationMessagePath = "/dev/termination-log"
		}
		if container.TerminationMessagePolicy == "" {
			container.TerminationMessagePolicy = corev1.TerminationMessageReadFile
		}
		if container.ImagePullPolicy == "" {
			container.ImagePullPolicy = corev1.PullIfNotPresent
		}
		// generate default readiness probe for model server container
		if container.Name == constants.InferenceServiceContainerName {
			if container.ReadinessProbe == nil {
				if len(container.Ports) == 0 {
					container.ReadinessProbe = &corev1.Probe{
						ProbeHandler: corev1.ProbeHandler{
							TCPSocket: &corev1.TCPSocketAction{
								Port: intstr.IntOrString{
									IntVal: 8080,
								},
							},
						},
						TimeoutSeconds:   1,
						PeriodSeconds:    10,
						SuccessThreshold: 1,
						FailureThreshold: 3,
					}
				} else {
					container.ReadinessProbe = &corev1.Probe{
						ProbeHandler: corev1.ProbeHandler{
							TCPSocket: &corev1.TCPSocketAction{
								Port: intstr.IntOrString{
									IntVal: container.Ports[0].ContainerPort,
								},
							},
						},
						TimeoutSeconds:   1,
						PeriodSeconds:    10,
						SuccessThreshold: 1,
						FailureThreshold: 3,
					}
				}
			}
		}
	}
}

func setDefaultDeploymentSpec(spec *appsv1.DeploymentSpec) {
	if spec.Strategy.Type == "" {
		spec.Strategy.Type = appsv1.RollingUpdateDeploymentStrategyType
	}
	if spec.Strategy.RollingUpdate == nil {
		spec.Strategy.RollingUpdate = &appsv1.RollingUpdateDeployment{
			MaxUnavailable: &intstr.IntOrString{Type: intstr.String, StrVal: "25%"},
			MaxSurge:       &intstr.IntOrString{Type: intstr.String, StrVal: "25%"},
		}
	}
	if spec.RevisionHistoryLimit == nil {
		revisionHistoryLimit := int32(10)
		spec.RevisionHistoryLimit = &revisionHistoryLimit
	}
	if spec.ProgressDeadlineSeconds == nil {
		progressDeadlineSeconds := int32(600)
		spec.ProgressDeadlineSeconds = &progressDeadlineSeconds
	}
}

// Reconcile ...
func (r *DeploymentReconciler) Reconcile() (*appsv1.Deployment, error) {
	//reconcile Deployment
	checkResult, deployment, err := r.checkDeploymentExist(r.client)
	if err != nil {
		return nil, err
	}
	log.Info("deployment reconcile", "checkResult", checkResult, "err", err)
	if checkResult == constants.CheckResultCreate {
		err = r.client.Create(context.TODO(), r.Deployment)
		if err != nil {
			return nil, err
		} else {
			return r.Deployment, nil
		}
	} else if checkResult == constants.CheckResultUpdate {
		err = r.client.Update(context.TODO(), r.Deployment)
		if err != nil {
			return nil, err
		} else {
			return r.Deployment, nil
		}
	} else {
		return deployment, nil
	}
}
