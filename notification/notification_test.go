package notification

import (
	"github.com/stretchr/testify/suite"
	"github.com/zloevil/jet"
	"testing"
)

type testNotificationSuite struct {
	jet.Suite
}

func (s *testNotificationSuite) SetupSuite() {
	s.Suite.Init(func() jet.CLogger { return jet.L(jet.InitLogger(&jet.LogConfig{Level: jet.TraceLevel})) })
}

func TestNotificationSuite(t *testing.T) {
	suite.Run(t, new(testNotificationSuite))
}

func (s *testNotificationSuite) Test_OnlyMessage() {
	n := New[string]().B(jet.StringPtr("data"))
	s.Equal("data", *n.Msg)
}

func (s *testNotificationSuite) Test_EmptyPolicyResolver() {
	rec := func(resolver Resolver) ([]*Receiver, error) {
		perms := resolver.Resolve()
		s.Empty(perms.Items)

		return []*Receiver{{Receiver: "receiver", Type: "type"}}, nil
	}
	r, err := NewReceivers().Scopes(rec).B()
	s.NoError(err)
	n := New[string]().Receivers(r).B(jet.StringPtr("data"))
	s.Equal("data", *n.Msg)
	s.Len(n.Receivers, 1)
	s.Equal("type", n.Receivers[0].Type)
	s.Equal("receiver", n.Receivers[0].Receiver)
}

func (s *testNotificationSuite) Test_ReceiversFilterByType() {
	rec1 := func(resolver Resolver) ([]*Receiver, error) {
		perms := resolver.Resolve()
		s.Empty(perms.Items)
		return []*Receiver{{Receiver: "receiver", Type: "type1"}}, nil
	}
	rec2 := func(resolver Resolver) ([]*Receiver, error) {
		perms := resolver.Resolve()
		s.Empty(perms.Items)
		return []*Receiver{{Receiver: "receiver", Type: "type2"}}, nil
	}

	r, err := NewReceivers().Scopes(rec1, rec1, rec2).B()
	s.NoError(err)
	n := New[string]().Receivers(r).Types("type2").B(jet.StringPtr("data1"))
	s.Equal("data1", *n.Msg)
	s.Len(n.Receivers, 1)

	n = New[string]().Receivers(r).Types("type1").B(jet.StringPtr("data2"))
	s.Equal("data2", *n.Msg)
	s.Len(n.Receivers, 2)
}

func (s *testNotificationSuite) Test_FewReceiversScope() {
	rec := func(resolver Resolver) ([]*Receiver, error) {
		perms := resolver.Resolve()
		s.Empty(perms.Items)

		return []*Receiver{{Receiver: "receiver", Type: "type"}}, nil
	}

	r, err := NewReceivers().Scopes(rec, rec, rec).B()
	s.NoError(err)
	n := New[string]().Receivers(r).B(jet.StringPtr("data"))
	s.Equal("data", *n.Msg)
	s.Len(n.Receivers, 3)
}

func (s *testNotificationSuite) Test_PermResolver() {
	resolve := func() Resolver {
		return Resource("rwd", "resource")
	}

	rec := func(resolver Resolver) ([]*Receiver, error) {
		perms := resolver.Resolve()
		s.Len(perms.Items, 1)
		s.Equal("resource", perms.Items[0].Resource)
		s.Len(perms.Items[0].Perm, 3)
		s.Contains(perms.Items[0].Perm, "r")
		s.Contains(perms.Items[0].Perm, "w")
		s.Contains(perms.Items[0].Perm, "d")

		return []*Receiver{{Receiver: "receiver1", Type: "type"}, {Receiver: "receiver2", Type: "type"}}, nil
	}

	r, err := NewReceivers().Scopes(rec).Permissions(resolve()).B()
	s.NoError(err)
	n := New[string]().Receivers(r).B(jet.StringPtr("data"))
	s.NoError(err)
	s.Equal("data", *n.Msg)
	s.Len(n.Receivers, 2)
}
