package ewelink

import (
	"fmt"
	"io"
	"io/ioutil"
	"strings"
)

type responseDecoder interface {
	decode(subject Response, response io.ReadCloser, status int) (Response, error)
}

type jsonResponseDecoder struct{}

func newJSONResponseDecoder() responseDecoder {
	return &jsonResponseDecoder{}
}

// DebugResponse enable or disable the trace of the response in stdout
var DebugResponse = false

func (j jsonResponseDecoder) decode(subject Response, response io.ReadCloser, status int) (Response, error) {
	responseAsString := tryReadCloserToString(response)

	if DebugResponse {
		fmt.Println("Response " + responseAsString)
	}

	// Decode the response into the expected response type
	decoded, err := subject.Decode(ioutil.NopCloser(strings.NewReader(responseAsString)))
	if err != nil {
		return nil, fmt.Errorf("failed to decode response %w", err)
	}

	// Check whether we encountered an API error
	if decoded.Envelope().Code() > 0 {
		return nil, j.decodeAsAPIError(decoded, status)
	}

	return decoded, nil
}

func (j jsonResponseDecoder) decodeAsAPIError(response Response, status int) error {
	return toAPIError(response.Envelope())
}

func toAPIError(envelope Envelope) error {
	if resp, ok := envelope.(*httpResponse); ok {
		return toHTTPResponseAPIError(resp)
	}

	return &ApiError{Code: envelope.Code(), Message: envelope.Cause()}
}

func toHTTPResponseAPIError(response *httpResponse) error {
	switch response.Code() {
	case wrongRegion:
		return &wrongRegionError{
			Region:   response.Region,
			ApiError: ApiError{Code: response.Code(), Message: response.Message, Cause: APIErrorCauses.WrongRegion},
		}
	case authenticationError:
		return &ApiError{Code: response.Code(), Message: response.Message, Cause: APIErrorCauses.AuthenticationError}
	case invalidRequest:
	case notAcceptable:
		return &ApiError{Code: response.Code(), Message: response.Message, Cause: APIErrorCauses.InvalidRequest}
	case internalError:
		return &ApiError{Code: response.Code(), Message: response.Message, Cause: APIErrorCauses.InternalError}
	}

	return &ApiError{Code: response.Code(), Message: response.Cause(), Cause: APIErrorCauses.UnknownError}
}

// nolint:deadcode,unused
func toWebsocketResponseAPIError(response *websocketResponse) error {
	return nil
}
