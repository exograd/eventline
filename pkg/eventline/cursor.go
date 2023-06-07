package eventline

import (
	"encoding/base64"
	"fmt"
	"net/url"
	"strconv"

	"github.com/exograd/eventline/pkg/utils"
	"github.com/galdor/go-service/pkg/pg"
)

type Order string

const (
	OrderAsc  Order = "asc"
	OrderDesc Order = "desc"
)

const (
	DefaultCursorSize = 20
	MinCursorSize     = 1
	MaxCursorSize     = 100
)

type Cursor struct {
	Before string `json:"before,omitempty"`
	After  string `json:"after,omitempty"`
	Size   int    `json:"size,omitempty"`
	Sort   string `json:"sort,omitempty"`
	Order  Order  `json:"order,omitempty"`
}

func (pc *Cursor) ParseQuery(query url.Values, sorts Sorts, accountSettings *AccountSettings) error {
	var c Cursor

	// Before
	if s := query.Get("before"); s != "" {
		key, err := base64.StdEncoding.DecodeString(s)
		if err != nil {
			return fmt.Errorf("invalid query parameter %q: invalid value",
				"before")
		}

		c.Before = string(key)
	}

	// After
	if s := query.Get("after"); s != "" {
		if c.Before != "" {
			return fmt.Errorf("invalid query parameters: %q and %q are "+
				"both set", "before", "after")
		}

		key, err := base64.StdEncoding.DecodeString(s)
		if err != nil {
			return fmt.Errorf("invalid query parameter %q: invalid value",
				"after")
		}

		c.After = string(key)
	}

	// Size
	if s := query.Get("size"); s != "" {
		i64, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid query parameter %q: invalid format",
				"size")
		} else if i64 < MinCursorSize {
			return fmt.Errorf("invalid query parameter %q: value must be "+
				"greater or equal to %d", "size", MinCursorSize)
		} else if i64 > MaxCursorSize {
			return fmt.Errorf("invalid query parameter %q: value must be "+
				"lower or equal to %d", "size", MaxCursorSize)
		}

		c.Size = int(i64)
	} else if accountSettings != nil && accountSettings.PageSize > 0 {
		c.Size = accountSettings.PageSize
	}

	// Sort
	if s := query.Get("sort"); s != "" {
		if !sorts.Contains(s) {
			return fmt.Errorf("invalid query parameter %q: unsupported sort",
				"sort")
		}

		c.Sort = s
	}

	// Order
	if s := query.Get("order"); s != "" {
		switch Order(s) {
		case OrderAsc:
			c.Order = OrderAsc
		case OrderDesc:
			c.Order = OrderDesc

		default:
			return fmt.Errorf("invalid query parameter %q: invalid value",
				"order")
		}
	}

	*pc = c
	return nil
}

func (c *Cursor) Query() url.Values {
	query := make(url.Values)

	base64Encode := func(s string) string {
		return base64.StdEncoding.EncodeToString([]byte(s))
	}

	if c.Before != "" {
		query.Add("before", base64Encode(c.Before))
	}

	if c.After != "" {
		query.Add("after", base64Encode(c.After))
	}

	if c.Size > 0 {
		query.Add("size", strconv.Itoa(c.Size))
	}

	if c.Sort != "" {
		query.Add("sort", c.Sort)
	}

	if c.Order != "" {
		query.Add("order", string(c.Order))
	}

	return query
}

func (c *Cursor) URL() *url.URL {
	query := c.Query()

	return &url.URL{
		RawQuery: query.Encode(),
	}
}

func (c *Cursor) SQLConditionOrderLimit(sorts Sorts) string {
	return c.SQLConditionOrderLimit2(sorts, "")
}

func (c *Cursor) SQLConditionOrderLimit2(sorts Sorts, correlation string) string {
	size := c.Size
	if size == 0 {
		size = DefaultCursorSize
	}

	order := c.Order
	if order == "" {
		order = OrderAsc
	}

	sort := c.Sort
	if sort == "" {
		sort = sorts.Default
	}
	sortPart := sorts.Column(sort)
	if sortPart == "" {
		utils.Panicf("unknown sort %q", sort)
	}
	if correlation != "" {
		sortPart = correlation + "." + sortPart
	}

	var orderPart string
	if order == OrderAsc {
		if c.Before == "" {
			orderPart = "ASC"
		} else {
			orderPart = "DESC"
		}
	} else if order == OrderDesc {
		if c.Before == "" {
			orderPart = "DESC"
		} else {
			orderPart = "ASC"
		}
	} else {
		utils.Panicf("unsupported order %q", order)
	}

	var condPart string
	switch {
	case c.After != "" && order == OrderAsc:
		condPart = fmt.Sprintf(`%s > %s `, sortPart, pg.QuoteString(c.After))
	case c.After != "" && order == OrderDesc:
		condPart = fmt.Sprintf(`%s < %s `, sortPart, pg.QuoteString(c.After))
	case c.Before != "" && order == OrderAsc:
		condPart = fmt.Sprintf(`%s < %s `, sortPart, pg.QuoteString(c.Before))
	case c.Before != "" && order == OrderDesc:
		condPart = fmt.Sprintf(`%s > %s `, sortPart, pg.QuoteString(c.Before))
	default:
		condPart = `TRUE `
	}

	return fmt.Sprintf(`%sORDER BY (%s) %s LIMIT %d`,
		condPart, sortPart, orderPart, size+1)
}
