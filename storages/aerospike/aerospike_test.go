//go:build example

package aerospike

import (
	"context"
	"encoding/json"
	aero "github.com/aerospike/aerospike-client-go/v8"
	"github.com/stretchr/testify/assert"
	"github.com/zloevil/jet"
	"math/rand"
	"testing"
	"time"
)

var logger = jet.InitLogger(&jet.LogConfig{Level: jet.TraceLevel})
var logf = func() jet.CLogger {
	return jet.L(logger)
}

type data struct {
	Id  string `json:"id"`
	Val string `json:"val"`
}

func connect(t *testing.T) Aerospike {
	// open
	aes := New()
	err := aes.Open(ctx, cfg, logf)
	if err != nil {
		t.Fatal(err)
	}
	return aes
}

func init() {
	rand.Seed(time.Now().UnixNano())
}

var (
	cfg = &Config{
		Host: "localhost",
		Port: 33000,
	}
	ctx = context.Background()
	ns  = "cache"
)

func genMany(n int) []*data {
	r := make([]*data, n)
	for i := 0; i < n; i++ {
		r[i] = &data{
			Id:  jet.NewId(),
			Val: jet.NewRandString(),
		}
	}
	return r
}

func Test_CRUD(t *testing.T) {

	aes := connect(t)
	defer aes.Close(ctx)

	b := &data{
		Id:  jet.NewId(),
		Val: jet.NewRandString(),
	}
	bj, _ := json.Marshal(b)
	putKey, err := aero.NewKey(ns, "", b.Id)
	if err != nil {
		t.Fatal(err)
	}
	bins := aero.BinMap{"data": bj}
	// write the bins
	err = aes.Instance().Put(nil, putKey, bins)
	if err != nil {
		t.Fatal(err)
	}

	getKey, err := aero.NewKey(ns, "", b.Id)
	if err != nil {
		t.Fatal(err)
	}
	rec, err := aes.Instance().Get(nil, getKey)
	if err != nil {
		t.Fatal(err)
	}

	assert.NotEmpty(t, rec.Bins)
	assert.NotEmpty(t, rec.Bins["data"])

	v := rec.Bins["data"].([]byte)
	actual := &data{}
	_ = json.Unmarshal(v, &actual)

	assert.Equal(t, actual, b)

	_, err = aes.Instance().Delete(nil, getKey)
	if err != nil {
		t.Fatal(err)
	}
}

func Test_Put_GetBatch(t *testing.T) {
	aes := connect(t)
	defer aes.Close(ctx)

	bids := genMany(1000)
	keys := make([]*aero.Key, len(bids))

	// put
	writePolicy := aero.NewWritePolicy(0, 10)
	for i, b := range bids {
		bj, _ := json.Marshal(b)
		putKey, err := aero.NewKey(ns, "", b.Id)
		if err != nil {
			t.Fatal(err)
		}
		keys[i] = putKey
		bin := aero.NewBin("data", bj)
		// write the bins
		err = aes.Instance().PutBins(writePolicy, putKey, bin)
		if err != nil {
			t.Fatal(err)
		}
	}

	batchPolicy := aero.NewBatchPolicy()
	records, err := aes.Instance().BatchGet(batchPolicy, keys, "data")
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, len(bids), len(records))

}

func Test_Put_ScanAll(t *testing.T) {
	aes := connect(t)
	defer aes.Close(ctx)

	bids := genMany(100)
	keys := make([]*aero.Key, len(bids))

	// put
	set := jet.NewRandString()[:10]
	writePolicy := aero.NewWritePolicy(0, 10)
	for i, b := range bids {
		bj, _ := json.Marshal(b)
		putKey, err := aero.NewKey(ns, set, b.Id)
		if err != nil {
			t.Fatal(err)
		}
		keys[i] = putKey
		bin := aero.NewBin("bin", bj)
		// write the bins
		err = aes.Instance().PutBins(writePolicy, putKey, bin)
		if err != nil {
			t.Fatal(err)
		}
	}

	scanPolicy := aero.NewScanPolicy()
	recordSet, err := aes.Instance().ScanAll(scanPolicy, ns, set, "bin")
	if err != nil {
		t.Fatal(err)
	}
	assert.NotEmpty(t, recordSet)

	var r []aero.BinMap

	for rec := range recordSet.Results() {
		if rec.Err != nil {
			// if there was an error, stop
			t.Fatal(rec.Err)
		}
		r = append(r, rec.Record.Bins)
	}
	assert.Equal(t, len(bids), len(r))
	for _, rec := range r {
		v := rec["bin"].([]byte)
		actual := &data{}
		_ = json.Unmarshal(v, &actual)
		assert.NotEmpty(t, actual)
		assert.NotEmpty(t, actual.Id)
		assert.NotEmpty(t, actual.Val)
	}

}

func Test_ExpList_Contains(t *testing.T) {
	aes := connect(t)
	defer aes.Close(ctx)

	type obj struct {
		Id     string
		Values []string
	}
	v1 := jet.NewRandString()
	v2 := jet.NewRandString()
	v3 := jet.NewRandString()
	v4 := jet.NewRandString()
	values := []*obj{
		{
			Id:     jet.NewRandString(),
			Values: []string{v1, v2},
		},
		{
			Id:     jet.NewRandString(),
			Values: []string{v2, v3},
		},
		{
			Id:     jet.NewRandString(),
			Values: []string{v3, v4, v1, v2},
		},
	}

	// put
	writePolicy := aero.NewWritePolicy(0, 60)
	writePolicy.SendKey = true
	for _, v := range values {
		putKey, err := aero.NewKey(ns, "test_values", v.Id)
		if err != nil {
			t.Fatal(err)
		}
		err = aes.Instance().PutBins(nil, putKey, aero.NewBin("values", v.Values))
		if err != nil {
			t.Fatal(err)
		}
	}

	queryPolicy := aero.NewQueryPolicy()
	queryPolicy.SendKey = true
	queryPolicy.FilterExpression =
		aero.ExpGreater(
			aero.ExpListGetByValueList(
				aero.ListReturnTypeCount,
				aero.ExpListValueVal(v3, v4),
				aero.ExpListBin("values"),
			),
			aero.ExpIntVal(0),
		)

	statement := aero.NewStatement(ns, "test_values")

	recordSet, err := aes.Instance().Query(queryPolicy, statement)
	if err != nil {
		t.Fatal(err)
	}
	var res []*obj
	for r := range recordSet.Results() {
		if r.Err != nil {
			t.Fatal(err)
		} else {
			v, err := AsStrings(ctx, r.Record.Bins, "values")
			if err != nil {
				t.Fatal(err)
			}
			res = append(res, &obj{
				//Id:     r.Record.Key.Value().String(),
				Values: v},
			)
		}
	}
	assert.NotEmpty(t, res)
}
