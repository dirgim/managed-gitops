/*
Copyright 2022.

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

package v1alpha1

import (
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"time"
)

// ApplicationSnapshotSpec defines the desired state of ApplicationSnapshot
type ApplicationSnapshotSpec struct {

	// NOTE: The name (kind) of this API resource, "ApplicationSnapshot" is likely to change in the short term (Q2 2022).
	// Stay tuned for refactoring needed for your component.

	// Application is a reference to the name of an Application resource within the same namespace, which defines the target application for the Snapshot (when used with a Binding).
	Application string `json:"application"`

	// DisplayName is a user-visible, user-definable name for the resource (and is not used for any functional behaviour)
	DisplayName string `json:"displayName,omitempty"`

	// DisplayDescription is a user-visible, user definable description for the resource (and is not used for any functional behaviour)
	DisplayDescription string `json:"displayDescription,omitempty"`

	// Type is an optional definiton of how the ApplicationSnapshot was constructed
	Type string `json:"type,omitempty"`

	// Components field contains the sets of components to deploy as part of this snapshot.
	Components []ApplicationSnapshotComponent `json:"components,omitempty"`

	// Artifacts is a placeholder section for 'artifact links' we want to maintain to other AppStudio resources.
	// See Environment API doc for details.
	Artifacts SnapshotArtifacts `json:"artifacts,omitempty"`
}

// ApplicationSnapshotReason represents a reason for the release "Succeeded" condition
type ApplicationSnapshotReason string

const (
	// applicationSnapshotConditionType is the type used when setting a release status condition
	applicationSnapshotConditionType string = "Succeeded"

	// ApplicationSnapshotReasonInitialized is the reason set when ApplicationSnapshot is initialized
	ApplicationSnapshotReasonInitialized ApplicationSnapshotReason = "Initialized"

	// ApplicationSnapshotReasonValidationError is the reason set when ApplicationSnapshot validation errored
	ApplicationSnapshotReasonValidationError ApplicationSnapshotReason = "Error"

	// ApplicationSnapshotReasonTestsFailed is the reason set when ApplicationSnapshot integration tests failed
	ApplicationSnapshotReasonTestsFailed ApplicationSnapshotReason = "TestsFailed"

	// ApplicationSnapshotReasonTestsRunning is the reason set when ApplicationSnapshot integration tests are running
	ApplicationSnapshotReasonTestsRunning ApplicationSnapshotReason = "TestsRunning"

	// ApplicationSnapshotReasonSucceeded is the reason set when the integration test PipelineRun has succeeded
	ApplicationSnapshotReasonSucceeded ApplicationSnapshotReason = "Succeeded"
)

func (asr ApplicationSnapshotReason) String() string {
	return string(asr)
}

// ApplicationSnapshotComponent
type ApplicationSnapshotComponent struct {

	// Name is the name of the component
	Name string `json:"name"`

	// ContainerImage is the container image to use when deploying the component, as part of a Snapshot
	ContainerImage string `json:"containerImage"`
}

// SnapshotArtifacts is a placeholder section for 'artifact links' we want to maintain to other AppStudio resources.
//
// For example: here I'm imagining we might want to keep track of container image <=> (source code repo, commit sha) links,
// Which might be useful to present to the user within the UI.
type SnapshotArtifacts struct {

	// NOTE: This field (and struct) are placeholders.
	// - Until this API is stabilized, consumers of the API may store any unstructured JSON/YAML data here,
	//   but no backwards compatibility will be preserved.
	UnstableFields *apiextensionsv1.JSON `json:"unstableFields,omitempty"`
}

// ApplicationSnapshotStatus defines the observed state of ApplicationSnapshot
type ApplicationSnapshotStatus struct {
	// StartTime is the time when the Release PipelineRun was created and set to run
	// +optional
	StartTime *metav1.Time `json:"startTime,omitempty"`

	// CompletionTime is the time the Release PipelineRun completed
	// +optional
	CompletionTime *metav1.Time `json:"completionTime,omitempty"`

	// Conditions represent the latest available observations for the release
	// +optional
	Conditions []metav1.Condition `json:"conditions"`

	// ReleasePipelineRun contains the namespaced name of the release PipelineRun executed as part of this release
	// +kubebuilder:validation:Pattern=^[a-z0-9]([-a-z0-9]*[a-z0-9])?\/[a-z0-9]([-a-z0-9]*[a-z0-9])?$
	// +optional
	ReleasePipelineRun string `json:"releasePipelineRun,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Succeeded",type=string,JSONPath=`.status.conditions[?(@.type=="Succeeded")].status`
//+kubebuilder:printcolumn:name="Reason",type=string,JSONPath=`.status.conditions[?(@.type=="Succeeded")].reason`

// ApplicationSnapshot is the Schema for the applicationsnapshots API
type ApplicationSnapshot struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ApplicationSnapshotSpec   `json:"spec,omitempty"`
	Status ApplicationSnapshotStatus `json:"status,omitempty"`
}

// HasStarted checks whether the ApplicationSnapshot has a valid start time set in its status.
func (a *ApplicationSnapshot) HasStarted() bool {
	return a.Status.StartTime != nil && !a.Status.StartTime.IsZero()
}

// HasSucceeded checks whether the ApplicationSnapshot has succeeded or not.
func (a *ApplicationSnapshot) HasSucceeded() bool {
	return !meta.IsStatusConditionTrue(a.Status.Conditions, applicationSnapshotConditionType)
}

// IsDone returns a boolean indicating whether the ApplicationSnapshot's status indicates that it is done or not.
func (a *ApplicationSnapshot) IsDone() bool {
	condition := meta.FindStatusCondition(a.Status.Conditions, applicationSnapshotConditionType)
	if condition != nil {
		return condition.Status != metav1.ConditionUnknown
	}

	return false
}

// MarkFailed registers the completion time and changes the Succeeded condition to False with
// the provided reason and message.
func (a *ApplicationSnapshot) MarkFailed(reason ApplicationSnapshotReason, message string) {
	if a.IsDone() && a.Status.CompletionTime != nil {
		return
	}

	a.Status.CompletionTime = &metav1.Time{Time: time.Now()}
	a.setStatusConditionWithMessage(metav1.ConditionFalse, reason, message)

}

// MarkInvalid changes the Succeeded condition to False with the provided reason and message.
func (a *ApplicationSnapshot) MarkInvalid(reason ApplicationSnapshotReason, message string) {
	if a.IsDone() {
		return
	}

	a.setStatusConditionWithMessage(metav1.ConditionFalse, reason, message)
}

// MarkRunning registers the start time and changes the Succeeded condition to Unknown.
func (a *ApplicationSnapshot) MarkRunning() {
	if a.HasStarted() && a.Status.StartTime != nil {
		return
	}

	a.Status.StartTime = &metav1.Time{Time: time.Now()}
	a.setStatusCondition(metav1.ConditionUnknown, ApplicationSnapshotReasonTestsRunning)
}

// MarkSucceeded registers the completion time and changes the Succeeded condition to True.
func (a *ApplicationSnapshot) MarkSucceeded() {
	if a.IsDone() && a.Status.CompletionTime != nil {
		return
	}

	a.Status.CompletionTime = &metav1.Time{Time: time.Now()}
	a.setStatusCondition(metav1.ConditionTrue, ApplicationSnapshotReasonSucceeded)
}

// SetCondition creates a new condition with the given status and reason. Then, it sets this new condition,
// unsetting previous conditions with the same type as necessary.
func (a *ApplicationSnapshot) setStatusCondition(status metav1.ConditionStatus, reason ApplicationSnapshotReason) {
	a.setStatusConditionWithMessage(status, reason, "")
}

// SetCondition creates a new condition with the given status, reason and message. Then, it sets this new condition,
// unsetting previous conditions with the same type as necessary.
func (a *ApplicationSnapshot) setStatusConditionWithMessage(status metav1.ConditionStatus, reason ApplicationSnapshotReason, message string) {
	meta.SetStatusCondition(&a.Status.Conditions, metav1.Condition{
		Type:    applicationSnapshotConditionType,
		Status:  status,
		Reason:  reason.String(),
		Message: message,
	})
}

//+kubebuilder:object:root=true

// ApplicationSnapshotList contains a list of ApplicationSnapshot
type ApplicationSnapshotList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ApplicationSnapshot `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ApplicationSnapshot{}, &ApplicationSnapshotList{})
}
