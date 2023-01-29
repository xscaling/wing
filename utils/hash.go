package utils

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"hash"
	"hash/fnv"

	"github.com/davecgh/go-spew/spew"
	"github.com/dgryski/go-farm"
	"k8s.io/apimachinery/pkg/util/rand"
)

// This is a copy from k8s.io/kubernetes/pkg/util/hash
// To avoid import k8s.io/kubernetes which has lots dependencies, so it does
// DeepHashObject writes specified object to hash using the spew library
// which follows pointers and prints actual values of the nested objects
// ensuring the hash does not change when a pointer changes.
func DeepHashObject(hasher hash.Hash, objectToWrite interface{}) {
	hasher.Reset()
	printer := spew.ConfigState{
		Indent:         " ",
		SortKeys:       true,
		DisableMethods: true,
		SpewKeys:       true,
	}
	printer.Fprintf(hasher, "%#v", objectToWrite)
}

func ComputeObjectHash(object interface{}) string {
	hasher := fnv.New32a()
	DeepHashObject(hasher, object)
	return rand.SafeEncodeString(fmt.Sprint(hasher.Sum32()))
}

func IsObjectEqual(x, y interface{}) bool {
	return ComputeObjectHash(x) == ComputeObjectHash(y)
}

func FarmHash(b *bytes.Buffer) string {
	lo, hi := farm.Hash128(b.Bytes())
	x := make([]byte, 16)
	binary.LittleEndian.PutUint64(x, lo)
	binary.LittleEndian.PutUint64(x[8:], hi)
	return hex.EncodeToString(x)
}
