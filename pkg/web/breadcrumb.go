package web

type Breadcrumb struct {
	Entries []*BreadcrumbEntry
}

type BreadcrumbEntry struct {
	Label    string
	URI      string
	Verbatim bool
	Disabled bool
}

func NewBreadcrumb() *Breadcrumb {
	return &Breadcrumb{}
}

func (b *Breadcrumb) AddEntry(e *BreadcrumbEntry) {
	b.Entries = append(b.Entries, e)
}
