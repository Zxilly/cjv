//go:build openharmony

package selfsign

import (
	"bytes"
	"debug/elf"
	"encoding/binary"
	"fmt"
)

const sectionName = ".codesign"

type executable struct {
	file   *elf.File
	header elf.Header64
}

func parseExecutable(data []byte) (*executable, error) {
	var header elf.Header64
	if err := binary.Read(bytes.NewReader(data), binary.LittleEndian, &header); err != nil {
		return nil, fmt.Errorf("read ELF header: %w", err)
	}
	// Validate the table before debug/elf allocates its section metadata. The
	// signer deliberately does not implement ELF extended section numbering.
	size := uint64(len(data))
	if header.Shnum == 0 || header.Shnum >= uint16(elf.SHN_LORESERVE) ||
		header.Shstrndx == 0 || header.Shstrndx >= header.Shnum ||
		header.Shentsize < uint16(binary.Size(elf.Section64{})) ||
		header.Shoff > size || uint64(header.Shnum)*uint64(header.Shentsize) > size-header.Shoff {
		return nil, fmt.Errorf("ELF has no supported section header table")
	}
	file, err := elf.NewFile(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("parse ELF: %w", err)
	}
	if file.Class != elf.ELFCLASS64 || file.Data != elf.ELFDATA2LSB ||
		(file.Type != elf.ET_EXEC && file.Type != elf.ET_DYN) {
		return nil, fmt.Errorf("self-signing requires a little-endian ELF64 executable or shared library")
	}
	for _, section := range file.Sections {
		// NOBITS sections describe memory, not a file extent (not even Offset).
		if section.Type != elf.SHT_NOBITS &&
			(section.Offset > size || section.FileSize > size-section.Offset) {
			return nil, fmt.Errorf("ELF section %q lies outside the input file", section.Name)
		}
	}
	for _, program := range file.Progs {
		if program.Off > size || program.Filesz > size-program.Off {
			return nil, fmt.Errorf("ELF segment lies outside the input file")
		}
	}
	return &executable{file: file, header: header}, nil
}

// appendSignatureSection preserves the original contents and section indices.
// debug/elf is read-only; encoding/binary writes its standard ELF64 structures.
func (e *executable) appendSignatureSection(data []byte) ([]byte, int, error) {
	if e.file.Section(sectionName) != nil {
		return nil, 0, fmt.Errorf("ELF already contains %s", sectionName)
	}
	if e.header.Shnum+1 >= uint16(elf.SHN_LORESERVE) {
		return nil, 0, fmt.Errorf("adding a signature would require extended section numbering")
	}
	names := e.file.Sections[e.header.Shstrndx]
	if names.Type != elf.SHT_STRTAB || names.Flags&elf.SHF_COMPRESSED != 0 || names.FileSize > uint64(^uint32(0))-uint64(len(sectionName)+1) {
		return nil, 0, fmt.Errorf("unsupported ELF section name table")
	}
	signOffset := align(uint64(len(data)), pageSize)
	namesOffset := signOffset + pageSize
	namesSize := names.FileSize + uint64(len(sectionName)+1)
	tableOffset := align(namesOffset+namesSize, 8)
	newSize := tableOffset + uint64(e.header.Shnum+1)*uint64(e.header.Shentsize)
	if newSize > uint64(int(^uint(0)>>1)) {
		return nil, 0, fmt.Errorf("signed ELF is too large")
	}
	out := make([]byte, int(newSize))
	copy(out, data)
	copy(out[namesOffset:], data[names.Offset:names.Offset+names.FileSize])
	copy(out[namesOffset+names.FileSize:], sectionName+"\x00")
	tableSize := uint64(e.header.Shnum) * uint64(e.header.Shentsize)
	copy(out[tableOffset:], data[e.header.Shoff:e.header.Shoff+tableSize])

	// Preserve any trailing extension bytes in existing section entries.
	nameEntryOffset := tableOffset + uint64(e.header.Shstrndx)*uint64(e.header.Shentsize)
	var nameEntry elf.Section64
	if _, err := binary.Decode(out[nameEntryOffset:], binary.LittleEndian, &nameEntry); err != nil {
		return nil, 0, err
	}
	nameEntry.Off, nameEntry.Size = namesOffset, namesSize
	if _, err := binary.Encode(out[nameEntryOffset:], binary.LittleEndian, nameEntry); err != nil {
		return nil, 0, err
	}
	entry := elf.Section64{
		Name: uint32(names.FileSize), Type: uint32(elf.SHT_PROGBITS),
		Off: signOffset, Size: pageSize, Addralign: pageSize,
	}
	if _, err := binary.Encode(out[tableOffset+tableSize:], binary.LittleEndian, entry); err != nil {
		return nil, 0, err
	}
	header := e.header
	header.Shoff, header.Shnum = tableOffset, header.Shnum+1
	if _, err := binary.Encode(out, binary.LittleEndian, header); err != nil {
		return nil, 0, err
	}
	return out, int(signOffset), nil
}

func align(value, alignment uint64) uint64 {
	return (value + alignment - 1) &^ (alignment - 1)
}
