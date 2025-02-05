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

package v1beta1

import (
	"errors"
	"fmt"
	"k8s.io/apimachinery/pkg/runtime"
	"regexp"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// log is for logging in this package.
var modulelog = logf.Log.WithName("module-resource")

func (m *Module) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(m).
		Complete()
}

//+kubebuilder:webhook:path=/validate-kmm-sigs-x-k8s-io-v1beta1-module,mutating=false,failurePolicy=fail,sideEffects=None,groups=kmm.sigs.x-k8s.io,resources=modules,verbs=create;update,versions=v1beta1,name=vmodule.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &Module{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (m *Module) ValidateCreate() error {
	modulelog.Info("Validating Module creation", "name", m.Name, "namespace", m.Namespace)

	if err := m.validateKernelMapping(); err != nil {
		return fmt.Errorf("failed to validate kernel mappings: %v", err)
	}

	return m.validateModprobe()
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (m *Module) ValidateUpdate(_ runtime.Object) error {
	modulelog.Info("Validating Module update", "name", m.Name, "namespace", m.Namespace)

	if err := m.validateKernelMapping(); err != nil {
		return fmt.Errorf("failed to validate kernel mappings: %v", err)
	}

	return m.validateModprobe()
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (m *Module) ValidateDelete() error {
	return nil
}

func (m *Module) validateKernelMapping() error {
	container := m.Spec.ModuleLoader.Container

	for idx, km := range container.KernelMappings {
		if km.Regexp != "" && km.Literal != "" {
			return fmt.Errorf("regexp and literal are mutually exclusive properties at kernelMappings[%d]", idx)
		}

		if km.Regexp == "" && km.Literal == "" {
			return fmt.Errorf("regexp or literal must be set at kernelMappings[%d]", idx)
		}

		if _, err := regexp.Compile(km.Regexp); err != nil {
			return fmt.Errorf("invalid regexp at index %d: %v", idx, err)
		}

		if container.ContainerImage == "" && km.ContainerImage == "" {
			return fmt.Errorf("missing spec.moduleLoader.container.kernelMappings[%d].containerImage", idx)
		}
	}

	return nil
}

func (m *Module) validateModprobe() error {
	modprobe := m.Spec.ModuleLoader.Container.Modprobe
	moduleNameDefined := modprobe.ModuleName != ""
	rawLoadArgsDefined := modprobe.RawArgs != nil && len(modprobe.RawArgs.Load) > 0
	rawUnloadArgsDefined := modprobe.RawArgs != nil && len(modprobe.RawArgs.Unload) > 0

	if moduleNameDefined {
		if rawLoadArgsDefined || rawUnloadArgsDefined {
			return errors.New("rawArgs cannot be set when moduleName is set")
		}
		return nil
	}

	if !rawLoadArgsDefined || !rawUnloadArgsDefined {
		return errors.New("load and unload rawArgs must be set when moduleName is unset")
	}

	return nil
}
