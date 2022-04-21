package modules

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/rs/zerolog"
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

func installModulePackage(ctx context.Context, rc *rest.Config, data []byte) error {
	log := zerolog.Ctx(ctx)

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
	log.Debug().
		Str("group", gvk.Group).
		Str("version", gvk.Version).
		Str("kind", gvk.Kind).
		Str("name", obj.GetName()).
		Msg("installing package")
	err = createOrUpdateResourceFromUnstructured(rc, dc, obj)
	if err != nil {
		return err
	}

	log.Debug().
		Str("group", gvk.Group).
		Str("version", gvk.Version).
		Str("kind", gvk.Kind).
		Str("name", obj.GetName()).
		Msg("waiting for package crds")

	err = waitForModuleCRDs(rc, gvk)
	if err != nil {
		return err
	}

	log.Debug().
		Str("group", gvk.Group).
		Str("version", gvk.Version).
		Str("kind", gvk.Kind).
		Str("name", obj.GetName()).
		Msg("package installed")

	return nil
}

func decodeModulePackage(s string) (*unstructured.Unstructured, *schema.GroupVersionKind, error) {
	data, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return nil, nil, err
	}

	obj, gvk, err := decodeUnstructured(data)
	if err != nil {
		return nil, nil, err
	}

	if gvk.GroupKind().String() != moduleConfigurationGroupAndKind {
		return nil, nil, fmt.Errorf("kind: %s in apiGroup: %s is not allowed", gvk.Kind, gvk.Group)
	}

	return obj, gvk, nil
}

func decodeModuleClaim(s string) (*unstructured.Unstructured, *schema.GroupVersionKind, error) {
	data, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return nil, nil, err
	}

	obj, gvk, err := decodeUnstructured(data)
	if err != nil {
		return nil, nil, err
	}

	if g := gvk.GroupKind().Group; !strings.HasSuffix(g, moduleClaimsGroupSuffix) {
		return nil, nil, fmt.Errorf("apiGroup: %s is not allowed", g)
	}

	return obj, gvk, nil
}

func decodeUnstructured(data []byte) (*unstructured.Unstructured, *schema.GroupVersionKind, error) {
	obj := &unstructured.Unstructured{}

	dec := yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme)
	_, gvk, err := dec.Decode(data, nil, obj)
	if err != nil {
		return nil, nil, err
	}

	return obj, gvk, nil
}

func installModuleClaims(ctx context.Context, rc *rest.Config, data []byte) error {
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

func buildCRDInfo(gvk *schema.GroupVersionKind) *apiextensionsv1.CustomResourceDefinition {
	return &apiextensionsv1.CustomResourceDefinition{
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
				Kind:   gvk.Kind,
				Plural: strings.ToLower(gvk.Kind),
			},
		},
	}
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
