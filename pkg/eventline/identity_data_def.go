package eventline

type IdentityDataType string

const (
	IdentityDataTypeString     IdentityDataType = "string"
	IdentityDataTypeStringList IdentityDataType = "string_list"
	IdentityDataTypeEnum       IdentityDataType = "enum"
	IdentityDataTypeEnumList   IdentityDataType = "enum_list"
	IdentityDataTypeDate       IdentityDataType = "date"
	IdentityDataTypeURI        IdentityDataType = "uri"
	IdentityDataTypeTextBlock  IdentityDataType = "text_block"
	IdentityDataTypeBoolean    IdentityDataType = "boolean"
)

type IdentityDataDef struct {
	Entries []*IdentityDataEntry
}

type IdentityDataEntry struct {
	Key                   string
	Label                 string
	Value                 interface{}
	Type                  IdentityDataType
	EnumValues            []string
	PreselectedEnumValues []string
	MultiselectEnumSize   int // size of the multiselect html element
	Optional              bool
	Verbatim              bool
	Secret                bool
	Internal              bool // set by eventline, not by the user
}

func NewIdentityDataDef() *IdentityDataDef {
	return &IdentityDataDef{}
}

func (v *IdentityDataDef) AddEntry(e *IdentityDataEntry) {
	v.Entries = append(v.Entries, e)
}
