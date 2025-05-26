/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package wasmer

import (
	"chainmaker.org/chainmaker/protocol/v2"
)

// GetStateLen get state length from chain
func (s *WaciInstance) GetStateLen() int32 {
	return s.getStateCore(true)
}

// GetState get state from chain
func (s *WaciInstance) GetState() int32 {
	return s.getStateCore(false)
}

func (s *WaciInstance) getStateCore(isLen bool) int32 {
	data, err := wacsi.GetState(s.RequestBody, s.Sc.Contract.Name, s.Sc.TxSimContext, s.Memory, s.Sc.GetStateCache, isLen)
	s.Sc.GetStateCache = data // reset _data
	if err != nil {
		s.recordMsg(err.Error())
		return protocol.ContractSdkSignalResultFail
	}
	return protocol.ContractSdkSignalResultSuccess
}

// GetBatchStateLen get batch state length from chain
func (s *WaciInstance) GetBatchStateLen() int32 {
	return s.getBatchStateCore(true)
}

// GetBatchState get state from chain
func (s *WaciInstance) GetBatchState() int32 {
	return s.getBatchStateCore(false)
}

func (s *WaciInstance) getBatchStateCore(isLen bool) int32 {
	data, err := wacsi.GetBatchState(s.RequestBody, s.Sc.Contract.Name, s.Sc.TxSimContext, s.Memory, s.Sc.GetStateCache, isLen)
	s.Sc.GetStateCache = data // reset _data
	if err != nil {
		s.recordMsg(err.Error())
		return protocol.ContractSdkSignalResultFail
	}
	return protocol.ContractSdkSignalResultSuccess
}

func (s *WaciInstance) Sha256() int32 {
	_, err := wacsi.Sha256(s.RequestBody, s.Sc.Contract.Name, s.Memory)
	if err != nil {
		s.recordMsg(err.Error())
		return protocol.ContractSdkSignalResultFail
	}
	return protocol.ContractSdkSignalResultSuccess
}

// HistoryKvIterator Select kv statement
func (s *WaciInstance) HistoryKvIterator() int32 {
	err := wacsi.HistoryKvIterator(s.RequestBody, s.Sc.Contract.Name, s.Sc.TxSimContext, s.Memory)
	if err != nil {
		s.recordMsg(err.Error())
		return protocol.ContractSdkSignalResultFail
	}
	return protocol.ContractSdkSignalResultSuccess
}

// HistoryKvIterHasNext to determine whether db has next statement
func (s *WaciInstance) HistoryKvIterHasNext() int32 {
	err := wacsi.HistoryKvIterHasNext(s.RequestBody, s.Sc.TxSimContext, s.Memory)
	if err != nil {
		s.recordMsg(err.Error())
		return protocol.ContractSdkSignalResultFail
	}
	return protocol.ContractSdkSignalResultSuccess
}

// HistoryKvIterNextLen comment at next version
func (s *WaciInstance) HistoryKvIterNextLen() int32 {
	return s.HistoryKvIterNextCore(true)
}

// HistoryKvIterNext to get kv statement
func (s *WaciInstance) HistoryKvIterNext() int32 {
	return s.HistoryKvIterNextCore(false)
}

func (s *WaciInstance) HistoryKvIterNextCore(isLen bool) int32 {
	data, err := wacsi.HistoryKvIterNext(s.RequestBody, s.Sc.TxSimContext,
		s.Memory, s.Sc.GetStateCache, s.Sc.Contract.Name, isLen)
	s.Sc.GetStateCache = data // reset _data
	if err != nil {
		s.recordMsg(err.Error())
		return protocol.ContractSdkSignalResultFail
	}
	return protocol.ContractSdkSignalResultSuccess
}

// KvIteratorClose Close kv statement
func (s *WaciInstance) HistoryKvIterClose() int32 {
	err := wacsi.HistoryKvIterClose(s.RequestBody, s.Sc.Contract.Name, s.Sc.TxSimContext, s.Memory)
	if err != nil {
		s.recordMsg(err.Error())
		return protocol.ContractSdkSignalResultFail
	}
	return protocol.ContractSdkSignalResultSuccess
}

// GetSenderAddressLen get SenderAddress length from chain
func (s *WaciInstance) GetSenderAddressLen() int32 {
	return s.getSenderAddressCore(true)
}

// GetSenderAddress get SenderAddress from chain
func (s *WaciInstance) GetSenderAddress() int32 {
	return s.getSenderAddressCore(false)
}

func (s *WaciInstance) getSenderAddressCore(isLen bool) int32 {
	data, err := wacsi.GetSenderAddress(s.RequestBody, s.Sc.Contract.Name, s.Sc.TxSimContext, s.Memory, s.Sc.SenderAddressCache, isLen)
	s.Sc.SenderAddressCache = data // reset _data
	if err != nil {
		s.recordMsg(err.Error())
		return protocol.ContractSdkSignalResultFail
	}
	return protocol.ContractSdkSignalResultSuccess
}

// PutState put state to chain
func (s *WaciInstance) PutState() int32 {
	err := wacsi.PutState(s.RequestBody, s.Sc.Contract.Name, s.Sc.TxSimContext)
	if err != nil {
		s.recordMsg(err.Error())
		return protocol.ContractSdkSignalResultFail
	}
	return protocol.ContractSdkSignalResultSuccess
}

// DeleteState delete state from chain
func (s *WaciInstance) DeleteState() int32 {
	err := wacsi.DeleteState(s.RequestBody, s.Sc.Contract.Name, s.Sc.TxSimContext)
	if err != nil {
		s.recordMsg(err.Error())
		return protocol.ContractSdkSignalResultFail
	}
	return protocol.ContractSdkSignalResultSuccess
}

// KvIterator Select kv statement
func (s *WaciInstance) KvIterator() int32 {
	err := wacsi.KvIterator(s.RequestBody, s.Sc.Contract.Name, s.Sc.TxSimContext, s.Memory)
	if err != nil {
		s.recordMsg(err.Error())
		return protocol.ContractSdkSignalResultFail
	}
	return protocol.ContractSdkSignalResultSuccess
}

// KvPreIterator comment at next version
func (s *WaciInstance) KvPreIterator() int32 {
	err := wacsi.KvPreIterator(s.RequestBody, s.Sc.Contract.Name, s.Sc.TxSimContext, s.Memory)
	if err != nil {
		s.recordMsg(err.Error())
		return protocol.ContractSdkSignalResultFail
	}
	return protocol.ContractSdkSignalResultSuccess
}

// KvIteratorHasNext to determine whether db has next statement
func (s *WaciInstance) KvIteratorHasNext() int32 {
	err := wacsi.KvIteratorHasNext(s.RequestBody, s.Sc.TxSimContext, s.Memory)
	if err != nil {
		s.recordMsg(err.Error())
		return protocol.ContractSdkSignalResultFail
	}
	return protocol.ContractSdkSignalResultSuccess
}

// KvIteratorNextLen comment at next version
func (s *WaciInstance) KvIteratorNextLen() int32 {
	return s.kvIteratorNextCore(true)
}

// KvIteratorNext to get kv statement
func (s *WaciInstance) KvIteratorNext() int32 {
	return s.kvIteratorNextCore(false)
}

func (s *WaciInstance) kvIteratorNextCore(isLen bool) int32 {
	data, err := wacsi.KvIteratorNext(s.RequestBody, s.Sc.TxSimContext,
		s.Memory, s.Sc.GetStateCache, s.Sc.Contract.Name, isLen)
	s.Sc.GetStateCache = data // reset _data
	if err != nil {
		s.recordMsg(err.Error())
		return protocol.ContractSdkSignalResultFail
	}
	return protocol.ContractSdkSignalResultSuccess
}

// KvIteratorClose Close kv statement
func (s *WaciInstance) KvIteratorClose() int32 {
	err := wacsi.KvIteratorClose(s.RequestBody, s.Sc.Contract.Name, s.Sc.TxSimContext, s.Memory)
	if err != nil {
		s.recordMsg(err.Error())
		return protocol.ContractSdkSignalResultFail
	}
	return protocol.ContractSdkSignalResultSuccess
}
