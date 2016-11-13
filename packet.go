package main

// MySQL constants for various commands that contain SQL queries.
const comQuery uint8 = 0x03
const comStmtPrepare uint8 = 0x16

type MysqlPacket struct {
	Length     uint32
	SequenceId uint8
	Command    uint8
	Statement  []byte
}

func ReadMysqlPacket(data []byte) *MysqlPacket {
	var mp MysqlPacket
	var offset uint64 = 0

	mp.Length, offset = fixedInt3(data, offset)
	mp.SequenceId, offset = fixedInt1(data, offset)
	mp.Command, offset = fixedInt1(data, offset)

	// We only care about packets that contain SQL statements to log. All
	// other packet types are irrelevant.
	if !mp.isQueryCommand() {
		return nil
	}

	// This can only happen if you have a SQL statement that's over 16 MB!
	if mp.isMultiPartPacket() {
		// FIXME store the payload and return nil
		return nil
	}

	var end uint64 = offset + uint64(mp.Length) - 1
	mp.Statement = data[offset:end]
	return &mp
}

func (mp *MysqlPacket) isMultiPartPacket() bool {
	return mp.Length == 0xFFFFFF
}

func (mp *MysqlPacket) isQueryCommand() bool {
	return mp.Command == comQuery || mp.Command == comStmtPrepare
}

// Fixed-length integers are just ordinary little-endian integers of the
// given number of bytes. Thankfully, for this purpose, we don't need to
// worry about variable-length integers or strings or anything like that.

func fixedInt1(data []byte, offset uint64) (uint8, uint64) {
	return data[offset], (offset + 1)
}

func fixedInt3(data []byte, offset uint64) (uint32, uint64) {
	var fixedInt uint32 = uint32(data[offset+2]<<16) | uint32(data[offset+1]<<8) | uint32(data[offset])
	return fixedInt, (offset + 3)
}
