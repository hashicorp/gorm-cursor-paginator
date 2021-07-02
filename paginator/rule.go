package paginator

import "github.com/hashicorp/gorm-cursor-paginator/internal/util"

// Rule for paginator
type Rule struct {
	Key     string
	Order   Order
	SQLRepr string
}

func (r *Rule) validate(dest interface{}) (err error) {
	if _, ok := util.ReflectFieldByPath(dest, r.Key); !ok {
		return ErrInvalidModel
	}
	if r.Order != "" {
		if err = r.Order.validate(); err != nil {
			return
		}
	}
	return nil
}
