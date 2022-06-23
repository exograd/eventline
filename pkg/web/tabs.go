package web

type Tabs struct {
	Tabs        []*Tab
	SelectedTab string
}

type Tab struct {
	Id    string
	Icon  string
	Label string
	URI   string
}

func NewTabs() *Tabs {
	return &Tabs{}
}

func (ts *Tabs) AddTab(t *Tab) {
	ts.Tabs = append(ts.Tabs, t)
}
