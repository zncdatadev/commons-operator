/*
Copyright 2023 zncdatadev.

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
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	s3v1alpha1 "github.com/zncdatadev/operator-go/pkg/apis/s3/v1alpha1"
)

// nolint:unused
// log is for logging in this package.
var s3bucketlog = logf.Log.WithName("s3bucket-resource")

// SetupS3BucketWebhookWithManager registers the webhook for S3Bucket in the manager.
func SetupS3BucketWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).For(&s3v1alpha1.S3Bucket{}).
		WithValidator(&S3BucketCustomValidator{}).
		WithDefaulter(&S3BucketCustomDefaulter{}).
		Complete()
}

// TODO(user): EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

// +kubebuilder:webhook:path=/mutate-s3-kubedoop-dev-v1alpha1-s3bucket,mutating=true,failurePolicy=fail,sideEffects=None,groups=s3.kubedoop.dev,resources=s3buckets,verbs=create;update,versions=v1alpha1,name=ms3bucket-v1alpha1.kb.io,admissionReviewVersions=v1

// S3BucketCustomDefaulter struct is responsible for setting default values on the custom resource of the
// Kind S3Bucket when those are created or updated.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as it is used only for temporary operations and does not need to be deeply copied.
type S3BucketCustomDefaulter struct {
	// TODO(user): Add more fields as needed for defaulting
}

var _ webhook.CustomDefaulter = &S3BucketCustomDefaulter{}

// Default implements webhook.CustomDefaulter so a webhook will be registered for the Kind S3Bucket.
func (d *S3BucketCustomDefaulter) Default(ctx context.Context, obj runtime.Object) error {
	s3bucket, ok := obj.(*s3v1alpha1.S3Bucket)

	if !ok {
		return fmt.Errorf("expected an S3Bucket object but got %T", obj)
	}
	s3bucketlog.Info("Defaulting for S3Bucket", "name", s3bucket.GetName())

	// TODO(user): fill in your defaulting logic.

	return nil
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
// NOTE: The 'path' attribute must follow a specific pattern and should not be modified directly here.
// Modifying the path for an invalid path can cause API server errors; failing to locate the webhook.
// +kubebuilder:webhook:path=/validate-s3-kubedoop-dev-v1alpha1-s3bucket,mutating=false,failurePolicy=fail,sideEffects=None,groups=s3.kubedoop.dev,resources=s3buckets,verbs=create;update,versions=v1alpha1,name=vs3bucket-v1alpha1.kb.io,admissionReviewVersions=v1

// S3BucketCustomValidator struct is responsible for validating the S3Bucket resource
// when it is created, updated, or deleted.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as this struct is used only for temporary operations and does not need to be deeply copied.
type S3BucketCustomValidator struct {
	// TODO(user): Add more fields as needed for validation
}

var _ webhook.CustomValidator = &S3BucketCustomValidator{}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type S3Bucket.
func (v *S3BucketCustomValidator) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	s3bucket, ok := obj.(*s3v1alpha1.S3Bucket)
	if !ok {
		return nil, fmt.Errorf("expected a S3Bucket object but got %T", obj)
	}
	s3bucketlog.Info("Validation for S3Bucket upon creation", "name", s3bucket.GetName())

	// TODO(user): fill in your validation logic upon object creation.

	return nil, nil
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type S3Bucket.
func (v *S3BucketCustomValidator) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	s3bucket, ok := newObj.(*s3v1alpha1.S3Bucket)
	if !ok {
		return nil, fmt.Errorf("expected a S3Bucket object for the newObj but got %T", newObj)
	}
	s3bucketlog.Info("Validation for S3Bucket upon update", "name", s3bucket.GetName())

	// TODO(user): fill in your validation logic upon object update.

	return nil, nil
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type S3Bucket.
func (v *S3BucketCustomValidator) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	s3bucket, ok := obj.(*s3v1alpha1.S3Bucket)
	if !ok {
		return nil, fmt.Errorf("expected a S3Bucket object but got %T", obj)
	}
	s3bucketlog.Info("Validation for S3Bucket upon deletion", "name", s3bucket.GetName())

	// TODO(user): fill in your validation logic upon object deletion.

	return nil, nil
}
