# uidgen
A fast and fully configurable unique ID generator similar to Twitter Snowflake. 

# Installing

```shell
go get github.com/sevings/uidgen
```

# Usage
Import the package into your project. Then create a new UidGenerator with the desired configuration.
You can start with the default SnowflakeConfig and modify it as needed. Following that, each call
to the NextID() will generate a new UniqueID.

```go
package main

import (
	"fmt"
	
	"github.com/sevings/uidgen"
)

func main() {
	// Get the default config.
	cfg := uidgen.SnowflakeConfig
	// Assume that we have two servers and this is the second.
	cfg.SrvLen = 1
	cfg.SrvID = 1
	// We also can use seconds (2^30 ns) instead of milliseconds (2^20 ns).
	cfg.IntervalLen = 30
	// 36 bits for the epoch field will be enough.
	cfg.EpochLen = 36
	// So we can use more bits for the sequence counter.
	cfg.CntLen = 26

	// Create a new UidGenerator.
	gen, err := uidgen.NewUidGenerator(cfg, 0)
	if err != nil {
		fmt.Println(err)
		return
	}

	// Generate an unique ID.
	id := gen.NextID()

	fmt.Printf("String ID:      %s\n", id)
	fmt.Printf("Int64 ID:       %d\n", id.Int64())
	fmt.Printf("Base32 ID:      %s\n", gen.ToBase32(id))
	fmt.Printf("Unix epoch:     %d\n", gen.Unix(id))
	fmt.Printf("Unixnano epoch: %d\n", gen.UnixNano(id))
	fmt.Printf("Server ID:      %d\n", gen.ServerID(id))
	fmt.Printf("Counter:        %d\n", gen.Count(id))
}

```

# Performance 

```
goos: linux
goarch: amd64
pkg: github.com/sevings/uidgen
cpu: Intel(R) Pentium(R) CPU 4415U @ 2.30GHz
BenchmarkUidGenerator
BenchmarkUidGenerator-4   	16650181	        62.68 ns/op	       0 B/op	       0 allocs/op
PASS
```
