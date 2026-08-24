package routes

// a_bogus signature generator for toutiao.com web APIs, ported from the
// reference JavaScript implementation used by RSSHub (see lib/routes/toutiao/a-bogus.ts).
// The JS pipeline operates on UTF-16 code points (not bytes), so []int is used
// for strings flowing through RC4/base64 stages. Includes a minimal SM3 hash.

import (
	"math"
	"math/rand"
	"strings"
	"time"
)

func nowMillis() int64 { return time.Now().UnixMilli() }

// hooks for deterministic tests
var (
	abNow     = nowMillis
	abRandInt = rand.Intn
)

// --- SM3 (GB/T 32905-2016) ---

var sm3IV = [8]uint32{
	0x7380166f, 0x4914b2b9, 0x172442d7, 0xda8a0600,
	0xa96f30bc, 0x163138aa, 0xe38dee4d, 0xb0fb0e4e,
}

func rotl32(x uint32, n int) uint32 {
	n %= 32
	return (x << n) | (x >> (32 - n))
}

func sm3P0(x uint32) uint32 { return x ^ rotl32(x, 9) ^ rotl32(x, 17) }
func sm3P1(x uint32) uint32 { return x ^ rotl32(x, 15) ^ rotl32(x, 23) }

func sm3Compress(v *[8]uint32, block []byte) {
	var w [68]uint32
	var w1 [64]uint32
	for i := 0; i < 16; i++ {
		w[i] = uint32(block[4*i])<<24 | uint32(block[4*i+1])<<16 | uint32(block[4*i+2])<<8 | uint32(block[4*i+3])
	}
	for i := 16; i < 68; i++ {
		a := w[i-16] ^ w[i-9] ^ rotl32(w[i-3], 15)
		w[i] = sm3P1(a) ^ rotl32(w[i-13], 7) ^ w[i-6]
	}
	for i := 0; i < 64; i++ {
		w1[i] = w[i] ^ w[i+4]
	}
	a, b, c, d, e, f, g, h := v[0], v[1], v[2], v[3], v[4], v[5], v[6], v[7]
	for j := 0; j < 64; j++ {
		t := uint32(0x79cc4519)
		if j >= 16 {
			t = 0x7a879d8a
		}
		ss1 := rotl32(rotl32(a, 12)+e+rotl32(t, j), 7)
		ss2 := ss1 ^ rotl32(a, 12)
		var tt1, tt2 uint32
		if j < 16 {
			tt1 = (a ^ b ^ c) + d + ss2 + w1[j]
			tt2 = (e ^ f ^ g) + h + ss1 + w[j]
		} else {
			tt1 = ((a & b) | (a & c) | (b & c)) + d + ss2 + w1[j]
			tt2 = ((e & f) | (^e & g)) + h + ss1 + w[j]
		}
		d = c
		c = rotl32(b, 9)
		b = a
		a = tt1
		h = g
		g = rotl32(f, 19)
		f = e
		e = sm3P0(tt2)
	}
	v[0] ^= a
	v[1] ^= b
	v[2] ^= c
	v[3] ^= d
	v[4] ^= e
	v[5] ^= f
	v[6] ^= g
	v[7] ^= h
}

func sm3Sum(msg []byte) [32]byte {
	v := sm3IV
	bitLen := uint64(len(msg)) * 8
	padded := make([]byte, 0, len(msg)+72)
	padded = append(padded, msg...)
	padded = append(padded, 0x80)
	for len(padded)%64 != 56 {
		padded = append(padded, 0)
	}
	for i := 7; i >= 0; i-- {
		padded = append(padded, byte(bitLen>>(8*i)))
	}
	for off := 0; off < len(padded); off += 64 {
		sm3Compress(&v, padded[off:off+64])
	}
	var out [32]byte
	for i, x := range v {
		out[4*i] = byte(x >> 24)
		out[4*i+1] = byte(x >> 16)
		out[4*i+2] = byte(x >> 8)
		out[4*i+3] = byte(x)
	}
	return out
}

// --- RC4 over code points (JS String semantics) ---

func rc4EncryptCP(plaintext []int, key []int) []int {
	s := make([]int, 256)
	for i := range s {
		s[i] = i
	}
	j := 0
	for i := 0; i < 256; i++ {
		j = (j + s[i] + key[i%len(key)]) % 256
		s[i], s[j] = s[j], s[i]
	}
	out := make([]int, len(plaintext))
	i, jj, k := 0, 0, 0 // JS re-declares i and j as 0 for the cipher loop
	for k < len(plaintext) {
		i = (i + 1) % 256
		jj = (jj + s[i]) % 256
		s[i], s[jj] = s[jj], s[i]
		t := (s[i] + s[jj]) % 256
		out[k] = s[t] ^ plaintext[k]
		k++
	}
	return out
}

// --- ByteDance custom base64 over code points ---

const (
	abAlphabetS3 = "ckdp1h4ZKsUB80/Mfvw36XIgR25+WQAlEi7NLboqYTOPuzmFjJnryx9HVGDaStCe"
	abAlphabetS4 = "Dkdpgh2ZmsQB80/MfvV36XI1R45-WUAlEixNLwoqYTOPuzKFjJnry79HbGcaStCe"
)

func abEncode(src []int, alphabet string) string {
	cp := func(i int) int {
		if i >= 0 && i < len(src) {
			return src[i]
		}
		return 0 // JS codePointAt past end yields NaN; NaN<<n collapses to 0
	}
	// JS loop condition `i < (n/3)*4` iterates ceil(n/3*4) times when the
	// product is fractional, emitting one extra char from a zero-padded tail.
	total := int(math.Ceil(float64(len(src)) / 3 * 4))
	var sb strings.Builder
	sb.Grow(total + 1)
	round := 0
	li := cp(0)<<16 | cp(1)<<8 | cp(2)
	for i := 0; i < total; i++ {
		if i/4 != round {
			round++
			r := round * 3
			li = cp(r)<<16 | cp(r+1)<<8 | cp(r+2)
		}
		switch i % 4 {
		case 0:
			sb.WriteByte(alphabet[(li&16515072)>>18])
		case 1:
			sb.WriteByte(alphabet[(li&258048)>>12])
		case 2:
			sb.WriteByte(alphabet[(li&4032)>>6])
		case 3:
			sb.WriteByte(alphabet[li&63])
		}
	}
	return sb.String()
}

func abGenerRandom(random int, opt [2]int) [4]int {
	return [4]int{
		(random & 255 & 170) | (opt[0] & 85),
		(random & 255 & 85) | (opt[0] & 170),
		((random >> 8) & 255 & 170) | (opt[1] & 85),
		((random >> 8) & 255 & 85) | (opt[1] & 170),
	}
}

func abRandomStr() []int {
	out := make([]int, 0, 12)
	for _, opt := range [][2]int{{3, 45}, {1, 0}, {1, 5}} {
		b := abGenerRandom(abRandInt(10000), opt)
		out = append(out, b[:]...)
	}
	return out
}

var abWindowEnvStr = "1536|747|1536|834|0|30|0|0|1536|834|1536|864|1525|747|24|24|Win32"

func abBuildBB(urlSearchParams, userAgent string) []int {
	const suffix = "cus"

	inner := sm3Sum([]byte(urlSearchParams + suffix))
	urlParamsList := sm3Sum(inner[:])

	cusInner := sm3Sum([]byte(suffix))
	cus := sm3Sum(cusInner[:])

	uaCipher := rc4EncryptCP(intSlice([]byte(userAgent)), []int{0x00, 0x01, 0x0e})
	uaDigest := sm3Sum([]byte(abEncode(uaCipher, abAlphabetS3)))

	startTime := abNow()
	endTime := abNow()

	// Fixed upstream configuration constants.
	const (
		pageID = 6241
		aid    = 6383
	)
	args := [3]int{0, 1, 14}

	u16 := uint32(startTime)
	u16e := uint32(endTime)

	b := map[int]int{}
	b[8] = 3
	b[18] = 44
	b[20] = int(u16 >> 24)
	b[21] = int(u16>>16) & 255
	b[22] = int(u16>>8) & 255
	b[23] = int(u16) & 255
	b[24] = int(startTime / 256 / 256 / 256 / 256) // JS float division then |0
	b[25] = int(startTime / 256 / 256 / 256 / 256 / 256)

	b[26] = args[0] >> 24
	b[27] = args[0] >> 16 & 255
	b[28] = args[0] >> 8 & 255
	b[29] = args[0] & 255

	b[30] = args[1] / 256 & 255
	b[31] = args[1] % 256
	b[32] = args[1] >> 24
	b[33] = args[1] >> 16 & 255

	b[34] = args[2] >> 24
	b[35] = args[2] >> 16 & 255
	b[36] = args[2] >> 8 & 255
	b[37] = args[2] & 255

	b[38] = int(urlParamsList[21])
	b[39] = int(urlParamsList[22])
	b[40] = int(cus[21])
	b[41] = int(cus[22])
	b[42] = int(uaDigest[23])
	b[43] = int(uaDigest[24])

	b[44] = int(u16e >> 24)
	b[45] = int(u16e>>16) & 255
	b[46] = int(u16e>>8) & 255
	b[47] = int(u16e) & 255
	b[48] = b[8]
	b[49] = int(endTime / 256 / 256 / 256 / 256)
	b[50] = int(endTime / 256 / 256 / 256 / 256 / 256)

	b[52] = pageID >> 24
	b[53] = pageID >> 16 & 255
	b[54] = pageID >> 8 & 255
	b[55] = pageID & 255

	b[57] = aid & 255
	b[58] = aid >> 8 & 255
	b[59] = aid >> 16 & 255
	b[60] = aid >> 24

	windowEnvList := intSlice([]byte(abWindowEnvStr))
	b[64] = len(windowEnvList)
	b[65] = b[64] & 255
	b[66] = b[64] >> 8

	checksum := 0
	for _, idx := range []int{18, 20, 26, 30, 38, 40, 42, 21, 27, 31, 35, 39, 41, 43, 22, 28, 32, 36, 23, 29, 33, 37, 44, 45, 46, 47, 48, 49, 50, 24, 25, 52, 53, 54, 55, 57, 58, 59, 60, 65, 66, 70, 71} {
		checksum ^= b[idx]
	}

	order := []int{18, 20, 52, 26, 30, 34, 58, 38, 40, 53, 42, 21, 27, 54, 55, 31, 35, 57, 39, 41, 43, 22, 28, 32, 60, 36, 23, 29, 33, 37, 44, 45, 59, 46, 47, 48, 49, 50, 24, 25, 65, 66, 70, 71}
	bb := make([]int, 0, len(order)+len(windowEnvList)+1)
	for _, idx := range order {
		bb = append(bb, b[idx])
	}
	bb = append(bb, windowEnvList...)
	bb = append(bb, checksum)
	return bb
}

func abGenerateRC4BBStr(urlSearchParams, userAgent string) []int {
	return rc4EncryptCP(abBuildBB(urlSearchParams, userAgent), []int{121})
}

func intSlice(bs []byte) []int {
	out := make([]int, len(bs))
	for i, v := range bs {
		out[i] = int(v)
	}
	return out
}

// generateABogus computes the a_bogus anti-crawler token for toutiao web APIs.
func generateABogus(urlSearchParams, userAgent string) string {
	random := abRandomStr()
	body := abGenerateRC4BBStr(urlSearchParams, userAgent)
	result := make([]int, 0, len(random)+len(body))
	result = append(result, random...)
	result = append(result, body...)
	return abEncode(result, abAlphabetS4) + "="
}
