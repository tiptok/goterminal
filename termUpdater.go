package main

var _DefaultTermUpdater *TermUpdater = NewTermUpdater()

type TermUpdater struct {
	List []*Terminal
}

func NewTermUpdater() *TermUpdater {
	return &TermUpdater{
		List: make([]*Terminal, 0),
	}
}

func (tu *TermUpdater) Add(t *Terminal) {
	tu.List = append(tu.List, t)
}
