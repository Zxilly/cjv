//go:build openharmony

package selfsign

import "crypto/sha256"

const pageSize = 4096

// merkleRoot follows the OpenHarmony self-sign format: the signature page has
// an all-zero leaf digest, rather than the digest of a zero-filled page. All
// other file/hash pages are SHA-256 hashed, with the last page zero padded.
// Intermediate hash levels are packed from leaves toward the root; only the
// prefix that fits after the signature payload is retained.
func merkleRoot(data []byte, signatureOffset int) ([sha256.Size]byte, []byte) {
	leaves := make([]byte, 0, ((len(data)+pageSize-1)/pageSize)*sha256.Size)
	for offset := 0; offset < len(data); offset += pageSize {
		var digest [sha256.Size]byte
		if offset != signatureOffset {
			digest = hashPage(data[offset:min(offset+pageSize, len(data))])
		}
		leaves = append(leaves, digest[:]...)
	}
	if len(leaves) == sha256.Size {
		return [sha256.Size]byte(leaves), nil
	}
	var intermediate []byte
	for len(leaves) > pageSize {
		next := make([]byte, 0, ((len(leaves)+pageSize-1)/pageSize)*sha256.Size)
		for offset := 0; offset < len(leaves); offset += pageSize {
			digest := hashPage(leaves[offset:min(offset+pageSize, len(leaves))])
			next = append(next, digest[:]...)
		}
		if len(next) > pageSize {
			remaining := pageSize - payloadSize - len(intermediate)
			intermediate = append(intermediate, next[:min(remaining, len(next))]...)
		}
		leaves = next
	}
	return hashPage(leaves), intermediate
}

func hashPage(data []byte) [sha256.Size]byte {
	var page [pageSize]byte
	copy(page[:], data)
	return sha256.Sum256(page[:])
}
