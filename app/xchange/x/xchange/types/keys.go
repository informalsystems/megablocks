package types

const (
	// ModuleName defines the module name
	ModuleName = "xchange"

	// StoreKey defines the primary module store key
	StoreKey = ModuleName

	// MemStoreKey defines the in-memory store key
	MemStoreKey = "mem_xchange"
)

var (
	ParamsKey = []byte("p_xchange")
)

func KeyPrefix(p string) []byte {
	return []byte(p)
}
