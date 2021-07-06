package paginator

import (
	"time"
)

func (s *paginatorSuite) TestPaginateDefaultOptions() {
	s.givenOrders(12)

	// Default Options
	// * Key: ID
	// * Limit: 10
	// * Order: DESC

	var p1 []TestOrder
	_, c, _ := New().Paginate(s.db, &p1)
	s.assertIDRange(p1, 12, 3)
	s.assertForwardOnly(c)

	var p2 []TestOrder
	_, c, _ = New(&Config{
		After: *c.After,
	}).Paginate(s.db, &p2)
	s.assertIDRange(p2, 2, 1)
	s.assertBackwardOnly(c)

	var p3 []TestOrder
	_, c, _ = New(&Config{
		Before: *c.Before,
	}).Paginate(s.db, &p3)
	s.assertIDRange(p3, 12, 3)
	s.assertForwardOnly(c)
}

/* data type */

func (s *paginatorSuite) TestPaginateSlicePtrs() {
	s.givenOrders(12)

	var p1 []*TestOrder
	_, c, _ := New().Paginate(s.db, &p1)
	s.assertIDRange(p1, 12, 3)
	s.assertForwardOnly(c)

	var p2 []*TestOrder
	_, c, _ = New(&Config{
		After: *c.After,
	}).Paginate(s.db, &p2)
	s.assertIDRange(p2, 2, 1)
	s.assertBackwardOnly(c)

	var p3 []*TestOrder
	_, c, _ = New(&Config{
		Before: *c.Before,
	}).Paginate(s.db, &p3)
	s.assertIDRange(p3, 12, 3)
	s.assertForwardOnly(c)
}

/*
TODO unclear why this is breaking. The generated query seems right:
SELECT * FROM "orders"   ORDER BY orders.id DESC LIMIT 11

Maybe a change in behavior for how Find works between v1/v2.
func (s *paginatorSuite) TestPaginateNonSlice() {
	s.givenOrders(3)

	var o TestOrder
	_, c, _ := New().Paginate(s.db, &o)
	s.Equal(3, o.ID)
	s.assertNoMore(c)
}
*/

func (s *paginatorSuite) TestPaginateNoMore() {
	s.givenOrders(3)

	var orders []TestOrder
	_, c, _ := New().Paginate(s.db, &orders)
	s.assertIDRange(orders, 3, 1)
	s.assertNoMore(c)
}

func (s *paginatorSuite) TestPaginateSpecialCharacter() {
	// ordered by Remark desc -> 2, 1, 4, 3 (":" > "," > "&" > "%")
	s.givenOrders([]TestOrder{
		{ID: 1, Remark: ptrStr("a,b,c")},
		{ID: 2, Remark: ptrStr("a:b:c")},
		{ID: 3, Remark: ptrStr("a%b%c")},
		{ID: 4, Remark: ptrStr("a&b&c")},
	})

	cfg := Config{
		Keys:  []string{"Remark"},
		Limit: 3,
	}

	var p1 []TestOrder
	_, c, _ := New(&cfg).Paginate(s.db, &p1)
	s.assertIDs(p1, 2, 1, 4)
	s.assertForwardOnly(c)

	var p2 []TestOrder
	_, c, _ = New(
		&cfg,
		WithAfter(*c.After),
	).Paginate(s.db, &p2)
	s.assertIDs(p2, 3)
	s.assertBackwardOnly(c)

	var p3 []TestOrder
	_, c, _ = New(
		&cfg,
		WithBefore(*c.Before),
	).Paginate(s.db, &p3)
	s.assertIDs(p3, 2, 1, 4)
	s.assertForwardOnly(c)
}

/* cursor */

func (s *paginatorSuite) TestPaginateForwardShouldTakePrecedenceOverBackward() {
	s.givenOrders(30)

	var p1 []TestOrder
	_, c, _ := New().Paginate(s.db, &p1)
	s.assertIDRange(p1, 30, 21)
	s.assertForwardOnly(c)

	var p2 []TestOrder
	_, c, _ = New(&Config{
		After: *c.After,
	}).Paginate(s.db, &p2)
	s.assertIDRange(p2, 20, 11)
	s.assertBothDirections(c)

	var p3 []TestOrder
	_, c, _ = New(&Config{
		After:  *c.After,
		Before: *c.Before,
	}).Paginate(s.db, &p3)
	s.assertIDRange(p3, 10, 1)
	s.assertBackwardOnly(c)
}

/* key */

func (s *paginatorSuite) TestPaginateSingleKey() {
	now := time.Now()
	// ordered by CreatedAt desc -> 1, 3, 2
	s.givenOrders([]TestOrder{
		{ID: 1, CreatedAt: now.Add(1 * time.Hour)},
		{ID: 2, CreatedAt: now.Add(-1 * time.Hour)},
		{ID: 3, CreatedAt: now},
	})

	cfg := Config{
		Keys:  []string{"CreatedAt"},
		Limit: 2,
	}

	var p1 []TestOrder
	_, c, _ := New(&cfg).Paginate(s.db, &p1)
	s.assertIDs(p1, 1, 3)
	s.assertForwardOnly(c)

	var p2 []TestOrder
	_, c, _ = New(
		&cfg,
		WithAfter(*c.After),
	).Paginate(s.db, &p2)
	s.assertIDs(p2, 2)
	s.assertBackwardOnly(c)

	var p3 []TestOrder
	_, c, _ = New(
		&cfg,
		WithBefore(*c.Before),
	).Paginate(s.db, &p3)
	s.assertIDs(p3, 1, 3)
	s.assertForwardOnly(c)
}

// OrderAndItems is an aggregated model that includes
// two simple models
type OrderAndItems struct {
	TestOrder
	TestItem
}

// TableName provides the "primary" table for querying
// an aggregated model OrderAndItems, table for the rest
// of data will be joined in the query itself
func (o OrderAndItems) TableName() string {
	return "orders"
}

func (s *paginatorSuite) TestPaginateAggregatedModel() {
	now := time.Now()
	// ordered by CreatedAt desc -> 1, 3, 2
	order1 := TestOrder{ID: 1, CreatedAt: now.Add(1 * time.Hour)}
	s.givenOrders([]TestOrder{
		order1,
		{ID: 2, CreatedAt: now.Add(-1 * time.Hour)},
		{ID: 3, CreatedAt: now},
	})

	s.givenItems(order1, 2)

	cfg := Config{
		Keys:  []string{"TestOrder.CreatedAt", "TestOrder.ID", "TestItem.ID"},
		Limit: 2,
	}

	var p1 []OrderAndItems
	db := s.db.Model(&OrderAndItems{}).
		Select("orders.*, items.*").
		Joins("left join items on items.order_id = orders.id")

	_, c, err := New(&cfg).Paginate(db, &p1)
	s.NoError(err)
	s.Equal(2, len(p1))
	s.Equal(1, p1[0].TestOrder.ID)
	s.Equal(2, p1[0].TestItem.ID)
	s.Equal(1, p1[1].TestOrder.ID)
	s.Equal(1, p1[1].TestItem.ID)
	s.assertForwardOnly(c)

	var p2 []OrderAndItems
	_, c, err = New(
		&cfg,
		WithAfter(*c.After),
	).Paginate(db, &p2)
	s.NoError(err)
	s.Equal(2, len(p1))
	s.Equal(3, p2[0].TestOrder.ID)
	s.Equal(0, p2[0].TestItem.ID)
	s.Equal(2, p2[1].TestOrder.ID)
	s.Equal(0, p2[1].TestItem.ID)
	s.assertBackwardOnly(c)

	var p3 []OrderAndItems
	_, c, _ = New(
		&cfg,
		WithBefore(*c.Before),
	).Paginate(db, &p3)
	s.Equal(1, p3[0].TestOrder.ID)
	s.Equal(2, p3[0].TestItem.ID)
	s.Equal(1, p3[1].TestOrder.ID)
	s.Equal(1, p3[1].TestItem.ID)
	s.assertForwardOnly(c)
}

func (s *paginatorSuite) TestPaginateMultipleKeys() {
	now := time.Now()
	// ordered by (CreatedAt desc, ID desc) -> 2, 3, 1
	s.givenOrders([]TestOrder{
		{ID: 1, CreatedAt: now},
		{ID: 2, CreatedAt: now.Add(1 * time.Hour)},
		{ID: 3, CreatedAt: now},
	})

	cfg := Config{
		Keys:  []string{"CreatedAt", "ID"},
		Limit: 2,
	}

	var p1 []TestOrder
	_, c, _ := New(&cfg).Paginate(s.db, &p1)
	s.assertIDs(p1, 2, 3)
	s.assertForwardOnly(c)

	var p2 []TestOrder
	_, c, _ = New(
		&cfg,
		WithAfter(*c.After),
	).Paginate(s.db, &p2)
	s.assertIDs(p2, 1)
	s.assertBackwardOnly(c)

	var p3 []TestOrder
	_, c, _ = New(
		&cfg,
		WithBefore(*c.Before),
	).Paginate(s.db, &p3)
	s.assertIDs(p3, 2, 3)
	s.assertForwardOnly(c)
}

func (s *paginatorSuite) TestPaginatePointerKey() {
	s.givenOrders([]TestOrder{
		{ID: 1, Remark: ptrStr("3")},
		{ID: 2, Remark: ptrStr("2")},
		{ID: 3, Remark: ptrStr("1")},
	})

	cfg := Config{
		Keys:  []string{"Remark", "ID"},
		Limit: 2,
	}

	var p1 []TestOrder
	_, c, _ := New(&cfg).Paginate(s.db, &p1)
	s.assertIDs(p1, 1, 2)
	s.assertForwardOnly(c)

	var p2 []TestOrder
	_, c, _ = New(
		&cfg,
		WithAfter(*c.After),
	).Paginate(s.db, &p2)
	s.assertIDs(p2, 3)
	s.assertBackwardOnly(c)

	var p3 []TestOrder
	_, c, _ = New(
		&cfg,
		WithBefore(*c.Before),
	).Paginate(s.db, &p3)
	s.assertIDs(p3, 1, 2)
	s.assertForwardOnly(c)
}

func (s *paginatorSuite) TestPaginateRulesShouldTakePrecedenceOverKeys() {
	now := time.Now()
	// ordered by ID desc -> 2, 1
	// ordered by CreatedAt desc -> 1, 2
	s.givenOrders([]TestOrder{
		{ID: 1, CreatedAt: now.Add(1 * time.Hour)},
		{ID: 2, CreatedAt: now},
	})

	cfg := Config{
		Rules: []Rule{
			{Key: "CreatedAt"},
		},
		Keys: []string{"ID"},
	}

	var orders []TestOrder
	_, _, _ = New(&cfg).Paginate(s.db, &orders)
	s.assertIDs(orders, 1, 2)
}

func (s *paginatorSuite) TestPaginateShouldUseGormColumnTag() {
	s.givenOrders(3)

	type order struct {
		ID        int
		OrderedAt time.Time `json:"orderedAt" gorm:"type:timestamp;column:created_at"`
	}

	var orders []order
	result, _, _ := New(WithKeys("OrderedAt")).Paginate(s.db, &orders)
	s.Nil(result.Error)
	s.assertIDs(orders, 3, 2, 1)
}

/* limit */

func (s *paginatorSuite) TestPaginateLimit() {
	s.givenOrders(10)

	var p1 []TestOrder
	_, c, _ := New(&Config{
		Limit: 1,
	}).Paginate(s.db, &p1)
	s.Len(p1, 1)
	s.assertForwardOnly(c)

	var p2 []TestOrder
	_, c, _ = New(&Config{
		Limit: 20,
		After: *c.After,
	}).Paginate(s.db, &p2)
	s.Len(p2, 9)
	s.assertBackwardOnly(c)

	var p3 []TestOrder
	_, c, _ = New(&Config{
		Limit:  100,
		Before: *c.Before,
	}).Paginate(s.db, &p3)
	s.Len(p3, 1)
	s.assertForwardOnly(c)
}

/* TestOrder */

func (s *paginatorSuite) TestPaginateOrder() {
	now := time.Now()
	// ordered by (CreatedAt desc, ID desc) -> 4, 2, 3, 1
	s.givenOrders([]TestOrder{
		{ID: 1, CreatedAt: now},
		{ID: 2, CreatedAt: now.Add(1 * time.Hour)},
		{ID: 3, CreatedAt: now},
		{ID: 4, CreatedAt: now.Add(2 * time.Hour)},
	})

	cfg := Config{
		Keys:  []string{"CreatedAt", "ID"},
		Limit: 2,
	}

	var p1 []TestOrder
	_, c, _ := New(
		&cfg,
		WithOrder(ASC),
	).Paginate(s.db, &p1)
	s.assertIDs(p1, 1, 3)
	s.assertForwardOnly(c)

	var p2 []TestOrder
	_, c, _ = New(
		&cfg,
		WithBefore(*c.After),
		WithOrder(DESC),
	).Paginate(s.db, &p2)
	s.assertIDs(p2, 4, 2)
	s.assertForwardOnly(c)

	var p3 []TestOrder
	_, c, _ = New(
		&cfg,
		WithBefore(*c.After),
		WithOrder(ASC),
	).Paginate(s.db, &p3)
	s.assertIDs(p3, 1, 3)
	s.assertForwardOnly(c)
}

func (s *paginatorSuite) TestPaginateOrderByKey() {
	now := time.Now()
	// ordered by (CreatedAt desc, ID asc) -> 4, 2, 1, 3
	s.givenOrders([]TestOrder{
		{ID: 1, CreatedAt: now},
		{ID: 2, CreatedAt: now.Add(1 * time.Hour)},
		{ID: 3, CreatedAt: now},
		{ID: 4, CreatedAt: now.Add(2 * time.Hour)},
	})

	cfg := Config{
		Rules: []Rule{
			{
				Key: "CreatedAt",
			},
			{
				Key:   "ID",
				Order: ASC,
			},
		},
		Limit: 2,
		Order: DESC, // default order for no order rule
	}

	var p1 []TestOrder
	_, c, _ := New(&cfg).Paginate(s.db, &p1)
	s.assertIDs(p1, 4, 2)
	s.assertForwardOnly(c)

	var p2 []TestOrder
	_, c, _ = New(
		&cfg,
		WithAfter(*c.After),
	).Paginate(s.db, &p2)
	s.assertIDs(p2, 1, 3)
	s.assertBackwardOnly(c)

	var p3 []TestOrder
	_, c, _ = New(
		&cfg,
		WithBefore(*c.Before),
	).Paginate(s.db, &p3)
	s.assertIDs(p3, 4, 2)
	s.assertForwardOnly(c)
}

/* join */

func (s *paginatorSuite) TestPaginateJoinQuery() {
	orders := s.givenOrders(3)
	// total 5 items
	// order 1 -> items (1, 2, 3)
	// order 2 -> items (4, 5)
	// order 3 -> items (6)
	s.givenItems(orders[0], 2)
	s.givenItems(orders[1], 2)
	s.givenItems(orders[2], 1)

	stmt := s.db.
		Table("items").
		Joins("JOIN orders ON orders.id = items.order_id")

	cfg := Config{
		Limit: 3,
	}

	var p1 []TestItem
	_, c, _ := New(&cfg).Paginate(stmt, &p1)
	s.assertIDRange(p1, 5, 3)
	s.assertForwardOnly(c)

	var p2 []TestItem
	_, c, _ = New(
		&cfg,
		WithAfter(*c.After),
	).Paginate(stmt, &p2)
	s.assertIDRange(p2, 2, 1)
	s.assertBackwardOnly(c)

	var p3 []TestItem
	_, c, _ = New(
		&cfg,
		WithBefore(*c.Before),
	).Paginate(stmt, &p3)
	s.assertIDRange(p3, 5, 3)
	s.assertForwardOnly(c)
}

/* compatibility */

func (s *paginatorSuite) TestPaginateJoinQueryWithAlias() {
	orders := s.givenOrders(2)
	// total 6 items
	// order 1 -> items (1, 3, 5)
	// order 2 -> items (2, 4, 6)
	for i := 0; i < 3; i++ {
		s.givenItems(orders[0], 1)
		s.givenItems(orders[1], 1)
	}

	type itemDTO struct {
		ID      int
		OrderID int
	}

	stmt := s.db.
		Select("its.id AS id, ods.id AS order_id").
		Table("items AS its").
		Joins("JOIN orders AS ods ON ods.id = its.order_id")

	cfg := Config{
		Rules: []Rule{
			{
				Key:     "OrderID",
				SQLRepr: "ods.id",
			},
			{
				Key:     "ID",
				SQLRepr: "its.id",
			},
		},
		Limit: 3,
	}

	var p1 []itemDTO
	_, c, _ := New(&cfg).Paginate(stmt, &p1)
	s.assertIDs(p1, 6, 4, 2)
	s.assertForwardOnly(c)

	var p2 []itemDTO
	_, c, _ = New(
		&cfg,
		WithAfter(*c.After),
	).Paginate(stmt, &p2)
	s.assertIDs(p2, 5, 3, 1)
	s.assertBackwardOnly(c)

	var p3 []itemDTO
	_, c, _ = New(
		&cfg,
		WithBefore(*c.Before),
	).Paginate(stmt, &p3)
	s.assertIDs(p3, 6, 4, 2)
	s.assertForwardOnly(c)
}

func (s *paginatorSuite) TestPaginateConsistencyBetweenBuilderAndKeyOptions() {
	now := time.Now()
	s.givenOrders([]TestOrder{
		{ID: 1, CreatedAt: now},
		{ID: 2, CreatedAt: now},
		{ID: 3, CreatedAt: now},
		{ID: 4, CreatedAt: now},
		{ID: 5, CreatedAt: now},
	})

	var temp []TestOrder
	result, c, err := New(
		WithKeys("CreatedAt", "ID"),
		WithLimit(3),
	).Paginate(s.db, &temp)
	if err != nil {
		s.FailNow(err.Error())
	}
	if result.Error != nil {
		s.FailNow(result.Error.Error())
	}

	anchorCursor := *c.After

	var optOrders, builderOrders []TestOrder
	var optCursor, builderCursor Cursor

	// forward - keys

	opts := []Option{
		WithKeys("CreatedAt", "ID"),
		WithLimit(3),
		WithOrder(ASC),
		WithAfter(anchorCursor),
	}
	_, optCursor, _ = New(opts...).Paginate(s.db, &optOrders)
	s.assertIDs(optOrders, 4, 5)
	s.assertBackwardOnly(optCursor)

	p := New()
	p.SetKeys("CreatedAt", "ID")
	p.SetLimit(3)
	p.SetOrder(ASC)
	p.SetAfterCursor(anchorCursor)
	_, builderCursor, _ = p.Paginate(s.db, &builderOrders)
	s.assertIDs(builderOrders, 4, 5)
	s.assertBackwardOnly(builderCursor)

	s.Equal(optOrders, builderOrders)
	s.Equal(optCursor, builderCursor)

	// backward - keys

	opts = []Option{
		WithKeys("CreatedAt", "ID"),
		WithLimit(3),
		WithOrder(ASC),
		WithBefore(anchorCursor),
	}
	_, optCursor, _ = New(opts...).Paginate(s.db, &optOrders)
	s.assertIDs(optOrders, 1, 2)
	s.assertForwardOnly(optCursor)

	p = New()
	p.SetKeys("CreatedAt", "ID")
	p.SetLimit(3)
	p.SetOrder(ASC)
	p.SetBeforeCursor(anchorCursor)
	_, builderCursor, _ = p.Paginate(s.db, &builderOrders)
	s.assertIDs(builderOrders, 1, 2)
	s.assertForwardOnly(builderCursor)

	s.Equal(optOrders, builderOrders)
	s.Equal(optCursor, builderCursor)
}

func (s *paginatorSuite) TestPaginateConsistencyBetweenBuilderAndRuleOptions() {
	now := time.Now()
	s.givenOrders([]TestOrder{
		{ID: 1, CreatedAt: now},
		{ID: 2, CreatedAt: now},
		{ID: 3, CreatedAt: now},
		{ID: 4, CreatedAt: now},
		{ID: 5, CreatedAt: now},
	})

	var temp []TestOrder
	result, c, err := New(
		WithKeys("CreatedAt", "ID"),
		WithLimit(3),
	).Paginate(s.db, &temp)
	if err != nil {
		s.FailNow(err.Error())
	}
	if result.Error != nil {
		s.FailNow(result.Error.Error())
	}

	anchorCursor := *c.After

	var optOrders, builderOrders []TestOrder
	var optCursor, builderCursor Cursor

	// forward - rules

	opts := []Option{
		WithRules([]Rule{
			{Key: "CreatedAt"},
			{Key: "ID"},
		}...),
		WithLimit(3),
		WithOrder(ASC),
		WithAfter(anchorCursor),
	}
	_, optCursor, _ = New(opts...).Paginate(s.db, &optOrders)
	s.assertIDs(optOrders, 4, 5)
	s.assertBackwardOnly(optCursor)

	p := New()
	p.SetRules([]Rule{
		{Key: "CreatedAt"},
		{Key: "ID"},
	}...)
	p.SetLimit(3)
	p.SetOrder(ASC)
	p.SetAfterCursor(anchorCursor)
	_, builderCursor, err = p.Paginate(s.db, &builderOrders)
	s.assertIDs(builderOrders, 4, 5)
	s.assertBackwardOnly(builderCursor)

	s.Equal(optOrders, builderOrders)
	s.Equal(optCursor, builderCursor)

	// backward - rules

	opts = []Option{
		WithRules([]Rule{
			{Key: "CreatedAt"},
			{Key: "ID"},
		}...),
		WithLimit(3),
		WithOrder(ASC),
		WithBefore(anchorCursor),
	}
	_, optCursor, _ = New(opts...).Paginate(s.db, &optOrders)
	s.assertIDs(optOrders, 1, 2)
	s.assertForwardOnly(optCursor)

	p = New()
	p.SetRules([]Rule{
		{Key: "CreatedAt"},
		{Key: "ID"},
	}...)
	p.SetLimit(3)
	p.SetOrder(ASC)
	p.SetBeforeCursor(anchorCursor)
	_, builderCursor, _ = p.Paginate(s.db, &builderOrders)
	s.assertIDs(builderOrders, 1, 2)
	s.assertForwardOnly(builderCursor)

	s.Equal(optOrders, builderOrders)
	s.Equal(optCursor, builderCursor)
}
