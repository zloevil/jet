package notification

import (
	"github.com/zloevil/jet"
)

type Receiver struct {
	Type     string
	Receiver string
}

type Notification[T any] struct {
	Receivers []*Receiver
	Msg       *T
}

type ReceiverFn func(resolver Resolver) ([]*Receiver, error)

type Builder[T any] interface {
	Receivers([]*Receiver) Builder[T]
	Types(types ...string) Builder[T]
	B(msg *T) *Notification[T]
}

type notificationImpl[T any] struct {
	n         *Notification[T]
	receivers []*Receiver
	types     []string
}

func New[T any]() Builder[T] {
	return &notificationImpl[T]{
		n: &Notification[T]{},
	}
}

func (n *notificationImpl[T]) Types(types ...string) Builder[T] {
	n.types = types
	return n
}

func (n *notificationImpl[T]) Receivers(r []*Receiver) Builder[T] {
	n.receivers = r
	return n
}

func (n *notificationImpl[T]) B(msg *T) *Notification[T] {
	if len(n.types) == 0 {
		n.n.Receivers = n.receivers
	} else {
		n.n.Receivers = jet.Filter(n.receivers, func(receiver *Receiver) bool {
			for _, tp := range n.types {
				if tp == receiver.Type {
					return true
				}
			}
			return false
		})
	}
	n.n.Msg = msg
	return n.n
}

type receiversImpl struct {
	receivers      []ReceiverFn
	policyResolver Resolver
}

type Receivers interface {
	Permissions(resolver Resolver) Receivers
	Scopes(fns ...ReceiverFn) Receivers
	B() ([]*Receiver, error)
}

func NewReceivers() Receivers {
	return &receiversImpl{}
}

func (n *receiversImpl) Scopes(fns ...ReceiverFn) Receivers {
	n.receivers = fns
	return n
}

func (n *receiversImpl) B() ([]*Receiver, error) {
	if n.policyResolver == nil {
		n.policyResolver = n.emptyResolver()
	}
	var res []*Receiver
	for _, fn := range n.receivers {
		receivers, err := fn(n.policyResolver)
		if err != nil {
			return nil, err
		}
		if len(receivers) != 0 {
			res = append(res, receivers...)
		}
	}
	return res, nil
}

func (n *receiversImpl) Permissions(resolver Resolver) Receivers {
	n.policyResolver = resolver
	return n
}

func (n *receiversImpl) emptyResolver() Resolver {
	return Resource("")
}
