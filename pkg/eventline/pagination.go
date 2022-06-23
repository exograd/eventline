package eventline

const (
	MinPageSize = 1
	MaxPageSize = 100
)

type PageElement interface {
	SortKey(string) string
}

type Page struct {
	Elements []PageElement `json:"elements"`
	Previous *Cursor       `json:"previous,omitempty"`
	Next     *Cursor       `json:"next,omitempty"`
}

func ReversePageElements(elts []PageElement) {
	for i := 0; i < len(elts)/2; i++ {
		j := len(elts) - i - 1
		elts[i], elts[j] = elts[j], elts[i]
	}
}

func NewPage(cursor *Cursor, elements []PageElement, sorts Sorts) *Page {
	if len(elements) == 0 {
		return &Page{
			Elements: elements,
		}
	}

	size := cursor.Size
	if size == 0 {
		size = DefaultCursorSize
	}

	order := cursor.Order
	if order == "" {
		order = OrderAsc
	}

	sort := cursor.Sort
	if sort == "" {
		sort = sorts.Default
	}

	var page Page

	firstElement := elements[0]

	if cursor.Before != "" {
		page.Next = &Cursor{
			After: firstElement.SortKey(sort),
			Size:  size,
			Order: order,
			Sort:  sort,
		}

		if len(elements) > size {
			elements = elements[:len(elements)-1]
			lastElement := elements[len(elements)-1]

			ReversePageElements(elements)
			page.Elements = elements

			page.Previous = &Cursor{
				Before: lastElement.SortKey(sort),
				Size:   size,
				Order:  order,
				Sort:   sort,
			}
		} else {
			ReversePageElements(elements)
			page.Elements = elements
		}
	} else {
		if cursor.After != "" {
			page.Previous = &Cursor{
				Before: firstElement.SortKey(sort),
				Size:   size,
				Order:  order,
				Sort:   sort,
			}
		}

		if len(elements) > size {
			elements = elements[:len(elements)-1]
			lastElement := elements[len(elements)-1]

			page.Elements = elements

			page.Next = &Cursor{
				After: lastElement.SortKey(sort),
				Size:  size,
				Order: order,
				Sort:  sort,
			}
		} else {
			page.Elements = elements
		}
	}

	return &page
}

func (p *Page) IsEmpty() bool {
	return len(p.Elements) == 0
}

func (p *Page) HasPreviousOrNextURI() bool {
	return p.Previous != nil || p.Next != nil
}

func (p *Page) PreviousURI() string {
	if p.Previous == nil {
		return ""
	}

	return p.Previous.URL().String()
}

func (p *Page) NextURI() string {
	if p.Next == nil {
		return ""
	}

	return p.Next.URL().String()
}
