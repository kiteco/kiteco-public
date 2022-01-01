package account

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/kiteco/kiteco/kite-go/web"
)

func unmarshalBody(prefix string, r *http.Request, res interface{}) web.ErrorData {
	buf, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return web.ErrorData{
			Debug: fmt.Sprintf("%s account.unmarshalBody: error reading request body: %v", prefix, err),
			Msg:   "We were unable to read your request, please try again.",
			Code:  http.StatusInternalServerError,
		}.RollbarError()
	}

	if err := json.Unmarshal(buf, res); err != nil {
		return web.ErrorData{
			Debug: fmt.Sprintf("%s account.unmarshalBody: error unmarshalling request: %v", prefix, err),
			Msg:   "Yikes something bad happened, please try again.",
			Code:  http.StatusBadRequest,
		}.RollbarError()
	}
	return web.ErrorData{}
}

func marshalResponse(prefix string, w http.ResponseWriter, resp interface{}) web.ErrorData {
	buf, err := json.Marshal(resp)
	if err != nil {
		// this is very bad, since the actually work of handling the request actually finished,
		// and so resubmitting the same request will probably not work
		return web.ErrorData{
			Debug: fmt.Sprintf("%s account.marshalResponse: error marshalling response: %v", prefix, err),
			Msg:   "Oops something bad happened, please contact support@kite.com.",
			Code:  http.StatusInternalServerError,
		}.RollbarCritical()
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(buf) // 200 set automatically
	return web.ErrorData{}
}
