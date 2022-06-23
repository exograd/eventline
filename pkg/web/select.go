package web

type Select struct {
	Id             string
	Name           string
	Options        []SelectOption
	SelectedOption string
}

func (s *Select) IsSelected(name string) bool {
	return name == s.SelectedOption
}

type SelectOption struct {
	Name  string
	Label string
}
