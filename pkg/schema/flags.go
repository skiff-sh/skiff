package schema

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/urfave/cli/v3"

	"github.com/skiff-sh/api/go/skiff/registry/v1alpha1"

	"github.com/skiff-sh/skiff/pkg/collection"
	"github.com/skiff-sh/skiff/pkg/fields"
)

var _ Entry = (*Flag)(nil)

type Flag struct {
	Package  string
	Accessor FlagAccessor
	Flag     cli.Flag
	Field    *Field
}

func (f *Flag) FieldName() string {
	return f.Field.Proto.GetName()
}

func (f *Flag) Value() Value {
	return NewValidatedValFromField(f.Flag.Get(), f.Field.Proto)
}

// FlagAccessor allows for all implementations of cli.Flag to have their non-generic fields read/written.
type FlagAccessor interface {
	SetName(name string)
	SetAliases(aliases []string)
	SetRequired(b bool)
	SetCategory(s string)

	Name() string
	Aliases() []string
	Required() bool
	Category() string
}

type flagAccessor[T any, C any, VC cli.ValueCreator[T, C]] struct {
	F *cli.FlagBase[T, C, VC]
}

func (f *flagAccessor[T, C, VC]) SetCategory(s string) {
	f.F.Category = s
}

func (f *flagAccessor[T, C, VC]) Category() string {
	return f.F.Category
}

func (f *flagAccessor[T, C, VC]) SetRequired(b bool) { f.F.Required = b }

func (f *flagAccessor[T, C, VC]) Required() bool { return f.F.Required }

func (f *flagAccessor[T, C, VC]) Name() string { return f.F.Name }

func (f *flagAccessor[T, C, VC]) Aliases() []string { return f.F.Aliases }

func (f *flagAccessor[T, C, VC]) SetName(name string) { f.F.Name = name }

func (f *flagAccessor[T, C, VC]) SetAliases(aliases []string) { f.F.Aliases = aliases }

func newFlagAccessor[T any, C any, VC cli.ValueCreator[T, C]](f *cli.FlagBase[T, C, VC]) FlagAccessor {
	return &flagAccessor[T, C, VC]{F: f}
}

func FieldToCLIFlag(f *Field) *Flag {
	switch f.Proto.GetType() {
	case v1alpha1.Field_string:
		enumVals := collection.Map(f.Enum, fields.Cast[string])
		out := &cli.StringFlag{
			Name:  f.Proto.GetName(),
			Usage: f.Proto.GetDescription(),
			Value: fields.Cast[string](f.Default),
		}

		if len(enumVals) > 0 {
			out.Action = func(_ context.Context, _ *cli.Command, val string) error {
				idx := slices.Index(enumVals, val)
				if idx < 0 {
					return fmt.Errorf(
						"%s cannot be '%s': expected one of %s",
						f.Proto.GetName(),
						val,
						strings.Join(enumVals, ", "),
					)
				}
				return nil
			}
		}

		return &Flag{Field: f, Flag: out, Accessor: newFlagAccessor(out)}
	case v1alpha1.Field_number:
		enumVals := collection.Map(f.Enum, fields.Cast[float64])
		out := &cli.Float64Flag{
			Name:  f.Proto.GetName(),
			Usage: f.Proto.GetDescription(),
			Value: fields.Cast[float64](f.Default),
		}

		if len(enumVals) > 0 {
			out.Action = func(_ context.Context, _ *cli.Command, val float64) error {
				idx := slices.Index(enumVals, val)
				if idx < 0 {
					return fmt.Errorf(
						"%s cannot be '%v': expected one of %s",
						f.Proto.GetName(),
						val,
						strings.Join(collection.Map(enumVals, fields.FormatFloat), ", "),
					)
				}
				return nil
			}
		}

		return &Flag{Field: f, Flag: out, Accessor: newFlagAccessor(out)}
	case v1alpha1.Field_bool:
		out := &cli.BoolFlag{
			Name:  f.Proto.GetName(),
			Usage: f.Proto.GetDescription(),
			Value: fields.Cast[bool](f.Default),
		}
		return &Flag{Field: f, Flag: out, Accessor: newFlagAccessor(out)}
	case v1alpha1.Field_array:
		//nolint:exhaustive // can only be a subset.
		switch f.Proto.GetItems().GetType() {
		case v1alpha1.Field_string:
			enumVals := collection.Map(f.Enum, fields.Cast[string])
			out := &cli.StringSliceFlag{
				Name:  f.Proto.GetName(),
				Usage: f.Proto.GetDescription(),
				Value: collection.Map(fields.Cast[[]any](f.Default), fields.Cast[string]),
			}

			if len(enumVals) > 0 {
				out.Action = func(_ context.Context, _ *cli.Command, val []string) error {
					for _, v := range val {
						idx := slices.Index(enumVals, v)
						if idx < 0 {
							return fmt.Errorf(
								"%s cannot be %s: expected one of %s",
								f.Proto.GetName(),
								v,
								strings.Join(enumVals, ", "),
							)
						}
					}
					return nil
				}
			}
			return &Flag{Field: f, Flag: out, Accessor: newFlagAccessor(out)}
		case v1alpha1.Field_number:
			enumVals := collection.Map(f.Enum, fields.Cast[float64])
			out := &cli.Float64SliceFlag{
				Name:  f.Proto.GetName(),
				Usage: f.Proto.GetDescription(),
				Value: collection.Map(fields.Cast[[]any](f.Default), fields.Cast[float64]),
			}

			if len(enumVals) > 0 {
				out.Action = func(_ context.Context, _ *cli.Command, val []float64) error {
					for _, v := range val {
						idx := slices.Index(enumVals, v)
						if idx < 0 {
							return fmt.Errorf(
								"%s cannot be %v: expected one of %s",
								f.Proto.GetName(),
								v,
								strings.Join(collection.Map(enumVals, fields.FormatFloat), ", "),
							)
						}
					}
					return nil
				}
			}

			return &Flag{Field: f, Flag: out, Accessor: newFlagAccessor(out)}
		}
	}
	return nil
}
