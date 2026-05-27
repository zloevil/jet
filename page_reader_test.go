package jet

import (
	"context"
	"github.com/stretchr/testify/suite"
	"testing"
)

type pageReaderTestSuite struct {
	Suite
}

func (s *pageReaderTestSuite) SetupSuite() {
	s.Suite.Init(func() CLogger { return L(InitLogger(&LogConfig{Level: TraceLevel})) })
}

func (s *pageReaderTestSuite) SetupTest() {
}

func (s *pageReaderTestSuite) TearDownSuite() {}

func TestPageReaderSuite(t *testing.T) {
	suite.Run(t, new(pageReaderTestSuite))
}

type Criteria struct {
	Value string
}

type Item struct {
	Id string
}

func (s *pageReaderTestSuite) Test() {

	test := func(rs [][]*Item, expected int) {
		i := 0
		readFn := func(ctx context.Context, rq PagingRequestG[Criteria]) (PagingResponseG[Item], error) {
			for i < len(rs) {
				r := rs[i]
				i++
				return PagingResponseG[Item]{Items: r}, nil
			}
			return PagingResponseG[Item]{}, nil
		}
		var res []*Item
		pr := NewPageReader(readFn, 10, s.L())
		ch := pr.GetPage(s.Ctx, Criteria{})
		func() {
			for v := range ch {
				res = append(res, v.Items...)
			}
		}()
		s.Len(res, expected)
	}

	test(nil, 0)
	test([][]*Item{{}, {}}, 0)
	test([][]*Item{{{Id: "1"}}, {{Id: "2"}}}, 2)
	test([][]*Item{{{Id: "1"}, {Id: "1.1"}}, {{Id: "2"}, {Id: "2.2"}}}, 4)
	test([][]*Item{{{Id: "1"}, {Id: "1.1"}, {Id: "1.3"}}, {{Id: "2"}, {Id: "2.2"}}}, 5)
}
