package entproto

import (
	"github.com/yoshino-s/entproto/annotations"
)

type (
	MessageOption = annotations.MessageOption
	EnumOption    = annotations.EnumOption
	FilterOption  = annotations.FilterOption
	FilterMode    = annotations.FilterMode
	FieldOption   = annotations.FieldOption
)

var (
	MessageAnnotation = annotations.MessageAnnotation
	Message           = annotations.Message
	SkipGen           = annotations.SkipGen
	PackageName       = annotations.PackageName

	EnumAnnotation            = annotations.EnumAnnotation
	ErrEnumFieldsNotAnnotated = annotations.ErrEnumFieldsNotAnnotated
	Enum                      = annotations.Enum
	OmitFieldPrefix           = annotations.OmitFieldPrefix
	NormalizeEnumIdentifier   = annotations.NormalizeEnumIdentifier

	FilterAnnotation   = annotations.FilterAnnotation
	Filter             = annotations.Filter
	FilterContains     = annotations.FilterContains
	WithFilterMode     = annotations.WithFilterMode
	FilterModeEQ       = annotations.FilterModeEQ
	FilterModeContains = annotations.FilterModeContains
	FilterModeIn       = annotations.FilterModeIn

	FieldAnnotation = annotations.FieldAnnotation
	Field           = annotations.Field
	Type            = annotations.Type
	TypeName        = annotations.TypeName

	SkipAnnotation = annotations.SkipAnnotation
	Skip           = annotations.Skip
)
