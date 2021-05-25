package loopia

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/kolo/xmlrpc"
	"github.com/stretchr/testify/assert"
	"github.com/subchen/go-xmldom"
)

var (
	handlers map[string]methodHandler
)

type methodHandler func(t *testing.T, w http.ResponseWriter, params []string)

func init() {
	handlers = make(map[string]methodHandler)
	handlers["getZoneRecords"] = getZoneRecordsHandler
	handlers["getSubdomains"] = getSubdomainsHandler
	handlers["addSubdomain"] = addSubdomainHandler
	handlers["addZoneRecord"] = addZoneRecordHandler
	handlers["updateZoneRecord"] = updateZoneRecordHandler
	handlers["removeZoneRecord"] = returnOkHandler
	handlers["removeSubdomain"] = returnOkHandler
}

type testContext struct {
	mux *http.ServeMux

	rpc    *xmlrpc.Client
	server *httptest.Server
}

func (tc *testContext) getProvider() *Provider {
	p := &Provider{}
	p.rpc = tc.rpc
	return p
}

func setupTest(t *testing.T) *testContext {
	tc := &testContext{}
	tc.mux = http.NewServeMux()
	tc.server = httptest.NewServer(tc.mux)
	tc.rpc, _ = xmlrpc.NewClient(tc.server.URL, nil)
	tc.mux.HandleFunc("/", apiHandler(t))
	return tc
}

func teardownTest(tc *testContext) {
	if tc.server != nil {
		tc.server.Close()
	}
}

func apiHandler(t *testing.T) func(w http.ResponseWriter, r *http.Request) {

	return func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, r.Method, "POST")
		body, err := ioutil.ReadAll(r.Body)
		assert.NoError(t, err, "Error reading request body")
		strBody := string(body)
		doc := xmldom.Must(xmldom.ParseXML(strBody))
		root := doc.Root

		method := root.GetChild("methodName").Text
		params := root.GetChild("params")
		values := params.Query("//value")
		// logger.Debug().Str("method", method).Int("values", len(values)).Msg("request")
		strValues := []string{}
		for _, v := range values {
			strValues = append(strValues, v.FirstChild().Text)
		}

		h := handlers[method]
		if h != nil {
			h(t, w, strValues)
			return
		}
		t.Errorf("method %s not implemented", method)
		// byteArray, _ := ioutil.ReadFile("testdata/error.xml")
		// fmt.Fprint(w, string(byteArray[:]))
	}
}

func getSubdomainsHandler(t *testing.T, w http.ResponseWriter, params []string) {
	// logger.Debug().Str("zone", params[3]).Msg("getSubdomainsHandler")
	byteArray, _ := ioutil.ReadFile("testdata/subdomains.xml")
	fmt.Fprint(w, string(byteArray[:]))
}

func getZoneRecordsHandler(t *testing.T, w http.ResponseWriter, params []string) {

	recname := params[4]
	if recname == "*" {
		recname = ""
	}
	filename := fmt.Sprintf("testdata/zone_records_%s.xml", recname)
	_, err := os.Stat(filename)
	if os.IsNotExist(err) {
		filename = "testdata/empty_list.xml"
	}
	// logger.Debug().Str("zone", params[3]).Str("name", params[4]).Str("filename", filename).Msg("getZoneRecordsHandler")
	byteArray, _ := ioutil.ReadFile(filename)
	fmt.Fprint(w, string(byteArray[:]))
}

func addSubdomainHandler(t *testing.T, w http.ResponseWriter, params []string) {
	//TODO: validate params
	// fmt.Printf("params:%v", params)
	fmt.Printf(" > addSubdomainHandler(%s, %s)\n", params[3], params[4])
	assert.Len(t, params, 5)
	assert.GreaterOrEqual(t, len(params[4]), 1)
	byteArray, _ := ioutil.ReadFile("testdata/ok.xml")
	fmt.Fprint(w, string(byteArray[:]))
}

func addZoneRecordHandler(t *testing.T, w http.ResponseWriter, params []string) {
	// fmt.Printf(" > addZoneRecordHandler(%+v)\n", params[4:])
	// logger.Debug().Str("name", params[4]).Str("value", params[7]).Msg("addZoneRecordHandler")
	byteArray, _ := ioutil.ReadFile("testdata/ok.xml")
	fmt.Fprint(w, string(byteArray[:]))
}

func updateZoneRecordHandler(t *testing.T, w http.ResponseWriter, params []string) {
	// logger.Debug().Str("name", params[4]).Str("value", params[7]).Msg("updateZoneRecordHandler")
	byteArray, _ := ioutil.ReadFile("testdata/ok.xml")
	fmt.Fprint(w, string(byteArray[:]))
}

func returnOkHandler(t *testing.T, w http.ResponseWriter, params []string) {
	// logger.Debug().Msg("returnOK Handler")
	byteArray, _ := ioutil.ReadFile("testdata/ok.xml")
	fmt.Fprint(w, string(byteArray[:]))
}
