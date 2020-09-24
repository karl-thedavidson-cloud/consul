package pbservice

import (
	"os"
	"strconv"
	"testing"
	"time"

	fuzz "github.com/google/gofuzz"
	"github.com/hashicorp/consul/agent/structs"
	"gotest.tools/v3/assert"
)

func TestNewCheckServiceNodeFromStructs_RoundTrip(t *testing.T) {
	fn := func(t *testing.T, fuzzer *fuzz.Fuzzer) {
		var target structs.CheckServiceNode
		fuzzer.Fuzz(&target)

		s := NewCheckServiceNodeFromStructs(&target)
		result := CheckServiceNodeToStructs(s)
		assert.DeepEqual(t, &target, result)
	}
	repeat(t, fn)
}

func repeat(t *testing.T, fn func(t *testing.T, fuzzer *fuzz.Fuzzer)) {
	reps := getEnvIntWithDefault(t, "TEST_FUZZ_COUNT", 1)
	seed := getEnvIntWithDefault(t, "TEST_RANDOM_SEED", time.Now().UnixNano())
	t.Logf("using seed %d for %d repetitions", seed, reps)

	fuzzer := fuzz.NewWithSeed(seed)
	fuzzer.Funcs(func(m map[string]interface{}, c fuzz.Continue) {
		// Populate it with some random stuff of different types
		// Int -> Float since trip through protobuf.Value will force this.
		m[c.RandString()] = interface{}(float64(c.RandUint64()))
		m[c.RandString()] = interface{}(c.RandString())
		m[c.RandString()] = interface{}([]interface{}{c.RandString(), c.RandString()})
		m[c.RandString()] = interface{}(map[string]interface{}{c.RandString(): c.RandString()})
	})
	fuzzer.Funcs(func(i *int, c fuzz.Continue) {
		// Potentially controversial but all of the int values we care about
		// instructs are expected to be lower than 32 bits - if they weren't then
		// we'd use (u)int64 and would already be breaking 32-bit compat. So we
		// explicitly call those int32 in protobuf. But gofuzz will happily assign
		// them values out of range of an in32 so we need to restrict it or the trip
		// through PB truncates them and fails the tests.
		*i = int(int32(c.RandUint64()))
	})
	fuzzer.Funcs(func(i *uint, c fuzz.Continue) {
		// See above
		*i = uint(uint32(c.RandUint64()))
	})

	for i := 0; i < int(reps); i++ {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			fn(t, fuzzer)
		})
	}
}

func getEnvIntWithDefault(t *testing.T, key string, d int64) int64 {
	t.Helper()
	raw, ok := os.LookupEnv(key)
	if !ok {
		return d
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		t.Fatalf("invald value for %v: %v", key, err.Error())
	}
	return int64(v)
}
