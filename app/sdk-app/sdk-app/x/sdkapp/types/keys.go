package types

const (
	// ModuleName defines the module name
	ModuleName = "sdkapp"

	// StoreKey defines the primary module store key
	StoreKey = ModuleName

	// MemStoreKey defines the in-memory store key
	MemStoreKey = "mem_sdkapp"
)

var (
	ParamsKey = []byte("p_sdkapp")
)

func KeyPrefix(p string) []byte {
	return []byte(p)
}
