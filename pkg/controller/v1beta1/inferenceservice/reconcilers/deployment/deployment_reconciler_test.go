package deployment

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

func TestCompareMetadata(t *testing.T) {
	name := "my-model"
	namespace := "test"
	serviceAccountName := "my-service-account"
	containerName := "my-container"
	imageName := "my-image"
	commandName := "my command"

	// annotations
	targetUtilizationPercentageAnnotation := "serving.kserve.io/targetUtilizationPercentage"
	deploymentRevisionAnnotation := "deployment.kubernetes.io/revision"
	irrelevantAnnotation := "annotation.custom.io/my-annotation"
	storageInitializerSourceUriInternalAnnotationKey := "internal.serving.kserve.io/storage-initializer-sourceuri"

	annotations := map[string]string{
		deploymentRevisionAnnotation:                     "2",
		storageInitializerSourceUriInternalAnnotationKey: "s3://my-s3-1234567890-us-west-2/files/my-data/my-file",
		"serving.kserve.io/autoscalerClass":              "hpa",
		"serving.kserve.io/deploymentMode":               "RawDeployment",
		"serving.kserve.io/metrics":                      "cpu",
		targetUtilizationPercentageAnnotation:            "80",
		irrelevantAnnotation:                             "hello",
	}

	// annotations with differences in values
	annotationsWithDifferentKServeValue := copyMap(annotations)
	annotationsWithDifferentKServeValue[targetUtilizationPercentageAnnotation] = "90"

	annotationsWithDifferentK8sValue := copyMap(annotations)
	annotationsWithDifferentK8sValue[deploymentRevisionAnnotation] = "3"

	annotationsWithDifferentIrrelevantValue := copyMap(annotations)
	annotationsWithDifferentIrrelevantValue[irrelevantAnnotation] = "goodbye"

	annotationsWithDifferentRelevantValues := map[string]string{
		deploymentRevisionAnnotation:                     "3",
		storageInitializerSourceUriInternalAnnotationKey: "s3://my-s3-1234567890-us-west-2/files/my-data/my-file-2",
		"serving.kserve.io/autoscalerClass":              "hpa",
		"serving.kserve.io/deploymentMode":               "Serverless",
		"serving.kserve.io/metrics":                      "memory",
		targetUtilizationPercentageAnnotation:            "90",
		irrelevantAnnotation:                             "hello",
	}

	annotationsWithDifferentRelevantAndIrrelevantValues := copyMap(annotationsWithDifferentRelevantValues)
	annotationsWithDifferentRelevantAndIrrelevantValues[irrelevantAnnotation] = "goodbye"

	// annotations with missing keys
	annotationsWithMissingKServeKey := copyMap(annotations)
	delete(annotationsWithMissingKServeKey, storageInitializerSourceUriInternalAnnotationKey)

	annotationsWithMissingK8sKey := copyMap(annotations)
	delete(annotationsWithMissingK8sKey, deploymentRevisionAnnotation)

	annotationsWithMissingIrrelevantKey := copyMap(annotations)
	delete(annotationsWithMissingIrrelevantKey, irrelevantAnnotation)

	// annotations with additional keys
	annotationsWithAdditionalKServeAnnotation := copyMap(annotations)
	annotationsWithAdditionalKServeAnnotation["serving.kserve.io/enable-tag-routing"] = "true"

	annotationsWithAdditionalIrrelevantAnnotation := copyMap(annotations)
	annotationsWithAdditionalIrrelevantAnnotation["annotation.custom.io/my-other-annotation"] = "hello-again"

	// labels
	kServiceComponentLabel := "component"
	k8sInstanceLabel := "app.kubernetes.io/instance"
	irrelevantLabel := "label.custom.io/my-label"

	labels := map[string]string{
		k8sInstanceLabel:               "my-instance",
		"app.kubernetes.io/managed-by": "Helm",
		"app.kubernetes.io/name":       "my-app",
		kServiceComponentLabel:         "predictor",
		"model":                        "sklearn",
		irrelevantLabel:                "hello",
	}

	// labels with differences in values
	k8sKServeAndIrrelevantLabelsWithDifferentKServeValue := copyMap(labels)
	k8sKServeAndIrrelevantLabelsWithDifferentKServeValue[kServiceComponentLabel] = "transformer"

	k8sKServeAndIrrelevantLabelsWithDifferentK8sValue := copyMap(labels)
	k8sKServeAndIrrelevantLabelsWithDifferentK8sValue[k8sInstanceLabel] = "my-different-instance"

	k8sKServeAndIrrelevantLabelsWithDifferentIrrelevantValue := copyMap(labels)
	k8sKServeAndIrrelevantLabelsWithDifferentIrrelevantValue[irrelevantLabel] = "goodbye"

	k8sKServeAndIrrelevantLabelsWithDifferentRelevantValues := map[string]string{
		k8sInstanceLabel:               "my-other-instance",
		"app.kubernetes.io/managed-by": "Kustomize",
		"app.kubernetes.io/name":       "my-other-app",
		kServiceComponentLabel:         "transformer",
		"model":                        "xgboost",
		irrelevantLabel:                "hello",
	}

	k8sKServeAndIrrelevantLabelsWithDifferentRelevantAndIrrelevantValues := copyMap(k8sKServeAndIrrelevantLabelsWithDifferentRelevantValues)
	k8sKServeAndIrrelevantLabelsWithDifferentRelevantAndIrrelevantValues[irrelevantLabel] = "goodbye"

	// labels with missing keys
	labelsWithMissingKServeKey := copyMap(labels)
	delete(labelsWithMissingKServeKey, kServiceComponentLabel)

	labelsWithMissingK8sKey := copyMap(labels)
	delete(labelsWithMissingK8sKey, k8sInstanceLabel)

	labelsWithMissingIrrelevantKey := copyMap(labels)
	delete(labelsWithMissingIrrelevantKey, irrelevantLabel)

	// labels with additional keys
	labelsWithAdditionalKServeKey := copyMap(labels)
	labelsWithAdditionalKServeKey["endpoint"] = "default"

	labelsWithAdditionalK8sKey := copyMap(labels)
	labelsWithAdditionalK8sKey["app.kubernetes.io/version"] = "5"

	labelsWithAdditionalIrrelevantKey := copyMap(labels)
	labelsWithAdditionalIrrelevantKey["label.custom.io/my-other-label"] = "hello-again"

	cases := []struct {
		name         string
		xAnnotations map[string]string
		yAnnotations map[string]string
		xLabels      map[string]string
		yLabels      map[string]string
		expectedDiff bool
	}{
		// identical annotations
		{
			name:         "identical deployment spec templates should return no diff",
			xAnnotations: annotations,
			yAnnotations: annotations,
			xLabels:      labels,
			yLabels:      labels,
			expectedDiff: false,
		},
		// differences in annotations
		{
			name:         "deployment spec template with a different KServe annotation should return diff",
			xAnnotations: annotations,
			yAnnotations: annotationsWithDifferentKServeValue,
			xLabels:      labels,
			yLabels:      labels,
			expectedDiff: true,
		},
		{
			name:         "deployment spec template with a different kubernetes annotation should return diff",
			xAnnotations: annotations,
			yAnnotations: annotationsWithDifferentK8sValue,
			xLabels:      labels,
			yLabels:      labels,
			expectedDiff: true,
		},
		{
			name:         "deployment spec template with a different irrelevant annotation should return no diff",
			xAnnotations: annotations,
			yAnnotations: annotationsWithDifferentIrrelevantValue,
			xLabels:      labels,
			yLabels:      labels,
			expectedDiff: false,
		},
		{
			name:         "deployment spec template with different relevant annotations should return diff",
			xAnnotations: annotations,
			yAnnotations: annotationsWithDifferentRelevantValues,
			xLabels:      labels,
			yLabels:      labels,
			expectedDiff: true,
		},
		{
			name:         "deployment spec template with a different relevant and irrelevant annotations should return diff",
			xAnnotations: annotations,
			yAnnotations: annotationsWithDifferentRelevantAndIrrelevantValues,
			xLabels:      labels,
			yLabels:      labels,
			expectedDiff: true,
		},
		// missing annotations
		{
			name:         "deployment spec template with a missing KServe annotation should return diff",
			xAnnotations: annotations,
			yAnnotations: annotationsWithMissingKServeKey,
			xLabels:      labels,
			yLabels:      labels,
			expectedDiff: true,
		},
		{
			name:         "deployment spec template with a missing K8s annotation should return diff",
			xAnnotations: annotations,
			yAnnotations: annotationsWithMissingK8sKey,
			xLabels:      labels,
			yLabels:      labels,
			expectedDiff: true,
		},
		{
			name:         "deployment spec template with a missing irrelevant annotation should return no diff",
			xAnnotations: annotations,
			yAnnotations: annotationsWithMissingIrrelevantKey,
			xLabels:      labels,
			yLabels:      labels,
			expectedDiff: false,
		},
		// additional annotations
		{
			name:         "deployment spec template with an additional KServe annotation should return diff",
			xAnnotations: annotations,
			yAnnotations: annotationsWithAdditionalKServeAnnotation,
			xLabels:      labels,
			yLabels:      labels,
			expectedDiff: true,
		},
		{
			name:         "deployment spec template with an additional K8s annotation should return diff",
			xAnnotations: annotations,
			yAnnotations: annotationsWithAdditionalKServeAnnotation,
			xLabels:      labels,
			yLabels:      labels,
			expectedDiff: true,
		},
		{
			name:         "deployment spec template with an additional irrelevant annotation should not return diff",
			xAnnotations: annotations,
			yAnnotations: annotationsWithAdditionalIrrelevantAnnotation,
			xLabels:      labels,
			yLabels:      labels,
			expectedDiff: false,
		},
		// differences in labels
		{
			name:         "deployment spec template with a different KServe label should return diff",
			xAnnotations: annotations,
			yAnnotations: annotations,
			xLabels:      labels,
			yLabels:      k8sKServeAndIrrelevantLabelsWithDifferentKServeValue,
			expectedDiff: true,
		},
		{
			name:         "deployment spec template with a different kubernetes label should return diff",
			xAnnotations: annotations,
			yAnnotations: annotations,
			xLabels:      labels,
			yLabels:      k8sKServeAndIrrelevantLabelsWithDifferentK8sValue,
			expectedDiff: true,
		},
		{
			name:         "deployment spec template with a different irrelevant label should return no diff",
			xAnnotations: annotations,
			yAnnotations: annotations,
			xLabels:      labels,
			yLabels:      k8sKServeAndIrrelevantLabelsWithDifferentIrrelevantValue,
			expectedDiff: false,
		},
		{
			name:         "deployment spec template with a different relevant labels should return diff",
			xAnnotations: annotations,
			yAnnotations: annotations,
			xLabels:      labels,
			yLabels:      k8sKServeAndIrrelevantLabelsWithDifferentRelevantValues,
			expectedDiff: true,
		},
		{
			name:         "deployment spec template with a different relevant and irrelevant labels should return diff",
			xAnnotations: annotations,
			yAnnotations: annotations,
			xLabels:      labels,
			yLabels:      k8sKServeAndIrrelevantLabelsWithDifferentRelevantAndIrrelevantValues,
			expectedDiff: true,
		},
		// missing labels
		{
			name:         "deployment spec template with a missing KServe label should return diff",
			xAnnotations: annotations,
			yAnnotations: annotations,
			xLabels:      labels,
			yLabels:      labelsWithMissingKServeKey,
			expectedDiff: true,
		},
		{
			name:         "deployment spec template with a missing K8s label should return diff",
			xAnnotations: annotations,
			yAnnotations: annotations,
			xLabels:      labels,
			yLabels:      labelsWithMissingK8sKey,
			expectedDiff: true,
		},
		{
			name:         "deployment spec template with a missing irrelevant label should return no diff",
			xAnnotations: annotations,
			yAnnotations: annotations,
			xLabels:      labels,
			yLabels:      labelsWithMissingIrrelevantKey,
			expectedDiff: false,
		},
		// additional labels
		{
			name:         "deployment spec template with an additional KServe label should return diff",
			xAnnotations: annotations,
			yAnnotations: annotations,
			xLabels:      labels,
			yLabels:      labelsWithAdditionalKServeKey,
			expectedDiff: true,
		},
		{
			name:         "deployment spec template with an additional K8s label should return diff",
			xAnnotations: annotations,
			yAnnotations: annotations,
			xLabels:      labels,
			yLabels:      labelsWithAdditionalK8sKey,
			expectedDiff: true,
		},
		{
			name:         "deployment spec template with an additional irrelevant label should return no diff",
			xAnnotations: annotations,
			yAnnotations: annotations,
			xLabels:      labels,
			yLabels:      labelsWithAdditionalIrrelevantKey,
			expectedDiff: false,
		},
		// differences in both annotations and labels
		{
			name:         "deployment spec template with different KServe labels and annotations should return no diff",
			xAnnotations: annotations,
			yAnnotations: annotationsWithDifferentKServeValue,
			xLabels:      labels,
			yLabels:      k8sKServeAndIrrelevantLabelsWithDifferentKServeValue,
			expectedDiff: true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// define deployment spec template X
			x := corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:        name,
					Namespace:   namespace,
					Labels:      tc.xLabels,
					Annotations: tc.xAnnotations,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Name:      containerName,
						Image:     imageName,
						Command:   []string{commandName},
						Resources: corev1.ResourceRequirements{},
					}},
					ServiceAccountName: serviceAccountName,
				},
			}

			// define deployment spec template Y
			y := corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:        name,
					Namespace:   namespace,
					Labels:      tc.yLabels,
					Annotations: tc.yAnnotations,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Name:      containerName,
						Image:     imageName,
						Command:   []string{commandName},
						Resources: corev1.ResourceRequirements{},
					}},
					ServiceAccountName: serviceAccountName,
				},
			}

			// check if compareMetadata(x, y) returns difference
			diff, _ := compareMetadata(x, y)
			// was there a difference
			actualDiff := diff != ""
			if !(actualDiff == tc.expectedDiff) {
				t.Errorf("Test %q unexpected status (-expectedDiff +actualDiff): %v", tc.name, actualDiff)
			}
		})
	}
}

func copyMap(originalMap map[string]string) (newMap map[string]string) {
	newMap = make(map[string]string)
	for k, v := range originalMap {
		newMap[k] = v
	}
	return newMap
}
