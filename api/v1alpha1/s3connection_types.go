/*
Copyright 2023 zncdata-labs.

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
	operatorv1 "github.com/zncdata-labs/operator-go/pkg/operators/v1alphav1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// S3Connection is the Schema for the s3connections API
type S3Connection struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	operatorv1.S3Connection `json:",inline"`
}

//+kubebuilder:object:root=true

// S3ConnectionList contains a list of S3Connection
type S3ConnectionList struct {
	metav1.TypeMeta             `json:",inline"`
	metav1.ListMeta             `json:"metadata,omitempty"`
	operatorv1.S3ConnectionList `json:",inline"`
}

func init() {
	SchemeBuilder.Register(&S3Connection{}, &S3ConnectionList{})
}
