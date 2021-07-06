package paginator

import (
	"fmt"
	"math"
	"reflect"
	"testing"
	"time"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	"github.com/stretchr/testify/suite"
)

func TestPaginator(t *testing.T) {
	suite.Run(t, &paginatorSuite{})
}

/* models */

type TestOrder struct {
	ID        int       `gorm:"primary_key"`
	Remark    *string   `gorm:"type:varchar(30)"`
	CreatedAt time.Time `gorm:"type:timestamp;not null"`
}

func (o TestOrder) TableName() string {
	return "orders"
}

type TestItem struct {
	ID      int     `gorm:"primary_key"`
	Name    string  `gorm:"type:varchar(30);not null"`
	Remark  *string `gorm:"type:varchar(30)"`
	OrderID int     `gorm:"type:integer;not null"`
	Order   Order   `gorm:"foreignkey:OrderID"`
}

func (i TestItem) TableName() string {
	return "items"
}

/* paginator suite */

type paginatorSuite struct {
	suite.Suite
	db *gorm.DB
}

/* setup */

func (s *paginatorSuite) SetupSuite() {
	db, err := gorm.Open("postgres", "host=localhost port=8765 dbname=test user=test password=test sslmode=disable")
	if err != nil {
		s.FailNow(err.Error())
	}
	s.db = db
	s.db.AutoMigrate(&TestOrder{}, &TestItem{})
	s.db.Model(&TestItem{}).AddForeignKey("order_id", "orders(id)", "CASCADE", "CASCADE")
}

/* teardown */

func (s *paginatorSuite) TearDownTest() {
	s.db.Exec("TRUNCATE orders, items RESTART IDENTITY;")
}

func (s *paginatorSuite) TearDownSuite() {
	s.db.DropTable(&TestItem{}, &TestOrder{})
	s.db.Close()
}

/* fixtures */

func (s *paginatorSuite) givenOrders(numOrOrders interface{}) (orders []TestOrder) {
	switch v := numOrOrders.(type) {
	case int:
		for i := 0; i < v; i++ {
			orders = append(orders, TestOrder{
				CreatedAt: time.Now().Add(time.Duration(i) * time.Hour),
			})
		}
	case []TestOrder:
		orders = v
	default:
		panic("givenOrders: numOrOrders should be number or orders")
	}
	for i := 0; i < len(orders); i++ {
		if err := s.db.Create(&orders[i]).Error; err != nil {
			panic(err.Error())
		}
	}
	return
}

func (s *paginatorSuite) givenItems(order TestOrder, numOrItems interface{}) (items []TestItem) {
	switch v := numOrItems.(type) {
	case int:
		for i := 0; i < v; i++ {
			items = append(items, TestItem{
				Name:    fmt.Sprintf("item %d", i+1),
				OrderID: order.ID,
			})
		}
	case []TestItem:
		items = v
	default:
		panic("givenItems: numOrItems should be number or items")
	}
	for i := 0; i < len(items); i++ {
		if err := s.db.Create(&items[i]).Error; err != nil {
			panic(err.Error())
		}
	}
	return
}

/* assertions */

func (s *paginatorSuite) assertIDRange(result interface{}, fromID, toID int) {
	rv := reflect.ValueOf(result)
	if rv.Kind() != reflect.Slice {
		panic("assertIDRange: result should be a slice")
	}
	s.Equal(
		int(math.Abs(float64(fromID-toID))+1),
		rv.Len(),
	)
	cur, vector := fromID, 1
	if fromID > toID {
		vector = -1
	}
	for i := 0; i < rv.Len(); i++ {
		e := rv.Index(i)
		if e.Kind() == reflect.Ptr {
			e = e.Elem()
		}
		s.Equal(cur, e.FieldByName("ID").Interface())
		cur += vector
	}
}

func (s *paginatorSuite) assertIDs(result interface{}, ids ...int) {
	rv := reflect.ValueOf(result)
	if rv.Kind() != reflect.Slice {
		panic("assertIDs: result should be a slice")
	}
	s.Equal(len(ids), rv.Len())

	for i := 0; i < rv.Len(); i++ {
		e := rv.Index(i)
		if e.Kind() == reflect.Ptr {
			e = e.Elem()
		}
		s.Equal(ids[i], e.FieldByName("ID").Interface())
	}
}

func (s *paginatorSuite) assertForwardOnly(c Cursor) {
	s.NotNil(c.After)
	s.Nil(c.Before)
}

func (s *paginatorSuite) assertBackwardOnly(c Cursor) {
	s.Nil(c.After)
	s.NotNil(c.Before)
}

func (s *paginatorSuite) assertBothDirections(c Cursor) {
	s.NotNil(c.After)
	s.NotNil(c.Before)
}

func (s *paginatorSuite) assertNoMore(c Cursor) {
	s.Nil(c.After)
	s.Nil(c.Before)
}

/* util */

func ptrStr(v string) *string {
	return &v
}
