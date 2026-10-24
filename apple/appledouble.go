package apple

import (
	"encoding/binary"
	"errors"
	"fmt"
)

const (
	AppleDoubleMagic   = 0x00051607
	AppleDoubleVersion = 0x00020000

	EntryResourceFork = 2
	EntryFinderInfo   = 9

	appleDoubleHeaderLength = 26
	appleDoubleEntryLength  = 12
)

// NetatalkFiller is the filler Netatalk and Synology DSM write into the
// header of the AppleDouble files they keep a resource fork in.
var NetatalkFiller = [16]byte([]byte("Netatalk        "))

type AppleDoubleEntry struct {
	ID   uint32
	Data []byte
}

// AppleDouble is an AppleDouble v2 file: a header plus a list of entries
// (Finder info, resource fork, ...). Entries keep their order, so a file that
// is read and written back keeps its layout.
type AppleDouble struct {
	Filler  [16]byte
	Entries []AppleDoubleEntry
}

func NewAppleDouble() *AppleDouble {
	return &AppleDouble{Filler: NetatalkFiller}
}

func ParseAppleDouble(data []byte) (*AppleDouble, error) {
	if len(data) < appleDoubleHeaderLength {
		return nil, errors.New("appledouble: file too short")
	}
	if magic := binary.BigEndian.Uint32(data[0:]); magic != AppleDoubleMagic {
		return nil, fmt.Errorf("appledouble: bad magic %08x", magic)
	}
	if version := binary.BigEndian.Uint32(data[4:]); version != AppleDoubleVersion {
		return nil, fmt.Errorf("appledouble: unsupported version %08x", version)
	}

	ad := &AppleDouble{}
	copy(ad.Filler[:], data[8:24])

	count := int(binary.BigEndian.Uint16(data[24:]))
	if len(data) < appleDoubleHeaderLength+count*appleDoubleEntryLength {
		return nil, errors.New("appledouble: entry table truncated")
	}

	for i := range count {
		pos := appleDoubleHeaderLength + i*appleDoubleEntryLength
		id := binary.BigEndian.Uint32(data[pos:])
		offset := uint64(binary.BigEndian.Uint32(data[pos+4:]))
		length := uint64(binary.BigEndian.Uint32(data[pos+8:]))
		if offset+length > uint64(len(data)) {
			return nil, fmt.Errorf("appledouble: entry %d runs past end of file", id)
		}
		ad.Entries = append(ad.Entries, AppleDoubleEntry{
			ID:   id,
			Data: append([]byte{}, data[offset:offset+length]...),
		})
	}

	return ad, nil
}

// Entry returns the data of entry id, or nil when the file has no such entry.
func (ad *AppleDouble) Entry(id uint32) []byte {
	for _, e := range ad.Entries {
		if e.ID == id {
			return e.Data
		}
	}
	return nil
}

func (ad *AppleDouble) HasEntry(id uint32) bool {
	for _, e := range ad.Entries {
		if e.ID == id {
			return true
		}
	}
	return false
}

// SetEntry replaces the data of entry id, or appends the entry when it does
// not exist yet.
func (ad *AppleDouble) SetEntry(id uint32, data []byte) {
	for i := range ad.Entries {
		if ad.Entries[i].ID == id {
			ad.Entries[i].Data = data
			return
		}
	}
	ad.Entries = append(ad.Entries, AppleDoubleEntry{ID: id, Data: data})
}

// Bytes lays the entries out back to back, in order, right after the entry
// table.
func (ad *AppleDouble) Bytes() []byte {
	headerLength := appleDoubleHeaderLength + len(ad.Entries)*appleDoubleEntryLength

	size := headerLength
	for _, e := range ad.Entries {
		size += len(e.Data)
	}

	out := make([]byte, headerLength, size)
	binary.BigEndian.PutUint32(out[0:], AppleDoubleMagic)
	binary.BigEndian.PutUint32(out[4:], AppleDoubleVersion)
	copy(out[8:24], ad.Filler[:])
	binary.BigEndian.PutUint16(out[24:], uint16(len(ad.Entries)))

	offset := headerLength
	for i, e := range ad.Entries {
		pos := appleDoubleHeaderLength + i*appleDoubleEntryLength
		binary.BigEndian.PutUint32(out[pos:], e.ID)
		binary.BigEndian.PutUint32(out[pos+4:], uint32(offset))
		binary.BigEndian.PutUint32(out[pos+8:], uint32(len(e.Data)))
		offset += len(e.Data)
	}

	for _, e := range ad.Entries {
		out = append(out, e.Data...)
	}

	return out
}
