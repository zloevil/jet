package http

import (
	"context"
	"fmt"
	"github.com/zloevil/jet"
	"regexp"
	"strings"
)

const (
	asc   = "asc"
	desc  = "desc"
	first = "first"
	last  = "last"
)

var sortDirections = map[string]bool{
	"":   false,
	asc:  false,
	desc: true,
}

var sortNullsLast = map[string]bool{
	"":    false,
	first: false,
	last:  true,
}

// ParseSortBy Converts string like "field1 asc first,field2 desc last,field3 desc,field4" to array of SortRequest
func ParseSortBy(ctx context.Context, sortString string) ([]*jet.SortRequest, error) {
	if sortString == "" {
		return nil, nil
	}

	elements := strings.Split(sortString, ",")

	var res []*jet.SortRequest
	ruleRegex := regexp.MustCompile(fmt.Sprintf("^([a-zA-Z][a-zA-Z0-9]+)(?: (%s|%s)(?: (%s|%s))?)?$", asc, desc, first, last))
	for _, elem := range elements {

		elemParts := ruleRegex.FindStringSubmatch(elem)
		if elemParts == nil {
			return nil, ErrHttpUrlWrongSortFormat(ctx, sortString)
		}

		sortRq := &jet.SortRequest{Field: elemParts[1]}

		sortDesc, ok := sortDirections[elemParts[2]]
		if !ok {
			return nil, ErrHttpUrlWrongSortFormat(ctx, sortString)
		}
		sortRq.Desc = sortDesc

		nullsLast, ok := sortNullsLast[elemParts[3]]
		if !ok {
			return nil, ErrHttpUrlWrongSortFormat(ctx, sortString)
		}
		sortRq.NullsLast = nullsLast

		res = append(res, sortRq)
	}
	return res, nil
}
