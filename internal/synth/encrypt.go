package synth

import (
	"crypto/md5"
	"crypto/rc4"
	"encoding/binary"
	"fmt"
)

// encryption implements the PDF standard security handler, revision 3 with
// a 128-bit RC4 key (ISO 32000-1, 7.6.3), enough to produce protected test
// documents. RC4 and MD5 are what the format prescribes, not a choice: this
// protects nothing real and exists only to test how readers handle
// passwords.
type encryption struct {
	key  []byte
	o, u []byte
	id   []byte
	p    int32
}

var padding = []byte{
	0x28, 0xbf, 0x4e, 0x5e, 0x4e, 0x75, 0x8a, 0x41, 0x64, 0x00, 0x4e, 0x56, 0xff, 0xfa, 0x01, 0x08,
	0x2e, 0x2e, 0x00, 0xb6, 0xd0, 0x68, 0x3e, 0x80, 0x2f, 0x0c, 0xa9, 0xfe, 0x64, 0x53, 0x69, 0x7a,
}

func pad(password string) []byte {
	b := append([]byte(password), padding...)
	return b[:32]
}

func rc4XOR(key, data []byte) []byte {
	c, err := rc4.NewCipher(key)
	if err != nil {
		panic(err)
	}
	out := make([]byte, len(data))
	c.XORKeyStream(out, data)
	return out
}

// rounds applies the 19 extra RC4 passes of revision 3, each with the key
// XORed with the pass number.
func rounds(key, data []byte) []byte {
	for i := 1; i <= 19; i++ {
		k := make([]byte, len(key))
		for j := range key {
			k[j] = key[j] ^ byte(i)
		}
		data = rc4XOR(k, data)
	}
	return data
}

func md5Rehash(sum [16]byte) []byte {
	for range 50 {
		sum = md5.Sum(sum[:])
	}
	return sum[:]
}

func newEncryption(password string) *encryption {
	e := &encryption{p: -4, id: []byte("cedolo-synthetic")}
	// Owner entry (algorithm 3), with the user password as owner password.
	ownerKey := md5Rehash(md5.Sum(pad(password)))
	e.o = rounds(ownerKey, rc4XOR(ownerKey, pad(password)))
	// File key (algorithm 2).
	h := md5.New()
	h.Write(pad(password))
	h.Write(e.o)
	binary.Write(h, binary.LittleEndian, e.p)
	h.Write(e.id)
	var sum [16]byte
	copy(sum[:], h.Sum(nil))
	e.key = md5Rehash(sum)
	// User entry (algorithm 5).
	hu := md5.Sum(append(append([]byte(nil), padding...), e.id...))
	e.u = append(rounds(e.key, rc4XOR(e.key, hu[:])), make([]byte, 16)...)
	return e
}

// object encrypts the data of object number n (generation 0).
func (e *encryption) object(n int, data []byte) []byte {
	k := append(append([]byte(nil), e.key...), byte(n), byte(n>>8), byte(n>>16), 0, 0)
	sum := md5.Sum(k)
	return rc4XOR(sum[:min(len(e.key)+5, 16)], data)
}

func (e *encryption) dict() string {
	return fmt.Sprintf("<< /Filter /Standard /V 2 /R 3 /Length 128 /O <%x> /U <%x> /P %d >>", e.o, e.u, e.p)
}
