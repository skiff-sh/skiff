package plugin

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/skiff-sh/skiff/sdk-go/skiff"
	"github.com/traefik/yaegi/interp"
	"github.com/traefik/yaegi/stdlib"

	"github.com/skiff-sh/skiff/api/go/skiff/plugin/v1alpha1"
)

func NewYaegiInterpreter(plug *Plugin) (Interpreter, error) {
	o := &yaegiInterpreter{
		Interpreter: interp.New(interp.Options{
			GoPath:       "",
			BuildTags:    nil,
			Stdin:        nil,
			Stdout:       nil,
			Stderr:       nil,
			Args:         nil,
			Env:          nil,
			Unrestricted: false,
		}),
	}

	err := o.Interpreter.Use(stdlib.Symbols)
	if err != nil {
		return nil, err
	}

	err = o.Interpreter.Use(skiffSymbols)
	if err != nil {
		return nil, err
	}

	_, err = o.Interpreter.Eval(string(plug.Content))
	if err != nil {
		return nil, err
	}

	o.WriteFileFunc, err = o.Interpreter.Eval("WriteFile")
	if err != nil {
		return nil, errors.New("missing 'WriteFile' func")
	}

	return o, nil
}

var skiffSymbols = map[string]map[string]reflect.Value{
	"github.com/skiff-sh/skiff/api/go/skiff/plugin/v1alpha1/v1alpha1": {
		"WriteFileRequest":  reflect.ValueOf((*v1alpha1.WriteFileRequest)(nil)),
		"WriteFileResponse": reflect.ValueOf((*v1alpha1.WriteFileResponse)(nil)),
	},
	"github.com/skiff-sh/skiff/sdk-go/skiff/skiff": {
		"Context": reflect.ValueOf((*skiff.Context)(nil)),
	},
}

type yaegiInterpreter struct {
	Interpreter   *interp.Interpreter
	WriteFileFunc reflect.Value
}

func (y *yaegiInterpreter) WriteFile(
	ctx *Context,
	req *v1alpha1.WriteFileRequest,
) (*v1alpha1.WriteFileResponse, error) {
	args := []reflect.Value{reflect.ValueOf(ctx.Ctx), reflect.ValueOf(req)}
	results := y.WriteFileFunc.Call(args)
	if len(results) > 1 && results[1].Interface() != nil {
		err, ok := results[1].Interface().(error)
		if !ok {
			return nil, fmt.Errorf("invalid error type: %T", results[0].Interface())
		}

		return nil, err
	}

	resp, ok := results[0].Interface().(*v1alpha1.WriteFileResponse)
	if !ok {
		return nil, fmt.Errorf("invalid handler type: %T", results[0].Interface())
	}
	return resp, nil
}
