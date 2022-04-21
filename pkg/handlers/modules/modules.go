package modules

import (
	"fmt"
	"strings"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer/yaml"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
)

const (
	moduleConfigurationGroupAndKind = "Configuration.pkg.crossplane.io"
	moduleClaimsGroupSuffix         = "krateo.io"
)

func installModulePackage(rc *rest.Config, data []byte) error {
	dc, err := dynamic.NewForConfig(rc)
	if err != nil {
		return err
	}

	obj := &unstructured.Unstructured{}
	dec := yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme)
	_, gvk, err := dec.Decode(data, nil, obj)
	if err != nil {
		return err
	}

	if gvk.GroupKind().String() != moduleConfigurationGroupAndKind {
		return fmt.Errorf("kind: %s in apiGroup: %s is not allowed", gvk.Kind, gvk.Group)
	}

	err = createOrUpdateResourceFromUnstructured(rc, dc, obj)
	if err != nil {
		return err
	}

	err = waitForModuleCRDs(rc, gvk)
	if err != nil {
		return err
	}

	return nil
}

func installModuleClaims(rc *rest.Config, data []byte) error {
	dc, err := dynamic.NewForConfig(rc)
	if err != nil {
		return err
	}

	obj := &unstructured.Unstructured{}
	dec := yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme)
	_, gvk, err := dec.Decode(data, nil, obj)
	if err != nil {
		return err
	}

	if g := gvk.GroupKind().Group; !strings.HasSuffix(g, moduleClaimsGroupSuffix) {
		return fmt.Errorf("apiGroup: %s is not allowed", g)
	}

	err = createOrUpdateResourceFromUnstructured(rc, dc, obj)
	if err != nil {
		return err
	}

	return nil
}

func waitForModuleCRDs(rc *rest.Config, gvk *schema.GroupVersionKind) error {
	return waitForCRDs(rc, []*apiextensionsv1.CustomResourceDefinition{
		{
			Spec: apiextensionsv1.CustomResourceDefinitionSpec{
				Group: gvk.Group,

				Versions: []apiextensionsv1.CustomResourceDefinitionVersion{
					{
						Name:    gvk.Version,
						Served:  false,
						Storage: false,
					},
				},
				Names: apiextensionsv1.CustomResourceDefinitionNames{
					Kind: gvk.Kind,
					//Plural: strings.ToLower(gvk.Kind),
				},
			},
		},
	},
	)
}
