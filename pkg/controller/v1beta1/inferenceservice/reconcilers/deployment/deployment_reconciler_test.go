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

	targetUtilizationPercentageAnnotation := "serving.kserve.io/targetUtilizationPercentage"
	deploymentRevisionAnnotation := "deployment.kubernetes.io/revision"
	irrelevantAnnotation := "annotation.custom.io/my-annotation"
	storageInitializerSourceUriInternalAnnotation := "internal.serving.kserve.io/storage-initializer-sourceuri"
	autoscalerClassAnnotation := "serving.kserve.io/autoscalerClass"
	deploymentModeAnnotation := "serving.kserve.io/deploymentMode"
	autoscalerMetricsAnnotation := "serving.kserve.io/metrics"

	annotations := map[string]string{
		deploymentRevisionAnnotation:                  "2",
		storageInitializerSourceUriInternalAnnotation: "s3://my-s3-1234567890-us-west-2/files/my-data/my-file",
		autoscalerClassAnnotation:                     "hpa",
		deploymentModeAnnotation:                      "RawDeployment",
		autoscalerMetricsAnnotation:                   "cpu",
		targetUtilizationPercentageAnnotation:         "80",
		irrelevantAnnotation:                          "hello",
	}

	annotationsWithDifferentKServeValue := copyMap(annotations)
	annotationsWithDifferentKServeValue[targetUtilizationPercentageAnnotation] = "90"

	annotationsWithDifferentIrrelevantValue := copyMap(annotations)
	annotationsWithDifferentIrrelevantValue[irrelevantAnnotation] = "goodbye"

	annotationsWithDifferentRelevantValues := map[string]string{
		deploymentRevisionAnnotation:                  "3",
		storageInitializerSourceUriInternalAnnotation: "s3://my-s3-1234567890-us-west-2/files/my-data/my-file-2",
		autoscalerClassAnnotation:                     "hpa",
		deploymentModeAnnotation:                      "Serverless",
		autoscalerMetricsAnnotation:                   "memory",
		targetUtilizationPercentageAnnotation:         "90",
		irrelevantAnnotation:                          "hello",
	}

	annotationsWithDifferentRelevantAndIrrelevantValues := copyMap(annotationsWithDifferentRelevantValues)
	annotationsWithDifferentRelevantAndIrrelevantValues[irrelevantAnnotation] = "goodbye"

	annotationsWithMissingKServeAnnotation := copyMap(annotations)
	delete(annotationsWithMissingKServeAnnotation, storageInitializerSourceUriInternalAnnotation)

	annotationsWithMissingIrrelevantAnnotation := copyMap(annotations)
	delete(annotationsWithMissingIrrelevantAnnotation, irrelevantAnnotation)

	annotationsWithAdditionalKServeAnnotation := copyMap(annotations)
	annotationsWithAdditionalKServeAnnotation["serving.kserve.io/enable-tag-routing"] = "true"

	annotationsWithAdditionalIrrelevantAnnotation := copyMap(annotations)
	annotationsWithAdditionalIrrelevantAnnotation["annotation.custom.io/my-other-annotation"] = "hello-again"

	kServiceComponentLabel := "component"
	kServiceModelLabel := "model"
	irrelevantLabel := "label.custom.io/my-label"

	labels := map[string]string{
		kServiceComponentLabel: "predictor",
		kServiceModelLabel:     "sklearn",
		irrelevantLabel:        "hello",
	}

	KServeAndIrrelevantLabelsWithDifferentKServeValue := copyMap(labels)
	KServeAndIrrelevantLabelsWithDifferentKServeValue[kServiceComponentLabel] = "transformer"

	KServeAndIrrelevantLabelsWithDifferentIrrelevantValue := copyMap(labels)
	KServeAndIrrelevantLabelsWithDifferentIrrelevantValue[irrelevantLabel] = "goodbye"

	KServeAndIrrelevantLabelsWithDifferentRelevantValues := map[string]string{
		kServiceComponentLabel: "transformer",
		kServiceModelLabel:     "xgboost",
		irrelevantLabel:        "hello",
	}

	KServeAndIrrelevantLabelsWithDifferentRelevantAndIrrelevantValues := copyMap(KServeAndIrrelevantLabelsWithDifferentRelevantValues)
	KServeAndIrrelevantLabelsWithDifferentRelevantAndIrrelevantValues[irrelevantLabel] = "goodbye"

	labelsWithMissingKServeLabel := copyMap(labels)
	delete(labelsWithMissingKServeLabel, kServiceComponentLabel)

	labelsWithMissingIrrelevantLabel := copyMap(labels)
	delete(labelsWithMissingIrrelevantLabel, irrelevantLabel)

	labelsWithAdditionalKServeLabel := copyMap(labels)
	labelsWithAdditionalKServeLabel["endpoint"] = "default"

	labelsWithAdditionalIrrelevantLabel := copyMap(labels)
	labelsWithAdditionalIrrelevantLabel["label.custom.io/my-other-label"] = "hello-again"

	cases := []struct {
		name         string
		xAnnotations map[string]string
		yAnnotations map[string]string
		xLabels      map[string]string
		yLabels      map[string]string
		expectedDiff bool
	}{
		{
			name:         "identical deployment spec templates should return no diff",
			xAnnotations: annotations,
			yAnnotations: annotations,
			xLabels:      labels,
			yLabels:      labels,
			expectedDiff: false,
		},
		{
			name:         "deployment spec template with a different KServe annotation should return diff",
			xAnnotations: annotations,
			yAnnotations: annotationsWithDifferentKServeValue,
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
		{
			name:         "deployment spec template with a missing KServe annotation should return diff",
			xAnnotations: annotations,
			yAnnotations: annotationsWithMissingKServeAnnotation,
			xLabels:      labels,
			yLabels:      labels,
			expectedDiff: true,
		},
		{
			name:         "deployment spec template with a missing irrelevant annotation should return no diff",
			xAnnotations: annotations,
			yAnnotations: annotationsWithMissingIrrelevantAnnotation,
			xLabels:      labels,
			yLabels:      labels,
			expectedDiff: false,
		},
		{
			name:         "deployment spec template with an additional KServe annotation should return diff",
			xAnnotations: annotations,
			yAnnotations: annotationsWithAdditionalKServeAnnotation,
			xLabels:      labels,
			yLabels:      labels,
			expectedDiff: true,
		},
		{
			name:         "deployment spec template with an additional KServe annotation should return diff",
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
		{
			name:         "deployment spec template with a different KServe label should return diff",
			xAnnotations: annotations,
			yAnnotations: annotations,
			xLabels:      labels,
			yLabels:      KServeAndIrrelevantLabelsWithDifferentKServeValue,
			expectedDiff: true,
		},
		{
			name:         "deployment spec template with a different irrelevant label should return no diff",
			xAnnotations: annotations,
			yAnnotations: annotations,
			xLabels:      labels,
			yLabels:      KServeAndIrrelevantLabelsWithDifferentIrrelevantValue,
			expectedDiff: false,
		},
		{
			name:         "deployment spec template with a different relevant labels should return diff",
			xAnnotations: annotations,
			yAnnotations: annotations,
			xLabels:      labels,
			yLabels:      KServeAndIrrelevantLabelsWithDifferentRelevantValues,
			expectedDiff: true,
		},
		{
			name:         "deployment spec template with a different relevant and irrelevant labels should return diff",
			xAnnotations: annotations,
			yAnnotations: annotations,
			xLabels:      labels,
			yLabels:      KServeAndIrrelevantLabelsWithDifferentRelevantAndIrrelevantValues,
			expectedDiff: true,
		},
		{
			name:         "deployment spec template with a missing KServe label should return diff",
			xAnnotations: annotations,
			yAnnotations: annotations,
			xLabels:      labels,
			yLabels:      labelsWithMissingKServeLabel,
			expectedDiff: true,
		},
		{
			name:         "deployment spec template with a missing irrelevant label should return no diff",
			xAnnotations: annotations,
			yAnnotations: annotations,
			xLabels:      labels,
			yLabels:      labelsWithMissingIrrelevantLabel,
			expectedDiff: false,
		},
		{
			name:         "deployment spec template with an additional KServe label should return diff",
			xAnnotations: annotations,
			yAnnotations: annotations,
			xLabels:      labels,
			yLabels:      labelsWithAdditionalKServeLabel,
			expectedDiff: true,
		},
		{
			name:         "deployment spec template with an additional irrelevant label should return no diff",
			xAnnotations: annotations,
			yAnnotations: annotations,
			xLabels:      labels,
			yLabels:      labelsWithAdditionalIrrelevantLabel,
			expectedDiff: false,
		},
		{
			name:         "deployment spec template with different KServe labels and annotations should return no diff",
			xAnnotations: annotations,
			yAnnotations: annotationsWithDifferentKServeValue,
			xLabels:      labels,
			yLabels:      KServeAndIrrelevantLabelsWithDifferentKServeValue,
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
			diff, err := compareMetadata(x, y)
			// was there a difference
			actualDiff := diff != ""
			if !((actualDiff == tc.expectedDiff) && err == nil) {
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
