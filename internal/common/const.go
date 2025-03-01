package common

const (
	AnnotationChartVersion = "helm.edgeflare.io/chart-version"
	AnnotationRevision     = "helm.edgeflare.io/revision"
	AnnotationValuesHash   = "helm.edgeflare.io/values-hash"
	ConditionTypeInstalled = "Installed"
	ConditionTypeError     = "Error"
	ConditionTypeReady     = "Ready"
	LabelVersion           = "app.kubernetes.io/version"
	LabelManagedBy         = "app.kubernetes.io/managed-by"
	LabelComponent         = "app.kubernetes.io/component"
	LabelProject           = "app.kubernetes.io/project"
	ReasonReconciling      = "Reconciling"
	ReasonReady            = "Ready"
	ReasonError            = "Error"
	ReasonComponentError   = "ComponentError"
)
