package pg

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"github.com/lib/pq"
	"github.com/zloevil/jet"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"time"
)

// GormDto specifies base attrs for GORM dto
type GormDto struct {
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *gorm.DeletedAt
}

type TotalCount struct {
	TotalCount int `gorm:"column:total"`
}

// StringToNull transforms empty string to nil string, so that gorm stores it as NULL
func StringToNull(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// NullToString transforms NULL to empty string
func NullToString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// JSONB is a PostgreSQL jsonb value backed by raw JSON bytes. It implements
// driver.Valuer and sql.Scanner so GORM stores and loads it as a jsonb column.
type JSONB []byte

// GormDataType tells GORM to use the jsonb column type.
func (JSONB) GormDataType() string {
	return "jsonb"
}

// Value implements driver.Valuer.
func (j JSONB) Value() (driver.Value, error) {
	if len(j) == 0 {
		return "{}", nil
	}
	return string(j), nil
}

// Scan implements sql.Scanner.
func (j *JSONB) Scan(src interface{}) error {
	switch v := src.(type) {
	case nil:
		*j = nil
	case []byte:
		*j = append((*j)[:0:0], v...)
	case string:
		*j = []byte(v)
	default:
		return ErrPgGetJsonb(fmt.Errorf("unsupported scan type %T", src))
	}
	return nil
}

// GetEmptyJson returns an empty jsonb value ("{}").
func GetEmptyJson() (*JSONB, error) {
	j := JSONB("{}")
	return &j, nil
}

// MapToJsonb converts a map to jsonb.
func MapToJsonb[T comparable, K any](payload map[T]K) (*JSONB, error) {
	if payload == nil {
		return GetEmptyJson()
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return nil, ErrPgSetJsonb(err)
	}
	j := JSONB(b)
	return &j, nil
}

// ToJsonb converts an arbitrary object to jsonb.
func ToJsonb[T any](payload *T) (*JSONB, error) {
	if payload == nil {
		return GetEmptyJson()
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return nil, ErrPgSetJsonb(err)
	}
	j := JSONB(b)
	return &j, nil
}

// FromJsonb decodes a jsonb value into T.
func FromJsonb[T any](j *JSONB) (*T, error) {
	if j == nil {
		return nil, nil
	}
	var v T
	if err := json.Unmarshal(*j, &v); err != nil {
		return nil, ErrPgGetJsonb(err)
	}
	return &v, nil
}

const (
	PageSizeMaxLimit = 100
	PageSizeDefault  = 20
)

func PagingLimit(rqLimit int) int {
	if rqLimit <= 0 {
		return PageSizeDefault
	}
	if rqLimit > PageSizeMaxLimit {
		return PageSizeMaxLimit
	}
	return rqLimit
}

func Paging(rq jet.PagingRequest) func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		// apply sort
		if len(rq.SortBy) == 0 {
			rq.SortBy = []*jet.SortRequest{{
				Field: "updated_at",
				Desc:  true,
			}}
		}
		for _, srt := range rq.SortBy {
			db = db.Order(clause.OrderByColumn{Column: clause.Column{Name: srt.Field}, Desc: srt.Desc})
		}

		// apply paging
		if rq.Index < 0 {
			rq.Index = 0
		}
		offset := (rq.Index - 1) * rq.Size
		if offset < 0 {
			offset = 0
		}
		return db.Limit(PagingLimit(rq.Size)).Offset(offset)
	}
}

func OrderByUpdatedAt(desc bool) func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Order(clause.OrderByColumn{Column: clause.Column{Name: "updated_at"}, Desc: desc})
	}
}

func OrderByCreatedAt(desc bool) func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Order(clause.OrderByColumn{Column: clause.Column{Name: "created_at"}, Desc: desc})
	}
}

func Merge() func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Clauses(clause.OnConflict{UpdateAll: true})
	}
}

func Update() func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Omit("created_at")
	}
}

func Single() func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Limit(1)
	}
}

func WhereStrings(field string, values []string) func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if field != "" && len(values) > 0 {
			return db.Where("? && "+field, pq.Array(values))
		}
		return db
	}
}
