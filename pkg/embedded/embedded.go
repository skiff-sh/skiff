package embedded

import (
	_ "embed"
	"encoding/json"
	"fmt"

	"github.com/google/jsonschema-go/jsonschema"
)

var (
	//go:embed jsonschema/skiff.cmd.v1alpha1.AddPackageRequest.json
	AddPackageRequestJSONSchemaContent []byte

	//go:embed jsonschema/skiff.cmd.v1alpha1.AddPackageResponse.json
	AddPackageResponseJSONSchemaContent []byte

	//go:embed jsonschema/skiff.cmd.v1alpha1.ListPackagesRequest.json
	ListPackagesRequestJSONSchemaContent []byte

	//go:embed jsonschema/skiff.cmd.v1alpha1.ListPackagesResponse.json
	ListPackagesResponseJSONSchemaContent []byte

	//go:embed jsonschema/skiff.cmd.v1alpha1.ViewPackagesRequest.json
	ViewPackagesRequestJSONSchemaContent []byte

	//go:embed jsonschema/skiff.cmd.v1alpha1.ViewPackagesResponse.json
	ViewPackagesResponseJSONSchemaContent []byte
)

type Exchange struct {
	RequestSchema  *jsonschema.Schema
	ResponseSchema *jsonschema.Schema
}

type ExchangeName int

const (
	ExchangeNameAddPackage ExchangeName = iota
	ExchangeNameListPackages
	ExchangeNameViewPackages
)

var exchangeToSchema = map[ExchangeName][2][]byte{
	ExchangeNameAddPackage: {
		AddPackageRequestJSONSchemaContent,
		AddPackageResponseJSONSchemaContent,
	},
	ExchangeNameListPackages: {
		ListPackagesRequestJSONSchemaContent,
		ListPackagesResponseJSONSchemaContent,
	},
	ExchangeNameViewPackages: {
		ViewPackagesRequestJSONSchemaContent,
		ViewPackagesResponseJSONSchemaContent,
	},
}

func LoadExchange(name ExchangeName) (*Exchange, error) {
	schemas, ok := exchangeToSchema[name]
	if !ok {
		return nil, fmt.Errorf("%d is not a valid exchange", name)
	}

	var req, resp jsonschema.Schema
	var reqRes, respRes *jsonschema.Resolved

	err := json.Unmarshal(schemas[0], &req)
	if err != nil {
		return nil, err
	}

	reqRes, err = req.Resolve(nil)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(schemas[1], &resp)
	if err != nil {
		return nil, err
	}

	respRes, err = resp.Resolve(nil)
	if err != nil {
		return nil, err
	}

	return &Exchange{
		RequestSchema:  reqRes.Schema(),
		ResponseSchema: respRes.Schema(),
	}, nil
}
