//go:build ignore

package checkout

// Fixture for example/demo.sh. Not compiled: //go:build ignore keeps
// `go test ./...` off this path. The three Set calls are the load-bearing
// block the demo records, then rewrites.

type Session struct {
	Token string
	ID    string
	Role  string
}

type kv struct{}

func (kv) Set(k, v string) {}

var store kv

func persist(s Session) {
	store.Set("CHECKOUT_userToken", s.Token)
	store.Set("CHECKOUT_userID", s.ID)
	store.Set("CHECKOUT_role", s.Role)
}
