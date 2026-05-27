//go:build integration

package centrifugo

import (
	"github.com/centrifugal/centrifuge-go"
	"github.com/stretchr/testify/suite"
	"github.com/zloevil/jet"
	"math"
	"testing"
	"time"
)

type testSuite struct {
	jet.Suite
}

func (s *testSuite) SetupSuite() {
	s.Suite.Init(func() jet.CLogger { return jet.L(jet.InitLogger(&jet.LogConfig{Level: jet.TraceLevel})) })
}

func TestSuite(t *testing.T) {
	suite.Run(t, new(testSuite))
}

var cfgClient = &ClientConfig{
	Url: "ws://localhost:18000/connection/websocket",
}

var cfgServer = &ServerConfig{
	Host:   "localhost",
	Port:   "20000",
	ApiKey: "cbf46e80-3e00-4642-8f3a-369b8707304d",
	Secret: "b10b2ab3-8e29-428b-85cb-42a32ba6ea57",
}

type Payload struct {
	V string `json:"v"`
}

func (s *testSuite) Test_WithDynamicSubscription() {

	userId := jet.NewId()
	channel := jet.NewRandString()

	// gen token
	token := s.genToken(userId, nil)

	// connect server
	srv := NewServer(cfgServer, s.L)
	s.NoError(srv.Connect(s.Ctx))
	defer srv.Close(s.Ctx)

	// connect client
	cl := NewClient(cfgClient, s.L)
	s.NoError(cl.Connect(s.Ctx, token))
	defer cl.Close(s.Ctx)

	// gen subscribe token
	subToken, err := GenerateSubscribeToken(s.Ctx, cfgServer.Secret, userId, channel, time.Minute)
	s.NoError(err)
	s.NotEmpty(subToken)

	var data []byte

	// client subscribe
	s.NoError(cl.Subscribe(s.Ctx, subToken, channel, func(p centrifuge.Publication) error {
		s.L().TrcObj("%v", p)
		data = p.Data
		return nil
	}))
	s.NoError(err)

	time.Sleep(time.Second)

	// server publish
	pl := &Payload{V: jet.NewRandString()}
	s.NoError(srv.Publish(s.Ctx, channel, pl))

	s.NoError(<-jet.Await(func() (bool, error) {
		return len(data) > 0, nil
	}, time.Millisecond*300, time.Second*3))

}

func (s *testSuite) Test_WithTokenSubscription() {

	userId := jet.NewId()
	channel := jet.NewRandString()

	// gen token
	token := s.genToken(userId, []string{channel})

	// connect server
	srv := NewServer(cfgServer, s.L)
	s.NoError(srv.Connect(s.Ctx))
	defer srv.Close(s.Ctx)

	// connect client
	cl := NewClient(cfgClient, s.L)
	s.NoError(cl.Connect(s.Ctx, token))
	defer cl.Close(s.Ctx)

	// gen subscribe token
	subToken, err := GenerateSubscribeToken(s.Ctx, cfgServer.Secret, userId, channel, time.Minute)
	s.NoError(err)
	s.NotEmpty(subToken)

	var data []byte

	// client subscribe
	s.NoError(cl.OnPublication(s.Ctx, func(p centrifuge.ServerPublicationEvent) error {
		s.L().TrcObj("%v", p)
		data = p.Data
		return nil
	}))
	s.NoError(err)

	time.Sleep(time.Second)

	// server publish
	pl := &Payload{V: jet.NewRandString()}
	s.NoError(srv.Publish(s.Ctx, channel, pl))

	s.NoError(<-jet.Await(func() (bool, error) {
		return len(data) > 0, nil
	}, time.Millisecond*300, time.Second*3))

}

func (s *testSuite) Test_WithPresenceInfo() {

	userId := jet.NewId()
	channel := jet.NewRandString()

	// connect server
	srv := NewServer(cfgServer, s.L)
	s.NoError(srv.Connect(s.Ctx))
	defer srv.Close(s.Ctx)

	type sessInfo struct {
		V string
	}

	info := &sessInfo{V: jet.NewRandString()}

	// gen client connect token
	token, err := GenerateConnectToken(s.Ctx, cfgServer.Secret, userId, []string{channel}, time.Second*5, info)
	s.NoError(err)

	// connect client
	cl := NewClient(cfgClient, s.L)
	s.NoError(cl.Connect(s.Ctx, token))
	defer cl.Close(s.Ctx)

	// await client connected
	s.NoError(<-jet.Await(func() (bool, error) {
		return cl.Connected(), nil
	}, time.Millisecond*300, time.Second*3))

	// client subscribe
	s.NoError(cl.OnPublication(s.Ctx, func(p centrifuge.ServerPublicationEvent) error {
		s.L().DbgF("%v", p)
		return nil
	}))
	s.NoError(err)

	time.Sleep(time.Second)

	presence, err := srv.GetPresence(s.Ctx, channel)
	s.NoError(err)
	s.L().DbgF("%v", presence)

	s.NotEmpty(presence.Presence[cl.ClientId()])
	s.Equal(presence.Presence[cl.ClientId()].User, userId)
	s.NotEmpty(presence.Presence[cl.ClientId()].ConnInfo)

	infoRes, err := jet.JsonDecode[sessInfo](presence.Presence[cl.ClientId()].ConnInfo)
	s.NoError(err)
	s.NotEmpty(infoRes)
	s.Equal(infoRes.V, info.V)

}

func (s *testSuite) Test_DisconnectUser() {

	userId := jet.NewId()
	channel := jet.NewRandString()

	// gen token
	token := s.genToken(userId, nil)

	// connect server
	srv := NewServer(cfgServer, s.L)
	s.NoError(srv.Connect(s.Ctx))
	defer srv.Close(s.Ctx)

	// connect client
	cl := NewClient(cfgClient, s.L)
	s.NoError(cl.Connect(s.Ctx, token))
	defer cl.Close(s.Ctx)

	// gen subscribe token
	subToken, err := GenerateSubscribeToken(s.Ctx, cfgServer.Secret, userId, channel, time.Minute)
	s.NoError(err)
	s.NotEmpty(subToken)

	var data []byte

	// client subscribe
	s.NoError(cl.Subscribe(s.Ctx, subToken, channel, func(p centrifuge.Publication) error {
		s.L().TrcObj("%v", p)
		data = p.Data
		return nil
	}))

	time.Sleep(time.Second)

	// server publish
	pl := &Payload{V: jet.NewRandString()}
	s.NoError(srv.Publish(s.Ctx, channel, pl))

	s.NoError(<-jet.Await(func() (bool, error) {
		return len(data) > 0, nil
	}, time.Millisecond*300, time.Second*3))

	disconnected := false
	cl.(*clientImpl).client.OnDisconnected(func(event centrifuge.DisconnectedEvent) {
		disconnected = true
	})

	// disconnect user
	s.NoError(srv.DisconnectUser(s.Ctx, userId))

	s.NoError(<-jet.Await(func() (bool, error) {
		return disconnected, nil
	}, time.Millisecond*300, time.Second*3))

}

func (s *testSuite) genToken(userId string, autoSubscribeChannels []string) string {
	claims := map[string]any{
		"expired_at": time.Now().Add(time.Duration(math.MaxInt)).Unix(),
		"created_at": time.Now().Unix(),
		"sub":        userId,
	}
	if len(autoSubscribeChannels) > 0 {
		claims["channels"] = autoSubscribeChannels
	}

	token, err := jet.GenJwtToken(s.Ctx, &jet.JwtRequest{
		Secret:   []byte(cfgServer.Secret),
		ExpireAt: time.Now().Add(time.Duration(math.MaxInt)),
		Claims:   claims,
	})
	s.NoError(err)
	return token
}
