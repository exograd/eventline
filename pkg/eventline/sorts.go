package eventline

const DefaultSort = "id"

var DefaultSorts Sorts = Sorts{
	Sorts: map[string]string{
		"id": "id",
	},

	Default: "id",
}

type Sorts struct {
	Sorts   map[string]string
	Default string
}

func (ss Sorts) Contains(name string) bool {
	return ss.Sorts[name] != ""
}

func (ss Sorts) Column(name string) string {
	return ss.Sorts[name]
}
