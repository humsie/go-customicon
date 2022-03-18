package apple

import "github.com/humsie/go-customicon/helpers"

const (
	RF_HEADER_LENGTH = 256
	RF_SIZE_BYTES    = 4
)

func NewResourceFork() *ResourceFork {
	return NewResourceForkWithData([]byte{})
}

func NewResourceForkWithData(data []byte) *ResourceFork {
	rf := ResourceFork{}

	rf.setupHeader()
	rf.setupFooter()
	rf.setIconSetData(data)

	return &rf

}

type ResourceFork struct {
	iconSetData []byte

	iconSetLength  int32
	iconSetSize    []byte
	iconSetEndByte []byte
	iconSetSizeW   []byte

	header []byte

	footer []byte
}

func (r *ResourceFork) setIconSetData(data []byte) {

	r.iconSetData = data

	r.iconSetLength = int32(len(data))
	r.iconSetSize = helpers.Int32toBytes(r.iconSetLength)
	r.iconSetEndByte = helpers.Int32toBytes(r.iconSetLength + 260)
	r.iconSetSizeW = helpers.Int32toBytes(r.iconSetLength + 4)

	return
}

func (r *ResourceFork) setupHeader() {
	r.header = make([]byte, RF_HEADER_LENGTH+RF_SIZE_BYTES)
	r.header[2] = 1
	r.header[15] = 50 // No idea yet

	return

}
func (r *ResourceFork) setupFooter() {

	r.footer = []byte{
		0, 0, 1, 0,
		0, 0, 0, 0, // IconSet End Byte
		0, 0, 0, 0, // IconSet size with size prefix
		0, 0, 0, 50,
		0, 0, 0, 0,
		9, 0, 0, 0,
		0, 28, 0, 50,
		0, 0, 105, 99,
		110, 115, 0, 0,
		0, 10, 191, 185,
		255, 255, 0, 0,
		0, 0, 1, 0,
		0, 0,
	}

	return
}

func (r *ResourceFork) Bytes() (out []byte, err error) {

	headBytes := r.header
	footBytes := r.footer

	headBytes[4] = r.iconSetEndByte[0]
	headBytes[5] = r.iconSetEndByte[1]
	headBytes[6] = r.iconSetEndByte[2]
	headBytes[7] = r.iconSetEndByte[3]
	headBytes[8] = r.iconSetSizeW[0]
	headBytes[9] = r.iconSetSizeW[1]
	headBytes[10] = r.iconSetSizeW[2]
	headBytes[11] = r.iconSetSizeW[3]

	headBytes[256] = r.iconSetSize[0]
	headBytes[257] = r.iconSetSize[1]
	headBytes[258] = r.iconSetSize[2]
	headBytes[259] = r.iconSetSize[3]

	footBytes[4] = r.iconSetEndByte[0]
	footBytes[5] = r.iconSetEndByte[1]
	footBytes[6] = r.iconSetEndByte[2]
	footBytes[7] = r.iconSetEndByte[3]
	footBytes[8] = r.iconSetSizeW[0]
	footBytes[9] = r.iconSetSizeW[1]
	footBytes[10] = r.iconSetSizeW[2]
	footBytes[11] = r.iconSetSizeW[3]

	out = append(append(headBytes, r.iconSetData...), footBytes...)
	return

}
