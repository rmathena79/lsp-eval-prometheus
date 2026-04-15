package v1

import "github.com/prometheus/client_golang/api/prometheus/v1/internal/codec"

func init() {
	codec.RegisterCustomJSONCodecs()
}
