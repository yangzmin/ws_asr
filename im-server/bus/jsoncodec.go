package bus

import (
    "encoding/json"

    "google.golang.org/grpc/encoding"
)

// jsonCodec implements gRPC encoding.Codec using JSON for payloads
type jsonCodec struct{}

func (jsonCodec) Name() string { return "json" }
func (jsonCodec) Marshal(v interface{}) ([]byte, error) { return json.Marshal(v) }
func (jsonCodec) Unmarshal(data []byte, v interface{}) error { return json.Unmarshal(data, v) }

func init() {
    encoding.RegisterCodec(jsonCodec{})
}