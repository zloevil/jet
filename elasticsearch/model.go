package elasticsearch

const (
	EsTypeKeyword         = "keyword"
	EsTypeText            = "text"
	EsTypeDate            = "date"
	EsTypeBool            = "boolean"
	EsTypeLong            = "long"
	EsTypeInteger         = "integer"
	EsTypeSearchAsYouType = "search_as_you_type"
	EsTypeFlattened       = "flattened"
	EsTypeNested          = "nested"
)

const (
	EsSortRequestMissingFirst = "_first"
	EsSortRequestMissingLast  = "_last"
)

var typesMap = map[string]struct{}{
	EsTypeKeyword:         {},
	EsTypeText:            {},
	EsTypeDate:            {},
	EsTypeBool:            {},
	EsTypeSearchAsYouType: {},
	EsTypeLong:            {},
	EsTypeInteger:         {},
	EsTypeFlattened:       {},
	EsTypeNested:          {},
}

type EsProperty struct {
	Type  string `json:"type,omitempty"`  // Type specifies a datatype
	Index *bool  `json:"index,omitempty"` // Index - if false, field isn't indexed
}

type EsProperties map[string]*EsProperty

type EsSettings struct {
	NumberOfShards   int `json:"number_of_shards"`
	NumberOfReplicas int `json:"number_of_replicas"`
}

type EsMapping struct {
	Settings EsSettings `json:"settings"`
	Mappings struct {
		Properties EsProperties `json:"properties"`
	} `json:"mappings"`
}
