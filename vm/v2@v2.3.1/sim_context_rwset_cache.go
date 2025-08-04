package vm

import "chainmaker.org/chainmaker/pb-go/v2/common"

//rw set
type rwSet struct {
	txReadKeyMap  map[string]*common.TxRead  // txReadKeyMap mapped key with TxRead
	txWriteKeyMap map[string]*common.TxWrite // txWriteKeyMap mapped key with TxWrite
}

// collectReadKey collect readMap key into slice
func (rw *rwSet) collectReadKey() []string {
	readKeys := make([]string, 0, 8)

	//collect all the keys in txReadKeyMap to readKeys slice
	for readKey := range rw.txReadKeyMap {
		readKeys = append(readKeys, readKey)
	}

	return readKeys
}

// collectWriteKey collect writeMap key into slice
func (rw *rwSet) collectWriteKey() []string {
	writeKeys := make([]string, 0, 8)

	//collect all the keys in txWriteKeyMap to writeKeys slice
	for writeKey := range rw.txWriteKeyMap {
		writeKeys = append(writeKeys, writeKey)
	}

	return writeKeys
}

// mergeTxReadSet merge read map
func mergeTxReadSet(dst, src map[string]*common.TxRead) map[string]*common.TxRead {
	//merge all the read key-value from src to dst
	for key, txRead := range src {
		dst[key] = txRead
	}

	return dst
}

// mergeTxWriteSet merge write map
func mergeTxWriteSet(dst, src map[string]*common.TxWrite) map[string]*common.TxWrite {
	//merge all the write key-value from src to dst
	for key, txWrite := range src {
		dst[key] = txWrite
	}

	return dst
}
