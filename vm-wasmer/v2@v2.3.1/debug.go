/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package vm

import (
	"fmt"
	"strings"

	"chainmaker.org/chainmaker/protocol/v2"
)

// PrintTxReadSet only for debug
func PrintTxReadSet(txSimContext protocol.TxSimContext) {
	txSimCxtImpl, ok := txSimContext.(*txSimContextImpl)
	if !ok {
		return
	}

	txRead, _ := txSimCxtImpl.collectReadSetAndKey()

	//check tx read set for contract or byte code key
	for key, txRead := range txRead {
		if strings.Contains(key, "Contract:") || strings.Contains(key, "ContractByteCode:") {
			//contract or byte code key
			fmt.Printf("[tx read set]: %v ==> {contract_name = '%v', key = '%s', len(val) = '%d'} \n",
				key, txRead.ContractName, txRead.Key, len(txRead.Value))
		} else {
			//normal key
			fmt.Printf("[tx read set]: %v ==> {contract_name = '%v', key = '%s', val = '%s'} \n",
				key,
				txRead.ContractName, txRead.Key, txRead.Value)
		}
	}
}

// PrintTxWriteSet only for debug
func PrintTxWriteSet(txSimContext protocol.TxSimContext) {
	txSimCxtImpl, ok := txSimContext.(*txSimContextImpl)
	if !ok {
		return
	}

	txWrite, _ := txSimCxtImpl.collectReadSetAndKey()

	//check tx write set for contract or byte code key
	for key, txWrite := range txWrite {
		if strings.Contains(key, "Contract:") || strings.Contains(key, "ContractByteCode:") {
			//contract or byte code key
			fmt.Printf("[tx write set]: %v ==>{contract_name = '%v', key = '%s', len(val) = '%d'} \n",
				key, txWrite.ContractName, txWrite.Key, len(txWrite.Value))
		} else {
			//normal key
			fmt.Printf("[tx write set]: %v ==> {contract_name = '%v', key = '%s', val = '%s'} \n",
				key,
				txWrite.ContractName, txWrite.Key, txWrite.Value)
		}
	}
}
