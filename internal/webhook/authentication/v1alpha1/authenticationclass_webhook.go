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

	authenticationv1alpha1 "github.com/zncdatadev/operator-go/pkg/apis/authentication/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// nolint:unused
// log is for logging in this package.
var authenticationclasslog = logf.Log.WithName("authenticationclass-resource")

// SetupAuthenticationClassWebhookWithManager registers the webhook for AuthenticationClass in the manager.
func SetupAuthenticationClassWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).For(&authenticationv1alpha1.AuthenticationClass{}).
		WithValidator(&AuthenticationClassCustomValidator{}).
		WithDefaulter(&AuthenticationClassCustomDefaulter{}).
		Complete()
}

// TODO(user): EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

// +kubebuilder:webhook:path=/mutate-authentication-kubedoop-dev-v1alpha1-authenticationclass,mutating=true,failurePolicy=fail,sideEffects=None,groups=authentication.kubedoop.dev,resources=authenticationclasses,verbs=create;update,versions=v1alpha1,name=mauthenticationclass-v1alpha1.kb.io,admissionReviewVersions=v1

// AuthenticationClassCustomDefaulter struct is responsible for setting default values on the custom resource of the
// Kind AuthenticationClass when those are created or updated.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as it is used only for temporary operations and does not need to be deeply copied.
type AuthenticationClassCustomDefaulter struct {
	// TODO(user): Add more fields as needed for defaulting
}

var _ webhook.CustomDefaulter = &AuthenticationClassCustomDefaulter{}

// Default implements webhook.CustomDefaulter so a webhook will be registered for the Kind AuthenticationClass.
func (d *AuthenticationClassCustomDefaulter) Default(ctx context.Context, obj runtime.Object) error {
	authenticationclass, ok := obj.(*authenticationv1alpha1.AuthenticationClass)

	if !ok {
		return fmt.Errorf("expected an AuthenticationClass object but got %T", obj)
	}
	authenticationclasslog.Info("Defaulting for AuthenticationClass", "name", authenticationclass.GetName())

	// TODO(user): fill in your defaulting logic.

	return nil
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
// NOTE: The 'path' attribute must follow a specific pattern and should not be modified directly here.
// Modifying the path for an invalid path can cause API server errors; failing to locate the webhook.
// +kubebuilder:webhook:path=/validate-authentication-kubedoop-dev-v1alpha1-authenticationclass,mutating=false,failurePolicy=fail,sideEffects=None,groups=authentication.kubedoop.dev,resources=authenticationclasses,verbs=create;update,versions=v1alpha1,name=vauthenticationclass-v1alpha1.kb.io,admissionReviewVersions=v1

// AuthenticationClassCustomValidator struct is responsible for validating the AuthenticationClass resource
// when it is created, updated, or deleted.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as this struct is used only for temporary operations and does not need to be deeply copied.
type AuthenticationClassCustomValidator struct {
	// TODO(user): Add more fields as needed for validation
}

var _ webhook.CustomValidator = &AuthenticationClassCustomValidator{}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type AuthenticationClass.
func (v *AuthenticationClassCustomValidator) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	authenticationclass, ok := obj.(*authenticationv1alpha1.AuthenticationClass)
	if !ok {
		return nil, fmt.Errorf("expected a AuthenticationClass object but got %T", obj)
	}
	authenticationclasslog.Info("Validation for AuthenticationClass upon creation", "name", authenticationclass.GetName())

	// TODO(user): fill in your validation logic upon object creation.

	return nil, nil
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type AuthenticationClass.
func (v *AuthenticationClassCustomValidator) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	authenticationclass, ok := newObj.(*authenticationv1alpha1.AuthenticationClass)
	if !ok {
		return nil, fmt.Errorf("expected a AuthenticationClass object for the newObj but got %T", newObj)
	}
	authenticationclasslog.Info("Validation for AuthenticationClass upon update", "name", authenticationclass.GetName())

	// TODO(user): fill in your validation logic upon object update.

	return nil, nil
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type AuthenticationClass.
func (v *AuthenticationClassCustomValidator) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	authenticationclass, ok := obj.(*authenticationv1alpha1.AuthenticationClass)
	if !ok {
		return nil, fmt.Errorf("expected a AuthenticationClass object but got %T", obj)
	}
	authenticationclasslog.Info("Validation for AuthenticationClass upon deletion", "name", authenticationclass.GetName())

	// TODO(user): fill in your validation logic upon object deletion.

	return nil, nil
}
