package web

type Menu struct {
	Entries       []*MenuEntry
	SelectedEntry string
}

type MenuEntry struct {
	Id           string
	Icon         string
	Label        string
	URI          string
	External     bool
	New          bool
	WhenLoggedIn bool
	Apart        bool
}
