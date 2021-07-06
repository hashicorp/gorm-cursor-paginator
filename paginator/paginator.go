package paginator

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/iancoleman/strcase"
	"github.com/jinzhu/gorm"

	"github.com/hashicorp/gorm-cursor-paginator/cursor"
	"github.com/hashicorp/gorm-cursor-paginator/internal/util"
)

// New creates paginator
func New(opts ...Option) *Paginator {
	p := &Paginator{}
	for _, opt := range append([]Option{&defaultConfig}, opts...) {
		opt.Apply(p)
	}
	return p
}

// Paginator a builder doing pagination
type Paginator struct {
	cursor Cursor
	rules  []Rule
	limit  int
	order  Order
}

// SetRules sets paging rules
func (p *Paginator) SetRules(rules ...Rule) {
	p.rules = make([]Rule, len(rules))
	copy(p.rules, rules)
}

// SetKeys sets paging keys
func (p *Paginator) SetKeys(keys ...string) {
	rules := make([]Rule, len(keys))
	for i, key := range keys {
		rules[i] = Rule{
			Key: key,
		}
	}
	p.SetRules(rules...)
}

// SetLimit sets paging limit
func (p *Paginator) SetLimit(limit int) {
	p.limit = limit
}

// SetOrder sets paging order
func (p *Paginator) SetOrder(order Order) {
	p.order = order
}

// SetAfterCursor sets paging after cursor
func (p *Paginator) SetAfterCursor(afterCursor string) {
	p.cursor.After = &afterCursor
}

// SetBeforeCursor sets paging before cursor
func (p *Paginator) SetBeforeCursor(beforeCursor string) {
	p.cursor.Before = &beforeCursor
}

// Paginate paginates data
func (p *Paginator) Paginate(db *gorm.DB, dest interface{}) (result *gorm.DB, c Cursor, err error) {
	if err = p.validate(dest); err != nil {
		return
	}
	p.setup(db, dest)
	fields, err := p.decodeCursor(dest)
	if err != nil {
		return
	}
	if result = p.appendPagingQuery(db, fields).Find(dest); result.Error != nil {
		return
	}
	// dest must be a pointer type or gorm will panic above
	elems := reflect.ValueOf(dest).Elem()
	// only encode next cursor when elems is not empty slice
	if elems.Kind() == reflect.Slice && elems.Len() > 0 {
		hasMore := elems.Len() > p.limit
		if hasMore {
			elems.Set(elems.Slice(0, elems.Len()-1))
		}
		if p.isBackward() {
			elems.Set(reverse(elems))
		}
		if c, err = p.encodeCursor(elems, hasMore); err != nil {
			return
		}
	}
	return
}

/* private */

func (p *Paginator) validate(dest interface{}) (err error) {
	if len(p.rules) == 0 {
		return ErrNoRule
	}
	if p.limit <= 0 {
		return ErrInvalidLimit
	}
	if err = p.order.validate(); err != nil {
		return
	}
	for _, rule := range p.rules {
		if err = rule.validate(dest); err != nil {
			return
		}
	}
	return
}

func (p *Paginator) setup(db *gorm.DB, dest interface{}) {
	var sqlTable string
	for i := range p.rules {
		multiKey := strings.Contains(p.rules[i].Key, ".")
		if p.rules[i].SQLRepr == "" {
			// if key has levels then recalculate table name regardless prev value
			// because it can be different for different keys in an aggregated model
			if multiKey {
				subkeys := strings.Split(p.rules[i].Key, ".")
				parentPath := strings.Join(subkeys[0:len(subkeys)-1], ".")
				if parent, ok := util.ReflectFieldByPath(dest, parentPath); ok {
					sqlTable = db.NewScope(reflect.New(parent.Type).Interface()).TableName()
				}
			} else if sqlTable == "" {
				sqlTable = db.NewScope(dest).TableName()
			}
			sqlKey := p.parseSQLKey(dest, p.rules[i].Key)
			p.rules[i].SQLRepr = fmt.Sprintf("%s.%s", sqlTable, sqlKey)
		}
		if p.rules[i].Order == "" {
			p.rules[i].Order = p.order
		}
	}
}

func (p *Paginator) parseSQLKey(dest interface{}, key string) string {
	// dest is already validated at validation phase
	f, _ := util.ReflectFieldByPath(dest, key)
	for _, tag := range strings.Split(string(f.Tag), " ") {
		// e.g., gorm:"type:varchar(255);column:field_name"
		if strings.HasPrefix(tag, "gorm:") {
			opts := strings.Split(
				strings.Trim(tag[len("gorm:"):], "\""),
				";",
			)
			for _, opt := range opts {
				if strings.HasPrefix(opt, "column:") {
					return opt[len("column:"):]
				}
			}
		}
	}
	return strcase.ToSnake(f.Name)
}

func (p *Paginator) decodeCursor(dest interface{}) (result []interface{}, err error) {
	if p.isForward() {
		if result, err = cursor.NewDecoder(p.getKeys()...).Decode(*p.cursor.After, dest); err != nil {
			err = ErrInvalidCursor
		}
	} else if p.isBackward() {
		if result, err = cursor.NewDecoder(p.getKeys()...).Decode(*p.cursor.Before, dest); err != nil {
			err = ErrInvalidCursor
		}
	}
	return
}

func (p *Paginator) isForward() bool {
	return p.cursor.After != nil
}

func (p *Paginator) isBackward() bool {
	// forward take precedence over backward
	return !p.isForward() && p.cursor.Before != nil
}

func (p *Paginator) appendPagingQuery(db *gorm.DB, fields []interface{}) *gorm.DB {
	stmt := db
	stmt = stmt.Limit(p.limit + 1)
	stmt = stmt.Order(p.buildOrderSQL())
	if len(fields) > 0 {
		stmt = stmt.Where(
			p.buildCursorSQLQuery(),
			p.buildCursorSQLQueryArgs(fields)...,
		)
	}
	return stmt
}

func (p *Paginator) buildOrderSQL() string {
	orders := make([]string, len(p.rules))
	for i, rule := range p.rules {
		order := rule.Order
		if p.isBackward() {
			order = order.flip()
		}
		orders[i] = fmt.Sprintf("%s %s", rule.SQLRepr, order)
	}
	return strings.Join(orders, ", ")
}

func (p *Paginator) buildCursorSQLQuery() string {
	queries := make([]string, len(p.rules))
	query := ""
	for i, rule := range p.rules {
		operator := "<"
		if (p.isForward() && rule.Order == ASC) ||
			(p.isBackward() && rule.Order == DESC) {
			operator = ">"
		}
		queries[i] = fmt.Sprintf("%s%s %s ?", query, rule.SQLRepr, operator)
		query = fmt.Sprintf("%s%s = ? AND ", query, rule.SQLRepr)
	}
	// for exmaple:
	// a > 1 OR a = 1 AND b > 2 OR a = 1 AND b = 2 AND c > 3
	return strings.Join(queries, " OR ")
}

func (p *Paginator) buildCursorSQLQueryArgs(fields []interface{}) (args []interface{}) {
	for i := 1; i <= len(fields); i++ {
		args = append(args, fields[:i]...)
	}
	return
}

func (p *Paginator) encodeCursor(elems reflect.Value, hasMore bool) (result Cursor, err error) {
	encoder := cursor.NewEncoder(p.getKeys()...)
	// encode after cursor
	if p.isBackward() || hasMore {
		c, err := encoder.Encode(elems.Index(elems.Len() - 1))
		if err != nil {
			return Cursor{}, err
		}
		result.After = &c
	}
	// encode before cursor
	if p.isForward() || (hasMore && p.isBackward()) {
		c, err := encoder.Encode(elems.Index(0))
		if err != nil {
			return Cursor{}, err
		}
		result.Before = &c
	}
	return
}

/* rules */

func (p *Paginator) getKeys() []string {
	keys := make([]string, len(p.rules))
	for i, rule := range p.rules {
		keys[i] = rule.Key
	}
	return keys
}
