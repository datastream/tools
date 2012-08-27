package main

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base32"
	"log"
	"os"
	"time"
)

const (
	SHA1_DIGEST_LENGTH        = 20
	BITS_PER_BASE32_CHAR      = 5
	SCRATCHCODES              = 5
	SCRATCHCODE_LENGTH        = 8
	BYTES_PER_SCRATCHCODE     = 4
	SECRET_BITS               = 8
	VERIFICATION_CODE_MODULUS = 1000 * 1000
)

func generatecode() string {
	return base32.StdEncoding.EncodeToString(get_random())
}

func get_random() []byte {
	fd, err := os.Open("/dev/urandom")
	defer fd.Close()
	if err != nil {
		return nil
	}
	buf := make([]byte, 10)
	_, _ = fd.Read(buf)
	return buf
}

func int_to_bytes(num int64) []byte {
	var val [8]byte
	i := 8
	for {
		i--
		if i < 0 {
			break
		}
		val[i] = byte(num)
		num >>= 8
	}
	return val[:]
}

func compute_code(secret []byte, value int64) uint32 {
	val := int_to_bytes(value)
	key := make([]byte, base32.StdEncoding.DecodedLen(len(secret)))
	_, er := base32.StdEncoding.Decode(key, secret)
	if er != nil {
		log.Println("decode 32:", er)
	}

	h := hmac.New(sha1.New, key)
	n, err := h.Write(val)
	if n != len(val) || err != nil {
		log.Println("create hmac_sha1 error:", err)
	}
	hash := h.Sum(nil)

	offset := hash[len(hash)-1] & 0xF
	truncatedHash := uint32(0)
	for i := 0; i < 4; i++ {
		truncatedHash <<= 8
		truncatedHash |= uint32(hash[int(offset)+i])
	}

	truncatedHash &= 0x7FFFFFFF
	truncatedHash %= 1000000
	return truncatedHash
}

func get_timestamp() int64 {
	return time.Now().Unix() / 30
}

func main() {
	log.Println(generatecode())
	log.Println(compute_code([]byte("OR7EDXQO6DMMYVX5"), get_timestamp()))
}
