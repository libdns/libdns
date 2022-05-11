package nicrudns

import "github.com/pkg/errors"

var (
	XmlDecodeError            = errors.New(`xml decode error`)
	XmlEncodeError            = errors.New(`xml encode error`)
	JsonDecodeError           = errors.New(`json decode error`)
	JsonEncodeError           = errors.New(`json encode error`)
	CreateFileError           = errors.New(`create file error`)
	ReadFileError             = errors.New(`read file error`)
	ApiNonSuccessError        = errors.New(`api non-success error`)
	RequestError              = errors.New(`request error`)
	ResponseError             = errors.New(`response error`)
	InvalidStatusCode         = errors.New(`invalid status code`)
	BufferReadError           = errors.New(`buffer read error`)
	Oauth2ClientError         = errors.New(`oauth2 client error`)
	NameFilterError           = errors.New(`name filter error`)
	TargetFilterError         = errors.New(`target filter error`)
	UpdateTokenCacheFileError = errors.New(`update token cache file error`)
	AuthorizationError        = errors.New(`authorization error`)
	NotImplementedRecordType  = errors.New(`not implemented record type`)
)
