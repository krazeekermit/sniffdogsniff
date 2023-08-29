package core_test

import (
	"fmt"
	"testing"

	"github.com/sniffdogsniff/core"
)

func TestMarshal_RpcRequest(t *testing.T) {
	core.InitializeGob()

	sr := core.NewSearchResult("title1", "http://www.google.com/", core.ResultPropertiesMap{}, core.LINK_DATA_TYPE)

	rpcRequest := core.RpcRequest{
		FuncCode:  134,
		Id:        "fakeId",
		Arguments: []any{"cat", uint8(125), false, sr},
	}

	bytez, err := core.GobMarshal(rpcRequest)
	if err != nil {
		t.Fatal(err)
	}

	var unRequest core.RpcRequest
	if core.GobUnmarshal(bytez, &unRequest) != nil {
		t.Fatal()
	}

	if unRequest.FuncCode != 134 {
		t.Fatal()
	}

	if unRequest.Id != "fakeId" {
		t.Fatal()
	}

	if len(unRequest.Arguments) != 4 {
		t.Fatal()
	}

	arg0, ok := unRequest.Arguments[0].(string)
	if !ok {
		t.Fatal()
	}
	if arg0 != "cat" {
		t.Fatal()
	}

	arg1, ok := unRequest.Arguments[1].(uint8)
	if !ok {
		t.Fatal()
	}
	if arg1 != 125 {
		t.Fatal()
	}

	arg2, ok := unRequest.Arguments[2].(bool)
	if !ok {
		t.Fatal()
	}
	if arg2 != false {
		t.Fatal()
	}

	arg3, ok := unRequest.Arguments[3].(core.SearchResult)
	if !ok {
		t.Fatal()
	}
	assertSearchResult(arg3, sr.ResultHash, sr.Title, sr.Url, sr.Properties, t)

}

func TestMarshal_RpcResponse(t *testing.T) {
	core.InitializeGob()

	sr1 := core.NewSearchResult("title1", "http://www.google.com/", core.ResultPropertiesMap{}, core.LINK_DATA_TYPE)
	sr2 := core.NewSearchResult("title2", "http://www.yahoo.com/", core.ResultPropertiesMap{}, core.LINK_DATA_TYPE)

	rpcResponse := core.RpcResponse{
		ErrCode:  20,
		Id:       "fakeId",
		RetValue: []core.SearchResult{sr1, sr2},
	}

	bytez, err := core.GobMarshal(rpcResponse)
	if err != nil {
		t.Fatal(err)
	}

	var unResponse core.RpcResponse
	if core.GobUnmarshal(bytez, &unResponse) != nil {
		t.Fatal()
	}

	if unResponse.ErrCode != 20 {
		t.Fatal()
	}

	if unResponse.Id != "fakeId" {
		t.Fatal()
	}

	arg0, ok := unResponse.RetValue.([]core.SearchResult)
	if !ok {
		fmt.Printf("type %T\n", arg0)
		t.Fatal()
	}
	if len(arg0) != 2 {
		t.Fatal()
	}
	assertSearchResult(arg0[0], sr1.ResultHash, sr1.Title, sr1.Url, sr1.Properties, t)
	assertSearchResult(arg0[1], sr2.ResultHash, sr2.Title, sr2.Url, sr2.Properties, t)
}
