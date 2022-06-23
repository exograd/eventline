package web

type MultiSelect struct {
	Name            string
	Options         []MultiSelectOption
	SelectedOptions []string
	Size            int
}

func (ms *MultiSelect) IsSelected(name string) bool {
	for _, so := range ms.SelectedOptions {
		if so == name {
			return true
		}
	}

	return false
}

type MultiSelectOption struct {
	Name  string
	Label string
}
