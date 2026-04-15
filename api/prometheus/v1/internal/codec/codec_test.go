package codec

import (
	"testing"

	json "github.com/json-iterator/go"

	"github.com/prometheus/common/model"
)

func TestRegisterCustomJSONCodecs(t *testing.T) {
	t.Parallel()

	RegisterCustomJSONCodecs()

	pair := model.SamplePair{
		Timestamp: 1010,
		Value:     model.SampleValue(20),
	}
	body, err := json.Marshal(pair)
	if err != nil {
		t.Fatalf("marshal sample pair: %v", err)
	}
	if string(body) != `[1.010,"20"]` {
		t.Fatalf("unexpected encoded sample pair: %s", body)
	}

	var stream model.SampleStream
	if err := json.Unmarshal([]byte(`{"metric":{"__name__":"up"},"values":[[1.010,"20"]]}`), &stream); err != nil {
		t.Fatalf("unmarshal sample stream: %v", err)
	}
	if got, want := len(stream.Values), 1; got != want {
		t.Fatalf("unexpected value count: got %d want %d", got, want)
	}
	if got, want := stream.Values[0], pair; got != want {
		t.Fatalf("unexpected decoded sample pair: got %#v want %#v", got, want)
	}
}

func TestUnmarshalQueryResultValue(t *testing.T) {
	t.Parallel()

	value, err := UnmarshalQueryResultValue([]byte(`{
		"resultType":"vector",
		"result":[{"metric":{"__name__":"up"},"value":[1,"2"]}]
	}`))
	if err != nil {
		t.Fatalf("unmarshal query result: %v", err)
	}

	vector, ok := value.(model.Vector)
	if !ok {
		t.Fatalf("expected model.Vector, got %T", value)
	}
	if got, want := len(vector), 1; got != want {
		t.Fatalf("unexpected vector length: got %d want %d", got, want)
	}
	if got, want := vector[0].Value, model.SampleValue(2); got != want {
		t.Fatalf("unexpected sample value: got %v want %v", got, want)
	}
}
