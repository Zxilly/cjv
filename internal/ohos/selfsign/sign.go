//go:build openharmony || hmos_sign

// Package selfsign implements the OpenHarmony developer self-sign format.
// The format and hash layout follow hqzing/ohos-selfsign (0BSD; see LICENSE).
// ELF handling is independently implemented with debug/elf and encoding/binary.
// Release tools also use this package to sign target ELFs on the host.
package selfsign

import (
	"bytes"
	"crypto/sha256"
	"debug/elf"
	"encoding/binary"
	"fmt"
)

const (
	descriptorSize = 256
	payloadSize    = 8 + descriptorSize + sha256.Size
	selfSignFlag   = 0x10
)

// descriptor is the 256-byte OpenHarmony fs-verity descriptor on disk.
// SHA-256 uses the first half of the 64-byte root field; the rest is zero.
type descriptor struct {
	Version       uint8
	HashAlgorithm uint8
	LogBlockSize  uint8
	SaltSize      uint8
	SignatureSize uint32
	FileSize      uint64
	RootHash      [64]byte
	Salt          [32]byte
	Flags         uint32
	Reserved      [139]byte
	CSVersion     uint8
}

type signatureInfo struct {
	Type       uint32
	Length     uint32
	Descriptor descriptor
	Signature  [sha256.Size]byte
}

func newDescriptor(size int, root [sha256.Size]byte) descriptor {
	d := descriptor{
		Version: 1, HashAlgorithm: 1, LogBlockSize: 12,
		FileSize: uint64(size), Flags: selfSignFlag, CSVersion: 3,
	}
	copy(d.RootHash[:], root[:])
	return d
}

func descriptorDigest(d descriptor) ([sha256.Size]byte, error) {
	d.SignatureSize = 0
	var encoded [descriptorSize]byte
	if _, err := binary.Encode(encoded[:], binary.LittleEndian, d); err != nil {
		return [sha256.Size]byte{}, err
	}
	return sha256.Sum256(encoded[:]), nil
}

// Sign adds a developer self-signature to an unsigned ELF64 executable/shared
// library. It preserves the input slice, loadable contents and section indices.
// This signature does not authenticate a publisher or grant execution permission.
func Sign(data []byte) ([]byte, error) {
	executable, err := parseExecutable(data)
	if err != nil {
		return nil, err
	}
	out, offset, err := executable.appendSignatureSection(data)
	if err != nil {
		return nil, err
	}
	root, intermediate := merkleRoot(out, offset)
	d := newDescriptor(len(out), root)
	digest, err := descriptorDigest(d)
	if err != nil {
		return nil, err
	}
	d.SignatureSize = sha256.Size
	info := signatureInfo{Type: 1, Length: descriptorSize + sha256.Size, Descriptor: d, Signature: digest}
	if _, err := binary.Encode(out[offset:], binary.LittleEndian, info); err != nil {
		return nil, err
	}
	copy(out[offset+payloadSize:offset+pageSize], intermediate)
	return out, nil
}

// Verify validates this self-sign format, including its file root, descriptor
// digest and stored intermediate hash prefix. It does not validate certificates
// or the device kernel's admission policy.
func Verify(data []byte) error {
	executable, err := parseExecutable(data)
	if err != nil {
		return err
	}
	section := executable.file.Section(sectionName)
	if section == nil || section.Type != elf.SHT_PROGBITS || section.Flags != 0 ||
		section.Size != pageSize || section.Offset%pageSize != 0 {
		return fmt.Errorf("missing or unsupported %s section", sectionName)
	}
	var info signatureInfo
	if _, err := binary.Decode(data[section.Offset:], binary.LittleEndian, &info); err != nil {
		return err
	}
	if info.Type != 1 || info.Length != descriptorSize+sha256.Size {
		return fmt.Errorf("invalid self-sign payload header")
	}
	root, intermediate := merkleRoot(data, int(section.Offset))
	expected := newDescriptor(len(data), root)
	expected.SignatureSize = sha256.Size
	if info.Descriptor != expected {
		return fmt.Errorf("self-sign descriptor or file digest mismatch")
	}
	digest, err := descriptorDigest(expected)
	if err != nil {
		return err
	}
	if info.Signature != digest {
		return fmt.Errorf("self-sign descriptor signature mismatch")
	}
	var tail [pageSize - payloadSize]byte
	copy(tail[:], intermediate)
	if !bytes.Equal(data[section.Offset+payloadSize:section.Offset+pageSize], tail[:]) {
		return fmt.Errorf("self-sign intermediate hashes mismatch")
	}
	return nil
}
